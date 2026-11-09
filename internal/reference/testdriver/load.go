//go:build reference

package testdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/measurement"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/client"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/store"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

type loadRoom struct {
	id      string
	players [2]int
	clients [2]*client.Client
	version int
}

func (r *loadRoom) close() {
	for _, c := range r.clients {
		if c != nil {
			c.Close()
		}
	}
}
func (d *Driver) makeRoom(ctx context.Context, index int) (*loadRoom, error) {
	room := &loadRoom{id: fmt.Sprintf("load-%d", index), players: [2]int{1000 + 2*index, 1001 + 2*index}, version: 1}
	st := store.Store{DB: d.DB, Node: "node1"}
	for i, p := range room.players {
		if e := st.EnsurePlayer(ctx, p); e != nil {
			room.close()
			return nil, e
		}
		out, e := d.http(ctx, "POST", d.processes["node1"].base+"/v1/sessions", "", map[string]any{"player": p, "credential": fmt.Sprintf("%s:%d", d.Secret, p)})
		if e != nil {
			room.close()
			return nil, e
		}
		token, _ := out["token"].(string)
		if token == "" {
			room.close()
			return nil, fmt.Errorf("load login failed")
		}
		c, _, e := client.Dial("ws"+d.processes["node1"].base[4:]+"/v1/ws", token, "")
		if e != nil {
			room.close()
			return nil, e
		}
		room.clients[i] = c
	}
	if e := st.Create(ctx, room.id, room.players[0], room.players[1], "recover"); e != nil {
		room.close()
		return nil, e
	}
	return room, nil
}
func (d *Driver) pass(ctx context.Context, r *loadRoom) (map[string]any, error) {
	player := (r.version - 1) % 2
	out, e := r.clients[player].Call(ctx, map[string]any{"type": "action", "match_id": r.id, "command_id": fmt.Sprintf("load-%d", r.version), "expected_version": r.version, "kind": "pass"})
	if e == nil && out["error"] == "" {
		r.version++
	}
	return out, e
}
func (d *Driver) stats(ctx context.Context) (map[string]any, error) {
	return d.http(ctx, "GET", d.processes["node1"].control+"/control", d.Token, nil)
}
func (d *Driver) exercise(ctx context.Context, a map[string]any) (map[string]any, error) {
	switch str(a, "kind", "") {
	case "room_churn":
		before, e := d.stats(ctx)
		if e != nil {
			return nil, e
		}
		count := num(a, "rooms", 200)
		cycles := num(a, "cycles", 5)
		for cycle := 0; cycle < cycles; cycle++ {
			var wg sync.WaitGroup
			sem := make(chan struct{}, 16)
			errs := make(chan error, count)
			for i := 0; i < count; i++ {
				wg.Add(1)
				go func(n int) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					r, e := d.makeRoom(ctx, 10000+cycle*count+n)
					if e != nil {
						errs <- e
						return
					}
					defer r.close()
					out, e := d.pass(ctx, r)
					if e != nil {
						errs <- e
						return
					}
					if out["error"] != "" {
						errs <- fmt.Errorf("churn action: %v", out["error"])
					}
				}(i)
			}
			wg.Wait()
			close(errs)
			for e := range errs {
				if e != nil {
					return nil, e
				}
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(num(a, "settle_seconds", 3)) * time.Second):
		}
		after, e := d.stats(ctx)
		if e != nil {
			return nil, e
		}
		return map[string]any{"live_rooms": after["live_rooms"], "goroutine_delta": num(after, "goroutines", 0) - num(before, "goroutines", 0)}, nil
	case "slow_reader":
		count := num(a, "rooms", 20)
		rooms := make([]*loadRoom, 0, count)
		defer func() {
			for _, r := range rooms {
				r.close()
			}
		}()
		for i := 0; i < count; i++ {
			r, e := d.makeRoom(ctx, 100+i)
			if e != nil {
				return nil, e
			}
			rooms = append(rooms, r)
		}
		h := http.Header{}
		login, e := d.http(ctx, "POST", d.processes["node1"].base+"/v1/sessions", "", map[string]any{"player": rooms[0].players[0], "credential": fmt.Sprintf("%s:%d", d.Secret, rooms[0].players[0])})
		if e != nil {
			return nil, e
		}
		slowToken, _ := login["token"].(string)
		h.Set("Authorization", "Bearer "+slowToken)
		slow, _, e := websocket.DefaultDialer.Dial("ws"+d.processes["node1"].base[4:]+"/v1/ws", h)
		if e != nil {
			return nil, e
		}
		defer slow.Close()
		if tcp, ok := slow.UnderlyingConn().(*net.TCPConn); ok {
			_ = tcp.SetReadBuffer(1024)
		}
		for i := 0; i < 180; i++ {
			_ = slow.SetWriteDeadline(time.Now().Add(time.Second))
			if slow.WriteJSON(map[string]any{"type": "join", "match_id": rooms[0].id, "request_id": fmt.Sprint(i)}) != nil {
				break
			}
		}
		completed, unexpected := 0, 0
		lat := []float64{}
		for i := 0; i < num(a, "commands", 100); i++ {
			r := rooms[1+i%(len(rooms)-1)]
			start := time.Now()
			out, e := d.pass(ctx, r)
			lat = append(lat, float64(time.Since(start).Microseconds())/1000)
			if e != nil || out["error"] != "" {
				unexpected++
			} else {
				completed++
			}
		}
		observed, e := d.stats(ctx)
		if e != nil {
			return nil, e
		}
		sort.Float64s(lat)
		return map[string]any{"healthy_completed": completed, "unexpected_errors": unexpected, "max_queue_depth": observed["max_queue_depth"], "slow_disconnects": observed["slow_disconnects"], "healthy_p99_ms": lat[(len(lat)*99-1)/100], "queue_overflow_policy": "disconnect_or_snapshot_coalesce"}, nil
	case "open_loop":
		return d.openLoop(ctx, a)
	case "tick_budget":
		return d.tickBudget(ctx, a)
	case "ds_drain_and_crash":
		return d.dsDrainAndCrash(ctx, a)
	case "reconciliation":
		return d.reconciliation(ctx, a)
	case "overload_and_recover":
		return d.overload(ctx, a)
	default:
		return nil, fmt.Errorf("exercise %s not implemented", str(a, "kind", ""))
	}
}

func (d *Driver) overload(ctx context.Context, a map[string]any) (map[string]any, error) {
	c, e := d.get(ctx, map[string]any{})
	if e != nil {
		return nil, e
	}
	clients := []*client.Client{}
	for i := 0; i < 160; i++ {
		clients = append(clients, c)
	}
	tx, e := d.DB.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	var id string
	if e = tx.QueryRowContext(ctx, "SELECT id FROM matches WHERE id=? FOR UPDATE", d.match).Scan(&id); e != nil {
		return nil, e
	}
	start := time.Now()
	results := make(chan map[string]any, len(clients))
	failures := make(chan error, len(clients))
	var wg sync.WaitGroup
	for i, c := range clients {
		wg.Add(1)
		go func(i int, c *client.Client) {
			defer wg.Done()
			out, e := c.Call(ctx, map[string]any{"type": "action", "match_id": d.match, "command_id": fmt.Sprintf("burst-%d", i), "expected_version": 1, "kind": "pass"})
			if e != nil {
				failures <- e
			} else {
				results <- out
			}
		}(i, c)
	}
	time.Sleep(300 * time.Millisecond)
	released := time.Now()
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	wg.Wait()
	close(results)
	close(failures)
	for e := range failures {
		if e != nil {
			return nil, e
		}
	}
	rejected := 0
	for out := range results {
		if out["error"] == "overloaded" {
			rejected++
		}
	}
	probe, e := d.action(ctx, map[string]any{"player": float64(1), "command_id": "recovery-probe", "expected_version": float64(2), "kind": "pass"})
	if e != nil {
		return nil, e
	}
	if probe["error"] != "" {
		return nil, fmt.Errorf("recovery probe: %v", probe)
	}
	recovery := float64(time.Since(released).Microseconds()) / 1000
	remaining := time.Duration(num(a, "duration_seconds", 30))*time.Second - time.Since(start)
	if remaining > 0 {
		select {
		case <-time.After(remaining):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	metrics, e := d.stats(ctx)
	if e != nil {
		return nil, e
	}
	var violations int
	e = d.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM matches m WHERE JSON_EXTRACT(m.state,'$.version') != 1+(SELECT COUNT(*) FROM events e WHERE e.match_id=m.id)").Scan(&violations)
	if e != nil {
		return nil, e
	}
	return map[string]any{"rejected": rejected, "max_queue_depth": metrics["max_queue_depth"], "recovery_ms": recovery, "invariant_violations": violations}, nil
}
func (d *Driver) openLoop(ctx context.Context, a map[string]any) (map[string]any, error) {
	count := num(a, "rooms", 100)
	rate := num(a, "rate_per_second", 200)
	seconds := num(a, "duration_seconds", 30)
	rooms := make([]*loadRoom, count)
	defer func() {
		for _, r := range rooms {
			if r != nil {
				r.close()
			}
		}
	}()
	// Admit load before starting the measurement window. Each lane has a queue;
	// schedule time, not actual write time, is the latency origin.
	for i := range rooms {
		r, e := d.makeRoom(ctx, 20000+i)
		if e != nil {
			return nil, e
		}
		rooms[i] = r
	}
	type job struct {
		id int
		at time.Time
	}
	lanes := make([]chan job, count)
	samples := make(chan measurement.Sample, rate*seconds)
	var wg sync.WaitGroup
	start := time.Now()
	for i := range rooms {
		lanes[i] = make(chan job, 128)
		wg.Add(1)
		go func(lane int) {
			defer wg.Done()
			r := rooms[lane]
			generation := 0
			for j := range lanes[lane] {
				outcome := "success"
				if r.version > 40 {
					r.close()
					generation++
					nr, e := d.makeRoom(ctx, 20000+generation*count+lane)
					if e != nil {
						outcome = "error"
					} else {
						r = nr
						rooms[lane] = r
					}
				}
				if outcome == "success" {
					out, e := d.pass(ctx, r)
					if e != nil {
						outcome = "error"
					} else if out["error"] != "" {
						outcome = "rejected"
					}
				}
				samples <- measurement.Sample{ID: fmt.Sprint(j.id), ScheduledMS: float64(j.at.Sub(start).Microseconds()) / 1000, FinishedMS: float64(time.Since(start).Microseconds()) / 1000, Outcome: outcome}
			}
		}(i)
	}
	for i := 0; i < rate*seconds; i++ {
		at := start.Add(time.Duration(i) * time.Second / time.Duration(rate))
		timer := time.NewTimer(time.Until(at))
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
		}
		lanes[i%count] <- job{i, at}
	}
	for _, ch := range lanes {
		close(ch)
	}
	wg.Wait()
	close(samples)
	raw := measurement.Run{Commit: os.Getenv("ARENALAB_COMMIT"), Environment: "WSL Docker isolated MySQL; see run metadata", Variant: "reference-open-loop", Seed: int64(num(a, "seed", 451)), WindowMS: float64(seconds * 1000), Expected: rate * seconds}
	if raw.Commit == "" {
		raw.Commit = "working-tree-uncommitted"
	}
	for s := range samples {
		raw.Samples = append(raw.Samples, s)
	}
	summary, e := measurement.Summarize(raw)
	if e != nil {
		return nil, e
	}
	f, e := os.CreateTemp(d.Artifacts, "open-loop-*.json")
	if e != nil {
		return nil, e
	}
	e = json.NewEncoder(f).Encode(raw)
	_ = f.Close()
	if e != nil {
		return nil, e
	}
	var violations int
	e = d.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM matches m WHERE JSON_EXTRACT(m.state,'$.version') != 1+(SELECT COUNT(*) FROM events e WHERE e.match_id=m.id)").Scan(&violations)
	if e != nil {
		return nil, e
	}
	return map[string]any{"offered": summary.Offered, "completed": summary.Succeeded, "unexpected_errors": summary.Errors, "invariant_violations": violations, "p99_ms": summary.AllP99MS, "raw_samples": len(raw.Samples)}, nil
}
