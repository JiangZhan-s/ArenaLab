//go:build reference

package testdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/realtime"
	"math"
	"math/rand"
	"time"
)

func (d *Driver) control(ctx context.Context, a map[string]any) (map[string]any, error) {
	method := "GET"
	if a != nil {
		method = "POST"
	}
	return d.http(ctx, method, d.processes["node1"].control+"/control", d.Token, a)
}
func (d *Driver) rtSnapshot(ctx context.Context) (realtime.Snapshot, error) {
	var snap realtime.Snapshot
	c, e := d.get(ctx, map[string]any{})
	if e != nil {
		return snap, e
	}
	out, e := d.http(ctx, "GET", d.processes["node1"].base+"/v1/realtime", c.Token, nil)
	if e != nil {
		return snap, e
	}
	b, e := json.Marshal(out)
	if e != nil {
		return snap, e
	}
	e = json.Unmarshal(b, &snap)
	return snap, e
}
func (d *Driver) tickBudget(ctx context.Context, a map[string]any) (map[string]any, error) {
	var label string
	var before, after int64
	if e := d.DB.QueryRowContext(ctx, "SHOW GLOBAL STATUS LIKE 'Com_commit'").Scan(&label, &before); e != nil {
		return nil, e
	}
	if _, e := d.control(ctx, map[string]any{"action": "realtime_on"}); e != nil {
		return nil, e
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Duration(num(a, "duration_seconds", 10)) * time.Second):
	}
	out, e := d.control(ctx, map[string]any{"action": "realtime_off"})
	if e != nil {
		return nil, e
	}
	if e = d.DB.QueryRowContext(ctx, "SHOW GLOBAL STATUS LIKE 'Com_commit'").Scan(&label, &after); e != nil {
		return nil, e
	}
	metrics, _ := out["metrics"].(map[string]any)
	if metrics == nil {
		return nil, fmt.Errorf("missing actual tick metrics")
	}
	metrics["tick_sql_commits"] = after - before
	return metrics, nil
}
func (d *Driver) reconciliation(ctx context.Context, a map[string]any) (map[string]any, error) {
	type incoming struct {
		at    time.Time
		input realtime.Input
	}
	type incomingSnapshot struct {
		at time.Time
		s  realtime.Snapshot
	}
	rng := rand.New(rand.NewSource(int64(num(a, "seed", 451))))
	oneway := num(a, "rtt_ms", 100) / 2
	jitter := num(a, "jitter_ms", 20)
	delay := func() time.Duration {
		n := oneway
		if jitter > 0 {
			n += rng.Intn(2*jitter+1) - jitter
		}
		if n < 0 {
			n = 0
		}
		return time.Duration(n) * time.Millisecond
	}
	c, e := d.get(ctx, map[string]any{})
	if e != nil {
		return nil, e
	}
	pred := realtime.Reconciler{}
	inputs := []incoming{}
	snapshots := []incomingSnapshot{}
	duplicateEffects, illegalMoves := 0, 0
	lastDue := time.Now()
	ticks := num(a, "ticks", 200)
	for tick := 0; tick < ticks+5; tick++ {
		began := time.Now()
		if tick < ticks {
			in := realtime.Input{Seq: tick + 1, DX: 1}
			if tick%40 >= 20 {
				in.DX = -1
			}
			pred.Predict(in)
			due := time.Now().Add(delay())
			if due.Before(lastDue) {
				due = lastDue
			}
			lastDue = due
			inputs = append(inputs, incoming{due, in})
		}
		for len(inputs) > 0 && !time.Now().Before(inputs[0].at) {
			in := inputs[0].input
			inputs = inputs[1:]
			out, e := c.Call(ctx, map[string]any{"type": "input", "seq": in.Seq, "dx": in.DX, "dy": in.DY})
			if e != nil {
				return nil, e
			}
			if out["accepted"] != true {
				return nil, fmt.Errorf("valid input rejected: %v", out)
			}
			if in.Seq%20 == 0 {
				out, e = c.Call(ctx, map[string]any{"type": "input", "seq": in.Seq, "dx": in.DX})
				if e != nil {
					return nil, e
				}
				if out["accepted"] == true {
					duplicateEffects++
				}
			}
			if in.Seq%30 == 0 {
				out, e = c.Call(ctx, map[string]any{"type": "input", "seq": in.Seq + 1, "dx": 100})
				if e != nil {
					return nil, e
				}
				if out["accepted"] == true {
					illegalMoves++
				}
			}
		}
		if _, e = d.control(ctx, map[string]any{"action": "ticks", "ticks": 1}); e != nil {
			return nil, e
		}
		snap, e := d.rtSnapshot(ctx)
		if e != nil {
			return nil, e
		}
		snapshots = append(snapshots, incomingSnapshot{time.Now().Add(delay()), snap})
		for len(snapshots) > 0 && !time.Now().Before(snapshots[0].at) {
			pred.Install(snapshots[0].s, 0)
			snapshots = snapshots[1:]
		}
		timer := time.NewTimer(50*time.Millisecond - time.Since(began))
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		}
	}
	final, e := d.rtSnapshot(ctx)
	if e != nil {
		return nil, e
	}
	pred.Install(final, 0)
	if len(inputs) != 0 || len(pred.Pending) != 0 {
		return nil, fmt.Errorf("inputs not drained")
	}
	errPos := math.Hypot(float64(pred.X-final.X[0]), float64(pred.Y-final.Y[0]))
	return map[string]any{"duplicate_effects": duplicateEffects, "illegal_moves": illegalMoves, "final_position_error": errPos, "max_pending_inputs": pred.MaxPending}, nil
}
