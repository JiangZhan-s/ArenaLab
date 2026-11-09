//go:build reference

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/rules"
	"sort"
	"time"
)

// Own renews only the epoch this process previously acquired. A resumed old
// process cannot discover a newer epoch and quietly start writing with it.
func (s *Store) Own(ctx context.Context, known map[string]int) (map[string]int, error) {
	if delay := s.DelayMS.Load(); delay > 0 {
		if _, e := s.DB.ExecContext(ctx, "SELECT SLEEP(?)", float64(delay)/1000); e != nil {
			return nil, e
		}
	}
	rows, e := s.DB.QueryContext(ctx, "SELECT id FROM matches WHERE status='running' AND rules_version='turn-v1'")
	if e != nil {
		return nil, e
	}
	var ids []string
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	owned := map[string]int{}
	for _, id := range ids {
		tx, e := s.DB.BeginTx(ctx, nil)
		if e != nil {
			return nil, e
		}
		m, e := scan(tx.QueryRowContext(ctx, "SELECT "+fields+" FROM matches WHERE id=? FOR UPDATE", id))
		if e != nil {
			tx.Rollback()
			return nil, e
		}
		var now time.Time
		e = tx.QueryRowContext(ctx, "SELECT UTC_TIMESTAMP(6)").Scan(&now)
		if e != nil {
			tx.Rollback()
			return nil, e
		}
		k, seen := known[id]
		allowed := m.Owner == s.Node && (!seen || k == m.Epoch)
		if !allowed && !now.Before(m.Lease) {
			if m.Policy == "abort" {
				_, e = tx.ExecContext(ctx, "UPDATE matches SET status='aborted' WHERE id=?", id)
				if e == nil {
					_, e = tx.ExecContext(ctx, "DELETE FROM occupancy WHERE match_id=?", id)
				}
			} else {
				m.Owner = s.Node
				m.Epoch++
				_, e = tx.ExecContext(ctx, "UPDATE matches SET owner=?,owner_epoch=? WHERE id=?", s.Node, m.Epoch, id)
				allowed = true
			}
		}
		if e == nil && allowed {
			_, e = tx.ExecContext(ctx, "UPDATE matches SET lease_until=TIMESTAMPADD(SECOND,3,UTC_TIMESTAMP(6)) WHERE id=?", id)
			owned[id] = m.Epoch
		}
		if e != nil {
			tx.Rollback()
			return nil, e
		}
		if e = tx.Commit(); e != nil {
			return nil, e
		}
	}
	return owned, nil
}
func (s *Store) Timeout(ctx context.Context, id string, epoch int) (bool, error) {
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return false, e
	}
	defer tx.Rollback()
	m, e := scan(tx.QueryRowContext(ctx, "SELECT "+fields+" FROM matches WHERE id=? FOR UPDATE", id))
	if e != nil {
		return false, e
	}
	var now time.Time
	if e = tx.QueryRowContext(ctx, "SELECT UTC_TIMESTAMP(6)").Scan(&now); e != nil {
		return false, e
	}
	if m.Status != "running" || m.Owner != s.Node || m.Epoch != epoch || !now.Before(m.Lease) || now.Before(m.Deadline) {
		return false, nil
	}
	next, e := rules.Apply(m.State, rules.Action{Player: m.State.Turn, Kind: "pass"})
	if e != nil {
		return false, e
	}
	b, _ := json.Marshal(next)
	status := "running"
	if next.Finished {
		status = "finished"
	}
	if _, e = tx.ExecContext(ctx, "UPDATE matches SET state=?,status=?,deadline=TIMESTAMPADD(SECOND,15,UTC_TIMESTAMP(6)) WHERE id=?", b, status, id); e != nil {
		return false, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO events(match_id,version,state,reason) VALUES(?,?,?,'timeout')", id, next.Version, b); e != nil {
		return false, e
	}
	if next.Finished {
		if _, e = tx.ExecContext(ctx, "INSERT INTO results(match_id,result_id,winner) VALUES(?,?,?)", id, "result-"+id, next.Winner); e != nil {
			return false, e
		}
	}
	e = tx.Commit()
	return e == nil, e
}
func (s *Store) Enqueue(ctx context.Context, player, epoch int, request string) (map[string]any, error) {
	if request == "" || len(request) > 128 {
		return map[string]any{"error": "invalid_request"}, nil
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	var current int
	if e = tx.QueryRowContext(ctx, "SELECT epoch FROM players WHERE id=? FOR UPDATE", player).Scan(&current); e != nil {
		return nil, e
	}
	if current != epoch {
		return map[string]any{"error": "stale_session"}, nil
	}
	var status, match string
	e = tx.QueryRowContext(ctx, "SELECT status,match_id FROM tickets WHERE player_id=? AND request_id=?", player, request).Scan(&status, &match)
	if e == nil {
		return map[string]any{"error": "", "state": status, "match_id": match}, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return nil, e
	}
	var active int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM occupancy WHERE player_id=?", player).Scan(&active); e != nil {
		return nil, e
	}
	if active > 0 {
		return map[string]any{"error": "already_assigned"}, nil
	}
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM tickets WHERE player_id=? AND status='queued'", player).Scan(&active); e != nil {
		return nil, e
	}
	if active > 0 {
		return map[string]any{"error": "already_queued"}, nil
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO tickets(player_id,request_id,status) VALUES(?,?,'queued')", player, request); e != nil {
		return nil, e
	}
	e = tx.Commit()
	return map[string]any{"error": "", "state": "queued"}, e
}
func (s *Store) Cancel(ctx context.Context, player, epoch int) (string, error) {
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	var current int
	if e = tx.QueryRowContext(ctx, "SELECT epoch FROM players WHERE id=? FOR UPDATE", player).Scan(&current); e != nil {
		return "", e
	}
	if current != epoch {
		return "stale_session", nil
	}
	var id int
	var status string
	e = tx.QueryRowContext(ctx, "SELECT id,status FROM tickets WHERE player_id=? ORDER BY id DESC LIMIT 1 FOR UPDATE", player).Scan(&id, &status)
	if e != nil {
		return "", e
	}
	if status == "queued" {
		status = "cancelled"
		if _, e = tx.ExecContext(ctx, "UPDATE tickets SET status='cancelled' WHERE id=?", id); e != nil {
			return "", e
		}
	}
	return status, tx.Commit()
}

// MatchQueued locks players in ID order before locking tickets. Discovery is
// optimistic; every ticket is rechecked after locks, so cancel wins safely.
func (s *Store) MatchQueued(ctx context.Context) error {
	rows, e := s.DB.QueryContext(ctx, "SELECT id,player_id FROM tickets WHERE status='queued' ORDER BY id LIMIT 2")
	if e != nil {
		return e
	}
	type ticket struct{ id, player int }
	var ts []ticket
	for rows.Next() {
		var t ticket
		if e = rows.Scan(&t.id, &t.player); e != nil {
			rows.Close()
			return e
		}
		ts = append(ts, t)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if len(ts) != 2 || ts[0].player == ts[1].player {
		return nil
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	ps := []int{ts[0].player, ts[1].player}
	sort.Ints(ps)
	for _, p := range ps {
		var epoch int
		if e = tx.QueryRowContext(ctx, "SELECT epoch FROM players WHERE id=? FOR UPDATE", p).Scan(&epoch); e != nil {
			return e
		}
	}
	for _, t := range ts {
		var status string
		if e = tx.QueryRowContext(ctx, "SELECT status FROM tickets WHERE id=? FOR UPDATE", t.id).Scan(&status); e != nil {
			return e
		}
		if status != "queued" {
			return nil
		}
		var n int
		if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM occupancy WHERE player_id=?", t.player).Scan(&n); e != nil {
			return e
		}
		if n > 0 {
			return nil
		}
	}
	id := fmt.Sprintf("match-%d-%d", ts[0].id, ts[1].id)
	b, _ := json.Marshal(rules.Initial())
	if _, e = tx.ExecContext(ctx, "INSERT INTO matches(id,p0,p1,state,status,deadline,owner,lease_until) VALUES(?,?,?,?,'waiting_ready',TIMESTAMPADD(SECOND,15,UTC_TIMESTAMP(6)),?,TIMESTAMPADD(SECOND,3,UTC_TIMESTAMP(6)))", id, ts[0].player, ts[1].player, b, s.Node); e != nil {
		return e
	}
	for _, t := range ts {
		if _, e = tx.ExecContext(ctx, "INSERT INTO occupancy(player_id,match_id) VALUES(?,?)", t.player, id); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, "UPDATE tickets SET status='assigned',match_id=? WHERE id=?", id, t.id); e != nil {
			return e
		}
	}
	return tx.Commit()
}
