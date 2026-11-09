//go:build reference

package testdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/store"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/worker"
	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (d *Driver) settle(ctx context.Context, a map[string]any) (map[string]any, error) {
	args := []string{"-once", "-match", d.match}
	if v, ok := a["winner"].(float64); ok {
		args = append(args, "-winner", strconv.Itoa(int(v)))
	}
	cmd := exec.CommandContext(ctx, filepath.Join(filepath.Dir(d.Binary), "settlement-worker"), args...)
	cmd.Env = append(os.Environ(), "ARENALAB_DSN="+d.dsn)
	b, e := cmd.Output()
	if e != nil {
		return nil, e
	}
	var out map[string]any
	e = json.Unmarshal(b, &out)
	return out, e
}
func (d *Driver) projection(ctx context.Context, a map[string]any) (map[string]any, error) {
	cfg, e := mysql.ParseDSN(d.dsn)
	if e != nil {
		return nil, e
	}
	if !strings.HasPrefix(cfg.DBName, "arenalab_test_") {
		return nil, fmt.Errorf("not an isolated test namespace")
	}
	r := redis.NewClient(&redis.Options{Addr: os.Getenv("ARENALAB_REDIS_ADDR")})
	defer r.Close()
	w := worker.Worker{Store: &store.Store{DB: d.DB}, Redis: r, Namespace: cfg.DBName}
	switch str(a, "action", "") {
	case "rebuild":
		e = w.Rebuild(ctx)
	case "deliver_stale":
		e = w.Project(ctx, num(a, "player", 0), num(a, "score", 0), num(a, "version", 0))
	case "flush_test_namespace":
		var cursor uint64
		for {
			var keys []string
			keys, cursor, e = r.Scan(ctx, cursor, cfg.DBName+":*", 100).Result()
			if e != nil {
				break
			}
			if len(keys) > 0 {
				if e = r.Del(ctx, keys...).Err(); e != nil {
					break
				}
			}
			if cursor == 0 {
				break
			}
		}
	default:
		return nil, fmt.Errorf("unknown projection operation")
	}
	if e != nil {
		return nil, e
	}
	out := map[string]any{"armed": true}
	if str(a, "action", "") == "flush_test_namespace" {
		return out, nil
	}
	for i := 0; i < 2; i++ {
		score, e := r.HGet(ctx, cfg.DBName+":{s1}:player:"+strconv.Itoa(i), "score").Int()
		if e != nil {
			return nil, e
		}
		out["p"+strconv.Itoa(i)+"_score"] = score
	}
	return out, nil
}
func (d *Driver) finishP0(ctx context.Context) error {
	// A legitimate path from opposite corners: four inward moves, approach, hits.
	moves := []struct {
		p, dx, dy int
		kind      string
	}{{0, 1, 0, "move"}, {1, -1, 0, "move"}, {0, 1, 0, "move"}, {1, 0, -1, "move"}, {0, 0, 1, "move"}, {1, 0, -1, "move"}, {0, 0, 1, "move"}, {1, 0, 0, "pass"}, {0, 0, 0, "attack"}, {1, 0, 0, "pass"}, {0, 0, 0, "attack"}, {1, 0, 0, "pass"}, {0, 0, 0, "attack"}}
	for i, m := range moves {
		a := map[string]any{"player": float64(m.p), "command_id": fmt.Sprintf("seed-%d", i), "expected_version": float64(i + 1), "kind": m.kind, "dx": float64(m.dx), "dy": float64(m.dy)}
		out, e := d.action(ctx, a)
		if e != nil {
			return e
		}
		if out["error"] != "" {
			return fmt.Errorf("seed action: %v", out)
		}
	}
	return nil
}
func (d *Driver) journey(ctx context.Context, a map[string]any) (map[string]any, error) {
	if str(a, "transport", "") != "websocket" {
		return d.dedicatedJourney(ctx, a)
	}
	for i := 0; i < 2; i++ {
		c, e := d.get(ctx, map[string]any{"player": float64(i)})
		if e != nil {
			return nil, e
		}
		if _, e = d.http(ctx, "POST", d.processes["node1"].base+"/v1/tickets", c.Token, map[string]any{"request_id": fmt.Sprintf("journey-%d", i)}); e != nil {
			return nil, e
		}
	}
	if _, e := d.Execute(ctx, "await", map[string]any{"condition": "assigned", "timeout_ms": float64(3000)}); e != nil {
		return nil, e
	}
	for i := 0; i < 2; i++ {
		c, e := d.get(ctx, map[string]any{"player": float64(i)})
		if e != nil {
			return nil, e
		}
		for _, typ := range []string{"join", "ready"} {
			out, e := c.Call(ctx, map[string]any{"type": typ, "match_id": d.match})
			if e != nil {
				return nil, e
			}
			if out["error"] != "" {
				return nil, fmt.Errorf("%s rejected: %v", typ, out)
			}
		}
	}
	var last map[string]any
	for i := 0; i < 40; i++ {
		if i == num(a, "reconnect_after", 10) {
			d.mu.Lock()
			d.defaults[0].Close()
			delete(d.defaults, 0)
			d.mu.Unlock()
			c, e := d.get(ctx, map[string]any{})
			if e != nil {
				return nil, e
			}
			if _, e = c.Call(ctx, map[string]any{"type": "resume", "match_id": d.match}); e != nil {
				return nil, e
			}
		}
		var e error
		last, e = d.action(ctx, map[string]any{"player": float64(i % 2), "command_id": fmt.Sprintf("journey-%d", i), "expected_version": float64(i + 1), "kind": "pass"})
		if e != nil {
			return nil, e
		}
		if last["error"] != "" {
			return nil, fmt.Errorf("journey action: %v", last)
		}
	}
	if out, e := d.settle(ctx, map[string]any{}); e != nil {
		return nil, e
	} else if out["error"] != "" {
		return nil, fmt.Errorf("settlement: %v", out)
	}
	versions := []int{}
	deadline := time.Now().Add(2 * time.Second)
	for i := 0; i < 2; i++ {
		c, e := d.get(ctx, map[string]any{"player": float64(i)})
		if e != nil {
			return nil, e
		}
		for time.Now().Before(deadline) {
			snap := c.Snapshot()
			if num(snap, "version", 0) == 41 {
				break
			}
			time.Sleep(time.Millisecond)
		}
		versions = append(versions, num(c.Snapshot(), "version", 0))
	}
	out, e := d.observe(ctx, map[string]any{"source": "sql"})
	if e != nil {
		return nil, e
	}
	state, _ := last["state"].(map[string]any)
	out["winner"] = state["winner"]
	out["client_versions"] = versions
	return out, nil
}
