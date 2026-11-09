//go:build reference

package store

import (
	"context"
	"fmt"
)

func (s *Store) ClaimConnection(ctx context.Context, player, epoch int) error {
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var current int
	var used bool
	if e = tx.QueryRowContext(ctx, "SELECT epoch,ws_consumed FROM players WHERE id=? FOR UPDATE", player).Scan(&current, &used); e != nil {
		return e
	}
	if current != epoch {
		return fmt.Errorf("stale_session")
	}
	if used {
		return fmt.Errorf("ticket_used")
	}
	if _, e = tx.ExecContext(ctx, "UPDATE players SET ws_consumed=TRUE WHERE id=?", player); e != nil {
		return e
	}
	return tx.Commit()
}
