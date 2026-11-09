//go:build reference

package testdriver

import (
	"context"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/dedicated"
	"time"
)

func (d *Driver) dsDrainAndCrash(ctx context.Context, a map[string]any) (map[string]any, error) {
	n := num(a, "instances", 3)
	if n != 3 || num(a, "clients", 6) != 6 {
		return nil, fmt.Errorf("fleet experiment requires three instances/six clients")
	}
	games := []dedicated.Allocation{}
	for game := 0; game < n; game++ {
		out, e := d.allocate(ctx, map[string]any{"player": float64(game * 2), "request_id": fmt.Sprintf("fleet-%d", game)})
		if e != nil {
			return nil, e
		}
		if out["error"] != "" {
			return nil, fmt.Errorf("allocation: %v", out)
		}
		games = append(games, d.allocation)
		for p := game * 2; p < game*2+2; p++ {
			out, e = d.dsJoin(ctx, map[string]any{"player": float64(p)})
			if e != nil {
				return nil, e
			}
			if out["accepted"] != true {
				return nil, fmt.Errorf("join: %v", out)
			}
		}
	}
	if out, e := d.dsCall(ctx, games[0].Instance, "/internal/ds/drain", nil); e != nil {
		return nil, e
	} else if out["draining"] != true {
		return nil, fmt.Errorf("drain failed")
	}
	var before int
	if e := d.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM allocations WHERE instance_id=?", games[0].Instance).Scan(&before); e != nil {
		return nil, e
	}
	d.stop(d.processes[games[1].Instance])
	deadline := time.Now().Add(34 * time.Second)
	finished := 0
	for time.Now().Before(deadline) {
		finished = 0
		for _, index := range []int{0, 2} {
			var state string
			if e := d.DB.QueryRowContext(ctx, "SELECT status FROM matches WHERE id=?", games[index].Match).Scan(&state); e != nil {
				return nil, e
			}
			if state == "finished" {
				finished++
			}
		}
		if finished == 2 {
			break
		}
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	for _, index := range []int{0, 2} {
		d.match = games[index].Match
		out, e := d.settle(ctx, map[string]any{})
		if e != nil {
			return nil, e
		}
		if out["error"] != "" {
			return nil, fmt.Errorf("settle: %v", out)
		}
	}
	out := map[string]any{"unaffected_matches_finished": finished}
	queries := map[string]string{
		"double_allocations":    "SELECT COUNT(*) FROM (SELECT instance_id FROM allocations WHERE status='allocated' GROUP BY instance_id HAVING COUNT(*)>1) x",
		"duplicate_ledger_rows": "SELECT COUNT(*) FROM (SELECT match_id,player_id FROM ledger GROUP BY match_id,player_id HAVING COUNT(*)>1) x",
		"aborted_score_rows":    "SELECT COUNT(*) FROM ledger l JOIN matches m ON l.match_id=m.id WHERE m.status='aborted'",
	}
	for key, q := range queries {
		var count int
		if e := d.DB.QueryRowContext(ctx, q).Scan(&count); e != nil {
			return nil, e
		}
		out[key] = count
	}
	var after int
	if e := d.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM allocations WHERE instance_id=?", games[0].Instance).Scan(&after); e != nil {
		return nil, e
	}
	out["drain_new_allocations"] = after - before
	return out, nil
}
