//go:build reference

package testdriver

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/client"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/dedicated"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/store"
	"github.com/go-sql-driver/mysql"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Driver struct {
	allocation                                 dedicated.Allocation
	dsTickets                                  map[int]string
	dsClients                                  map[int]*client.Client
	restored                                   *Driver
	backupFile                                 string
	AdminDSN, Binary, Secret, Token, Artifacts string
	mu                                         sync.Mutex
	clients                                    map[string]*client.Client
	defaults                                   map[int]*client.Client
	DB                                         *sql.DB
	dsn, match, profile                        string
	processes                                  map[string]*process
	fault                                      string
}
type process struct {
	cmd           *exec.Cmd
	base, control string
	log           *os.File
	stopped       bool
}

func New(admin, binary, secret, token, artifacts string) *Driver {
	return &Driver{AdminDSN: admin, Binary: binary, Secret: secret, Token: token, Artifacts: artifacts, clients: map[string]*client.Client{}, defaults: map[int]*client.Client{}, processes: map[string]*process{}}
}
func (d *Driver) Close() {
	if d.restored != nil {
		d.restored.Close()
		d.restored = nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range d.dsClients {
		c.Close()
	}
	for _, c := range d.clients {
		c.Close()
	}
	for _, c := range d.defaults {
		c.Close()
	}
	for _, p := range d.processes {
		d.stop(p)
	}
	if d.DB != nil {
		d.DB.Close()
	}
}
func (d *Driver) stop(p *process) {
	if !p.stopped {
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
		p.stopped = true
		p.log.Close()
	}
}
func freePort() (string, error) {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		return "", e
	}
	addr := l.Addr().String()
	e = l.Close()
	return addr, e
}
func (d *Driver) start(node string) (*process, error) {
	addr, e := freePort()
	if e != nil {
		return nil, e
	}
	control, e := freePort()
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(d.Artifacts, 0700); e != nil {
		return nil, e
	}
	f, e := os.CreateTemp(d.Artifacts, node+"-*.log")
	if e != nil {
		return nil, e
	}
	cmd := exec.Command(d.Binary, "-listen", addr, "-node", node)
	cmd.Env = append(os.Environ(), "ARENALAB_DSN="+d.dsn, "ARENALAB_DEV_SECRET="+d.Secret, "ARENALAB_TEST_TOKEN="+d.Token, "ARENALAB_CONTROL_ADDR="+control)
	if d.profile == "dedicated" {
		cmd.Env = append(cmd.Env, "ARENALAB_MODE=dedicated")
	}
	if d.profile == "realtime" {
		cmd.Env = append(cmd.Env, "ARENALAB_MODE=realtime", "ARENALAB_MANUAL_CLOCK=1")
	}
	cmd.Stdout = f
	cmd.Stderr = f
	if e = cmd.Start(); e != nil {
		f.Close()
		return nil, e
	}
	p := &process{cmd: cmd, base: "http://" + addr, control: "http://" + control, log: f}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, e = d.http(context.Background(), "GET", p.base+"/healthz", "", nil); e == nil {
			d.processes[node] = p
			return p, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	d.stop(p)
	return nil, fmt.Errorf("process %s did not become ready; inspect %s", node, f.Name())
}
func (d *Driver) http(ctx context.Context, method, url, token string, body any) (map[string]any, error) {
	var reader io.Reader
	if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			return nil, e
		}
		reader = bytes.NewReader(b)
	}
	req, e := http.NewRequestWithContext(ctx, method, url, reader)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, e := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	var out map[string]any
	if e = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); e != nil {
		return nil, e
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return out, nil
}
func str(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}
func num(m map[string]any, key string, def int) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return def
}
func (d *Driver) reset(ctx context.Context, a map[string]any) (map[string]any, error) {
	out, e := d.resetState(ctx, a)
	if e != nil {
		return nil, e
	}
	if str(a, "seed", "") == "finished_p0" {
		if e = d.finishP0(ctx); e != nil {
			return nil, e
		}
	}
	return out, nil
}
func (d *Driver) resetState(ctx context.Context, a map[string]any) (map[string]any, error) {
	d.Close()
	d.mu.Lock()
	defer d.mu.Unlock()
	d.clients = map[string]*client.Client{}
	d.defaults = map[int]*client.Client{}
	d.processes = map[string]*process{}
	d.fault = ""
	d.dsTickets = map[int]string{}
	d.dsClients = map[int]*client.Client{}
	d.allocation = dedicated.Allocation{}
	var random [8]byte
	if _, e := rand.Read(random[:]); e != nil {
		return nil, e
	}
	name := "arenalab_test_" + hex.EncodeToString(random[:])
	cfg, e := mysql.ParseDSN(d.AdminDSN)
	if e != nil {
		return nil, e
	}
	cfg.DBName = ""
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	admin, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		return nil, e
	}
	defer admin.Close()
	// The identifier consists only of our fixed prefix and random hex. Never use a caller supplied name.
	if _, e = admin.ExecContext(ctx, "CREATE DATABASE `"+name+"`"); e != nil {
		return nil, e
	}
	cfg.DBName = name
	d.dsn = cfg.FormatDSN()
	st, e := store.Open(d.dsn, "node1")
	if e != nil {
		return nil, e
	}
	d.DB = st.DB
	if e = st.Migrate(ctx); e != nil {
		return nil, e
	}
	for _, p := range []int{0, 1} {
		if e = st.EnsurePlayer(ctx, p); e != nil {
			return nil, e
		}
	}
	d.profile = str(a, "profile", "turn")
	seed := str(a, "seed", "running")
	d.match = "m1"
	if d.profile != "turn" && d.profile != "turn_abort" && d.profile != "realtime" && d.profile != "dedicated" {
		return nil, fmt.Errorf("profile %s is not implemented", d.profile)
	}
	if seed != "empty" && d.profile != "realtime" && d.profile != "dedicated" {
		policy := "recover"
		if d.profile == "turn_abort" {
			policy = "abort"
		}
		if e = st.Create(ctx, d.match, 0, 1, policy); e != nil {
			return nil, e
		}
	}
	if _, e = d.start("node1"); e != nil {
		return nil, e
	}
	if d.profile == "dedicated" {
		for i := 2; i < 6; i++ {
			if e = st.EnsurePlayer(ctx, i); e != nil {
				return nil, e
			}
		}
		for _, id := range []string{"ds1", "ds2", "ds3"} {
			if _, e = d.startDS(id); e != nil {
				return nil, e
			}
		}
	}
	// Let ownership initialization finish, observing the actual control endpoint.
	time.Sleep(150 * time.Millisecond)
	return map[string]any{"isolated": true, "state": seed}, nil
}
func (d *Driver) connect(ctx context.Context, a map[string]any) (map[string]any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.connectLocked(ctx, a)
}
func (d *Driver) connectLocked(ctx context.Context, a map[string]any) (map[string]any, error) {
	player := num(a, "player", 0)
	p := d.processes["node1"]
	if p == nil {
		return nil, fmt.Errorf("reset required")
	}
	credential := d.Secret + ":" + strconv.Itoa(player)
	if str(a, "credential", "") == "invalid" {
		credential = "invalid"
	}
	out, e := d.http(ctx, "POST", p.base+"/v1/sessions", "", map[string]any{"player": player, "credential": credential})
	if e != nil {
		return nil, e
	}
	token, _ := out["token"].(string)
	if token == "" {
		return out, nil
	}
	c, resp, e := client.Dial("ws"+strings.TrimPrefix(p.base, "http")+"/v1/ws", token, str(a, "origin", ""))
	if e != nil {
		if resp != nil {
			defer resp.Body.Close()
			var result map[string]any
			if json.NewDecoder(resp.Body).Decode(&result) == nil {
				return result, nil
			}
		}
		return nil, e
	}
	handle := str(a, "handle", "p"+strconv.Itoa(player))
	d.clients[handle] = c
	d.defaults[player] = c
	return map[string]any{"error": "", "epoch": out["epoch"]}, nil
}
func (d *Driver) get(ctx context.Context, a map[string]any) (*client.Client, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if handle := str(a, "handle", ""); handle != "" {
		c := d.clients[handle]
		if c == nil {
			return nil, fmt.Errorf("unknown connection %s", handle)
		}
		return c, nil
	}
	player := num(a, "player", 0)
	c := d.defaults[player]
	if c == nil {
		if _, e := d.connectLocked(ctx, map[string]any{"player": float64(player)}); e != nil {
			return nil, e
		}
		c = d.defaults[player]
	}
	return c, nil
}
func (d *Driver) action(ctx context.Context, a map[string]any) (map[string]any, error) {
	if str(a, "target", "") == "restored" {
		if d.restored == nil {
			return nil, fmt.Errorf("restored instance absent")
		}
		copy := map[string]any{}
		for k, v := range a {
			if k != "target" {
				copy[k] = v
			}
		}
		return d.restored.action(ctx, copy)
	}
	c, e := d.get(ctx, a)
	if e != nil {
		return nil, e
	}
	msg := map[string]any{"type": "action", "match_id": d.match, "command_id": str(a, "command_id", ""), "expected_version": num(a, "expected_version", 1), "kind": str(a, "kind", "pass"), "dx": num(a, "dx", 0), "dy": num(a, "dy", 0)}
	d.mu.Lock()
	fault := d.fault
	d.fault = ""
	p := d.processes["node1"]
	d.mu.Unlock()
	if fault != "" {
		type result struct {
			out map[string]any
			e   error
		}
		ch := make(chan result, 1)
		go func() { o, e := c.Call(ctx, msg); ch <- result{o, e} }()
		deadline := time.Now().Add(4 * time.Second)
		reached := false
		for time.Now().Before(deadline) {
			v, e := d.http(ctx, "GET", p.control+"/control", d.Token, nil)
			if e != nil {
				return nil, e
			}
			if v["reached"] == fault {
				reached = true
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !reached {
			return nil, fmt.Errorf("fault barrier %s not reached", fault)
		}
		d.mu.Lock()
		d.stop(p)
		d.mu.Unlock()
		r := <-ch
		if r.e == nil {
			return nil, fmt.Errorf("killed command unexpectedly ACKed")
		}
		return map[string]any{"error": "outcome_unknown"}, nil
	}
	out, e := c.Call(ctx, msg)
	if e != nil {
		return nil, e
	}
	code, _ := out["error"].(string)
	switch code {
	case "", "stale_version", "not_turn":
		out["classified"] = true
	}
	return out, nil
}
func (d *Driver) observe(ctx context.Context, a map[string]any) (map[string]any, error) {
	source := str(a, "source", "sql")
	if source == "snapshot" {
		v, e := d.control(ctx, nil)
		if e != nil {
			return nil, e
		}
		snap, _ := v["snapshot"].(map[string]any)
		xs, _ := snap["x"].([]any)
		ys, _ := snap["y"].([]any)
		seqs, _ := snap["processed"].([]any)
		if len(xs) != 2 || len(ys) != 2 || len(seqs) != 2 {
			return nil, fmt.Errorf("invalid snapshot")
		}
		return map[string]any{"tick": snap["tick"], "p0_x": xs[0], "p0_y": ys[0], "last_processed_seq": seqs[0]}, nil
	}
	if source == "restored_sql" {
		if d.restored == nil {
			return nil, fmt.Errorf("restored instance absent")
		}
		return d.restored.observe(ctx, map[string]any{"source": "sql"})
	}
	if source == "client" {
		c, e := d.get(ctx, a)
		if e != nil {
			return nil, e
		}
		snap := c.Snapshot()
		state, _ := snap["state"].(map[string]any)
		return map[string]any{"version": snap["version"], "ply": state["ply"]}, nil
	}
	if source == "room" {
		c, e := d.get(ctx, a)
		if e != nil {
			return nil, e
		}
		out, e := d.http(ctx, "GET", d.processes["node1"].base+"/v1/matches/"+d.match, c.Token, nil)
		if e != nil {
			return nil, e
		}
		state, _ := out["state"].(map[string]any)
		return map[string]any{"version": out["version"], "ply": state["ply"]}, nil
	}
	if source != "sql" {
		return nil, fmt.Errorf("unsupported source %s", source)
	}
	out := map[string]any{}
	var b []byte
	var status string
	if e := d.DB.QueryRowContext(ctx, "SELECT state,status FROM matches WHERE id=?", d.match).Scan(&b, &status); e == nil {
		var state map[string]any
		if e = json.Unmarshal(b, &state); e != nil {
			return nil, e
		}
		out["version"] = state["version"]
		out["ply"] = state["ply"]
		out["state"] = status
	} else if e != sql.ErrNoRows {
		return nil, e
	}
	for key, q := range map[string]string{"events": "SELECT COUNT(*) FROM events", "commands": "SELECT COUNT(*) FROM commands", "assignments": "SELECT COUNT(*) FROM matches", "occupied_players": "SELECT COUNT(*) FROM occupancy", "duplicate_occupancies": "SELECT COUNT(*) FROM (SELECT player_id FROM occupancy GROUP BY player_id HAVING COUNT(*)>1) d", "ledger_rows": "SELECT COUNT(*) FROM ledger", "pending_jobs": "SELECT COUNT(*) FROM results WHERE state='pending'"} {
		var n int
		if e := d.DB.QueryRowContext(ctx, q).Scan(&n); e != nil {
			return nil, e
		}
		out[key] = n
	}
	for i := 0; i < 2; i++ {
		var n int
		if e := d.DB.QueryRowContext(ctx, "SELECT COALESCE(SUM(delta),0) FROM ledger WHERE player_id=?", i).Scan(&n); e != nil {
			return nil, e
		}
		out["p"+strconv.Itoa(i)+"_score"] = n
	}
	var allocations int
	if e := d.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM allocations").Scan(&allocations); e != nil {
		return nil, e
	}
	out["allocations"] = allocations
	return out, nil
}
func (d *Driver) Execute(ctx context.Context, op string, a map[string]any) (map[string]any, error) {
	switch op {
	case "reset":
		return d.reset(ctx, a)
	case "connect":
		return d.connect(ctx, a)
	case "action":
		return d.action(ctx, a)
	case "observe":
		return d.observe(ctx, a)
	case "send_frame":
		c, e := d.get(ctx, a)
		if e != nil {
			return nil, e
		}
		_ = c.Oversize(num(a, "bytes", 1048577))
		return map[string]any{"closed": c.WaitClosed(ctx)}, nil
	case "lookup", "resync":
		c, e := d.get(ctx, a)
		if e != nil {
			return nil, e
		}
		kind := "lookup"
		if op == "resync" {
			kind = "resume"
		}
		return c.Call(ctx, map[string]any{"type": kind, "match_id": d.match, "command_id": str(a, "command_id", ""), "last_seen_version": num(a, "last_seen_version", 0)})
	case "fault":
		point := str(a, "point", "")
		if point == "database" {
			return d.http(ctx, "POST", d.processes["node1"].control+"/control", d.Token, map[string]any{"action": "database_delay", "delay_ms": num(a, "delay_ms", 0)})
		}
		if point == "redis" {
			return d.projection(ctx, map[string]any{"action": "flush_test_namespace"})
		}
		if point == "event_retention" {
			_, e := d.DB.ExecContext(ctx, "DELETE FROM events WHERE match_id=? AND version<?", d.match, num(a, "retain_from_version", 1))
			return map[string]any{"armed": e == nil}, e
		}
		if point != "before_commit" && point != "after_commit_before_ack" {
			return nil, fmt.Errorf("unsupported fault %s", point)
		}
		out, e := d.http(ctx, "POST", d.processes["node1"].control+"/control", d.Token, map[string]any{"action": "arm", "point": point})
		if e == nil {
			d.mu.Lock()
			d.fault = point
			d.mu.Unlock()
		}
		return out, e
	case "process":
		action := str(a, "action", "")
		d.mu.Lock()
		defer d.mu.Unlock()
		switch action {
		case "restart":
			for _, c := range d.defaults {
				c.Close()
			}
			d.defaults = map[int]*client.Client{}
			if _, e := d.start("node1"); e != nil {
				return nil, e
			}
			time.Sleep(150 * time.Millisecond)
			return map[string]any{"ready": true}, nil
		case "drain":
			return d.http(ctx, "POST", d.processes["node1"].control+"/control", d.Token, map[string]any{"action": "drain"})
		case "pause_owner":
			if _, e := d.http(ctx, "POST", d.processes["node1"].control+"/control", d.Token, map[string]any{"action": "quiesce"}); e != nil {
				return nil, e
			}
			if e := pause(d.processes["node1"].cmd.Process); e != nil {
				return nil, e
			}
			if _, e := d.start("node2"); e != nil {
				return nil, e
			}
			return map[string]any{"paused": true}, nil
		case "resume_old_owner":
			e := resume(d.processes["node1"].cmd.Process)
			if e == nil {
				_, e = d.http(ctx, "POST", d.processes["node1"].control+"/control", d.Token, map[string]any{"action": "unquiesce"})
			}
			return map[string]any{"resumed": e == nil}, e
		case "kill_owner":
			d.stop(d.processes["node1"])
			if _, e := d.start("node2"); e != nil {
				return nil, e
			}
			return map[string]any{"killed": true}, nil
		case "kill_ds":
			d.stop(d.processes[d.allocation.Instance])
			return map[string]any{"killed": true}, nil
		case "drain_ds":
			return d.dsCall(ctx, str(a, "instance", "ds1"), "/internal/ds/drain", nil)
		default:
			return nil, fmt.Errorf("unsupported process operation %s", action)
		}
	case "enqueue", "cancel":
		c, e := d.get(ctx, a)
		if e != nil {
			return nil, e
		}
		method := "POST"
		if op == "cancel" {
			method = "DELETE"
		}
		return d.http(ctx, method, d.processes["node1"].base+"/v1/tickets", c.Token, map[string]any{"request_id": str(a, "request_id", "")})
	case "await":
		condition := str(a, "condition", "assigned")
		deadline := time.Now().Add(time.Duration(num(a, "timeout_ms", 3000)) * time.Millisecond)
		for time.Now().Before(deadline) {
			if condition == "allocation_aborted" {
				var state string
				if e := d.DB.QueryRowContext(ctx, "SELECT status FROM allocations WHERE match_id=?", d.match).Scan(&state); e != nil {
					return nil, e
				}
				if state == "aborted" {
					return map[string]any{"state": state}, nil
				}
				time.Sleep(20 * time.Millisecond)
				continue
			}
			if condition == "new_owner" || condition == "aborted" {
				var state string
				var epoch int
				e := d.DB.QueryRowContext(ctx, "SELECT status,owner_epoch FROM matches WHERE id=?", d.match).Scan(&state, &epoch)
				if e != nil {
					return nil, e
				}
				if condition == "new_owner" && epoch > 1 {
					return map[string]any{"owner_epoch": epoch}, nil
				}
				if condition == "aborted" && state == "aborted" {
					return map[string]any{"state": state}, nil
				}
				time.Sleep(20 * time.Millisecond)
				continue
			}
			var id string
			e := d.DB.QueryRowContext(ctx, "SELECT match_id FROM tickets WHERE status='assigned' LIMIT 1").Scan(&id)
			if e == nil {
				d.match = id
				return map[string]any{"state": "assigned"}, nil
			}
			if e != sql.ErrNoRows {
				return nil, e
			}
			time.Sleep(20 * time.Millisecond)
		}
		return nil, fmt.Errorf("await deadline exceeded")
	case "settle":
		return d.settle(ctx, a)
	case "backup":
		return d.backup(ctx, a)
	case "allocate":
		if d.profile == "turn" {
			c, e := d.get(ctx, a)
			if e != nil {
				return nil, e
			}
			return d.http(ctx, "POST", d.processes["node1"].base+"/v1/tickets", c.Token, map[string]any{"request_id": "drain-probe"})
		}
		return d.allocate(ctx, a)
	case "ds_join":
		return d.dsJoin(ctx, a)
	case "ds_result":
		return d.dsResult(ctx, a)
	case "projection":
		return d.projection(ctx, a)
	case "journey":
		return d.journey(ctx, a)
	case "exercise":
		return d.exercise(ctx, a)
	case "input":
		c, e := d.get(ctx, a)
		if e != nil {
			return nil, e
		}
		return c.Call(ctx, map[string]any{"type": "input", "seq": num(a, "seq", 0), "dx": num(a, "dx", 0), "dy": num(a, "dy", 0)})
	case "clock":
		v, e := d.control(ctx, map[string]any{"action": "ticks", "ticks": num(a, "advance_ticks", 1)})
		if e != nil {
			return nil, e
		}
		snap, _ := v["snapshot"].(map[string]any)
		return map[string]any{"tick": snap["tick"]}, nil
	case "submit_result":
		c, e := d.get(ctx, a)
		if e != nil {
			return nil, e
		}
		return d.http(ctx, "POST", d.processes["node1"].base+"/v1/results", c.Token, a)
	default:
		return nil, fmt.Errorf("operation %s not implemented", op)
	}
}
func (d *Driver) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/execute" || r.Method != "POST" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+d.Token {
			http.Error(w, "unauthorized", 401)
			return
		}
		var req struct {
			Op   string         `json:"op"`
			Args map[string]any `json:"args"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if e := decoder.Decode(&req); e != nil {
			http.Error(w, e.Error(), 400)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
		defer cancel()
		out, e := d.Execute(ctx, req.Op, req.Args)
		// Independent observations only; expected values never enter this API.
		// Do not persist arguments: they may contain credentials or tickets.
		if logErr := d.record(req.Op, out, e); logErr != nil {
			http.Error(w, logErr.Error(), 500)
			return
		}
		if e != nil {
			http.Error(w, e.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
}
func ResolveBinary(path string) (string, error) { return filepath.Abs(path) }

func (d *Driver) record(op string, observed map[string]any, operationError error) error {
	if err := os.MkdirAll(d.Artifacts, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(d.Artifacts, "observation-*.json")
	if err != nil {
		return err
	}
	defer f.Close()
	message := ""
	if operationError != nil {
		message = operationError.Error()
	}
	return json.NewEncoder(f).Encode(map[string]any{"time": time.Now().UTC(), "operation": op, "observed": observed, "error": message})
}
