//go:build reference

package worker

import (
	"context"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/store"
	"github.com/redis/go-redis/v9"
	"strconv"
)

// A versioned absolute projection, never INCR followed by a separate marker.
const projection = `
local old = redis.call('HGET', KEYS[1], 'version')
if not old or tonumber(ARGV[1]) > tonumber(old) then
 redis.call('HSET', KEYS[1], 'version', ARGV[1], 'score', ARGV[2])
 redis.call('ZADD', KEYS[2], ARGV[2], ARGV[3])
end
return redis.call('HGET', KEYS[1], 'score')`

type Worker struct {
	Store     *store.Store
	Redis     *redis.Client
	Namespace string
}

func (w *Worker) Project(ctx context.Context, player, score, version int) error {
	// Hash tags keep all keys for this small teaching season in one Redis slot.
	_, e := w.Redis.Eval(ctx, projection, []string{w.Namespace + ":{s1}:player:" + strconv.Itoa(player), w.Namespace + ":{s1}:ranking"}, version, score, player).Result()
	return e
}
func (w *Worker) Rebuild(ctx context.Context) error {
	rows, e := w.Store.DB.QueryContext(ctx, "SELECT player_id,score,version FROM ratings")
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var p, s, v int
		if e = rows.Scan(&p, &s, &v); e != nil {
			return e
		}
		if e = w.Project(ctx, p, s, v); e != nil {
			return e
		}
	}
	return rows.Err()
}
func (w *Worker) Once(ctx context.Context, match string, winner *int) error {
	if e := w.Store.Settle(ctx, match, winner); e != nil {
		return e
	}
	return w.Rebuild(ctx)
}
func (w *Worker) Pending(ctx context.Context) error {
	rows, e := w.Store.DB.QueryContext(ctx, "SELECT match_id FROM results WHERE state='pending'")
	if e != nil {
		return e
	}
	var ids []string
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range ids {
		if e = w.Once(ctx, id, nil); e != nil {
			return fmt.Errorf("settlement %s: %w", id, e)
		}
	}
	// Rebuild also retries projections for transactions committed before Redis failure.
	return w.Rebuild(ctx)
}
