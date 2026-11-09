//go:build reference

package store

import (
	"context"
	"time"
)

// ExpireReady serializes with Ready on the match row. Expiry never awards points.
func (s *Store) ExpireReady(ctx context.Context) error {
	var pending int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM matches WHERE status='waiting_ready' AND deadline<=UTC_TIMESTAMP(6)").Scan(&pending); err != nil {
		return err
	}
	if pending == 0 {
		return nil
	}
	// Discovery is only an optimization. Recheck under row locks below.
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT id FROM matches WHERE status='waiting_ready' AND deadline<=UTC_TIMESTAMP(6) FOR UPDATE")
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err = tx.ExecContext(ctx, "UPDATE matches SET status='aborted' WHERE id=?", id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "DELETE FROM occupancy WHERE match_id=?", id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Ready(ctx context.Context, player, epoch int, id string) (string, error) {
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
	m, e := scan(tx.QueryRowContext(ctx, "SELECT "+fields+" FROM matches WHERE id=? FOR UPDATE", id))
	if e != nil {
		return "", e
	}
	slot := -1
	for i, p := range m.Players {
		if p == player {
			slot = i
		}
	}
	if slot < 0 {
		return "forbidden", nil
	}
	if m.Status != "waiting_ready" {
		return m.Status, nil
	}
	var now time.Time
	if e = tx.QueryRowContext(ctx, "SELECT UTC_TIMESTAMP(6)").Scan(&now); e != nil {
		return "", e
	}
	if !now.Before(m.Deadline) {
		return "ready_expired", nil
	}
	column := "ready0"
	if slot == 1 {
		column = "ready1"
	}
	if _, e = tx.ExecContext(ctx, "UPDATE matches SET "+column+"=TRUE WHERE id=?", id); e != nil {
		return "", e
	}
	var all bool
	if e = tx.QueryRowContext(ctx, "SELECT ready0 AND ready1 FROM matches WHERE id=?", id).Scan(&all); e != nil {
		return "", e
	}
	state := "waiting_ready"
	if all {
		state = "running"
		if _, e = tx.ExecContext(ctx, "UPDATE matches SET status='running',deadline=TIMESTAMPADD(SECOND,15,UTC_TIMESTAMP(6)),lease_until=TIMESTAMPADD(SECOND,3,UTC_TIMESTAMP(6)) WHERE id=?", id); e != nil {
			return "", e
		}
	}
	return state, tx.Commit()
}
