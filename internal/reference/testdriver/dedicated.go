//go:build reference

package testdriver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/client"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/dedicated"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"time"
)

func (d *Driver) startDS(id string) (*process, error) {
	addr, e := freePort()
	if e != nil {
		return nil, e
	}
	f, e := os.CreateTemp(d.Artifacts, id+"-*.log")
	if e != nil {
		return nil, e
	}
	cmd := exec.Command(filepath.Join(filepath.Dir(d.Binary), "ds-sim"), "-listen", addr, "-id", id, "-backend", d.processes["node1"].base)
	cmd.Env = append(os.Environ(), "ARENALAB_DS_AUTH="+dedicated.Key(d.Secret, "ds", id), "ARENALAB_DS_TICKET_KEY="+dedicated.Key(d.Secret, "ticket", id), "ARENALAB_TEST_TOKEN="+d.Token)
	cmd.Stdout = f
	cmd.Stderr = f
	if e = cmd.Start(); e != nil {
		f.Close()
		return nil, e
	}
	p := &process{cmd: cmd, base: "http://" + addr, log: f}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var state string
		if e = d.DB.QueryRow("SELECT status FROM ds_instances WHERE id=?", id).Scan(&state); e == nil && state == "ready" {
			d.processes[id] = p
			return p, nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	d.stop(p)
	return nil, fmt.Errorf("DS failed to register: %s", f.Name())
}
func (d *Driver) allocate(ctx context.Context, a map[string]any) (map[string]any, error) {
	c, e := d.get(ctx, a)
	if e != nil {
		return nil, e
	}
	out, e := d.http(ctx, "POST", d.processes["node1"].base+"/v1/allocations", c.Token, map[string]any{"request_id": str(a, "request_id", "alloc1"), "build": str(a, "build", "ds-v1"), "region": str(a, "region", "local")})
	if e != nil {
		return nil, e
	}
	if out["error"] == "" {
		b, _ := json.Marshal(out)
		var allocation dedicated.Allocation
		if e = json.Unmarshal(b, &allocation); e != nil {
			return nil, e
		}
		d.allocation = allocation
		d.match = allocation.Match
	}
	return out, nil
}
func (d *Driver) dsJoin(ctx context.Context, a map[string]any) (map[string]any, error) {
	player := num(a, "player", 0)
	c, e := d.get(ctx, a)
	if e != nil {
		return nil, e
	}
	kind := str(a, "ticket", "valid")
	ticket := ""
	if kind == "replay" {
		ticket = d.dsTickets[player]
	} else {
		out, e := d.http(ctx, "POST", d.processes["node1"].base+"/v1/allocations/"+d.allocation.ID+"/ticket", c.Token, nil)
		if e != nil {
			return nil, e
		}
		ticket, _ = out["ticket"].(string)
		if ticket == "" {
			return nil, fmt.Errorf("ticket rejected: %v", out)
		}
		if kind == "expired" {
			claims, e := dedicated.Verify(dedicated.Key(d.Secret, "ticket", d.allocation.Instance), ticket)
			if e != nil {
				return nil, e
			}
			claims.Expires = time.Now().Add(-time.Second).Unix()
			ticket = dedicated.Sign(dedicated.Key(d.Secret, "ticket", d.allocation.Instance), claims)
		} else {
			d.dsTickets[player] = ticket
		}
	}
	ws, _, e := websocket.DefaultDialer.Dial("ws"+d.allocation.Endpoint[4:]+"/v1/ws", nil)
	if e != nil {
		return nil, e
	}
	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	if e = ws.WriteJSON(map[string]any{"ticket": ticket}); e != nil {
		ws.Close()
		return nil, e
	}
	var out map[string]any
	if e = ws.ReadJSON(&out); e != nil {
		ws.Close()
		return nil, e
	}
	if out["accepted"] != true {
		ws.Close()
		return out, nil
	}
	_ = ws.SetReadDeadline(time.Time{})
	if old := d.dsClients[player]; old != nil {
		old.Close()
	}
	d.dsClients[player] = client.Adopt(ws, "")
	return out, nil
}
func (d *Driver) dsCall(ctx context.Context, id, path string, payload any) (map[string]any, error) {
	b, _ := json.Marshal(payload)
	req, e := http.NewRequestWithContext(ctx, "POST", d.processes["node1"].base+path, bytes.NewReader(b))
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+dedicated.Key(d.Secret, "ds", id))
	req.Header.Set("X-DS-ID", id)
	resp, e := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	var out map[string]any
	e = json.NewDecoder(resp.Body).Decode(&out)
	return out, e
}
func (d *Driver) dsResult(ctx context.Context, a map[string]any) (map[string]any, error) {
	identity := str(a, "identity", "allocated_ds")
	payload := map[string]any{"allocation_id": d.allocation.ID, "generation": num(a, "generation", d.allocation.Generation), "winner": num(a, "winner", 0)}
	var out map[string]any
	var e error
	switch identity {
	case "player":
		c, err := d.get(ctx, map[string]any{})
		if err != nil {
			return nil, err
		}
		out, e = d.http(ctx, "POST", d.processes["node1"].base+"/internal/ds/result", c.Token, payload)
	case "old_ds":
		out, e = d.dsCall(ctx, d.allocation.Instance, "/internal/ds/result", payload)
	case "allocated_ds":
		out, e = d.http(ctx, "POST", d.allocation.Endpoint+"/test/result", d.Token, payload)
	default:
		return nil, fmt.Errorf("unknown DS identity")
	}
	if e != nil {
		return nil, e
	}
	if out["error"] == "" {
		settled, e := d.settle(ctx, map[string]any{})
		if e != nil {
			return nil, e
		}
		if settled["error"] != "" {
			return nil, fmt.Errorf("settlement: %v", settled)
		}
	}
	return out, nil
}
func (d *Driver) dedicatedJourney(ctx context.Context, a map[string]any) (map[string]any, error) {
	if out, e := d.allocate(ctx, map[string]any{"request_id": "journey"}); e != nil {
		return nil, e
	} else if out["error"] != "" {
		return nil, fmt.Errorf("allocation: %v", out)
	}
	for i := 0; i < 2; i++ {
		if out, e := d.dsJoin(ctx, map[string]any{"player": float64(i)}); e != nil {
			return nil, e
		} else if out["accepted"] != true {
			return nil, fmt.Errorf("join: %v", out)
		}
	}
	for seq := 1; seq <= 20; seq++ {
		for i := 0; i < 2; i++ {
			dx := 1
			if i == 1 {
				dx = -1
			}
			if _, e := d.dsClients[i].Call(ctx, map[string]any{"type": "input", "seq": seq, "dx": dx}); e != nil {
				return nil, e
			}
		}
		if seq == 10 {
			if _, e := d.dsJoin(ctx, map[string]any{"player": float64(0)}); e != nil {
				return nil, e
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	deadline := time.Now().Add(33 * time.Second)
	state := ""
	for time.Now().Before(deadline) {
		if e := d.DB.QueryRowContext(ctx, "SELECT status FROM matches WHERE id=?", d.match).Scan(&state); e != nil {
			return nil, e
		}
		if state == "finished" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if state != "finished" {
		return nil, fmt.Errorf("DS did not finish: %s", state)
	}
	if out, e := d.settle(ctx, map[string]any{}); e != nil {
		return nil, e
	} else if out["error"] != "" {
		return nil, fmt.Errorf("settle: %v", out)
	}
	left, right := d.dsClients[0].Snapshot(), d.dsClients[1].Snapshot()
	agreement := num(left, "version", 0) == 600 && num(right, "version", 0) == 600 && reflect.DeepEqual(left["state"], right["state"])
	out, e := d.observe(ctx, map[string]any{"source": "sql"})
	if e != nil {
		return nil, e
	}
	out["client_agreement"] = agreement
	out["runtime"] = "go_ds_sim"
	return out, nil
}
