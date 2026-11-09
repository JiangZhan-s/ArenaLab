//go:build reference

// Package realtime deliberately has no database dependency. Tick state is
// volatile; only the terminal result crosses into durable backend settlement.
package realtime

import (
	"math"
	"sort"
	"sync"
	"time"
)

type Input struct {
	Seq int `json:"seq"`
	DX  int `json:"dx"`
	DY  int `json:"dy"`
}
type Snapshot struct {
	Tick      int    `json:"tick"`
	X         [2]int `json:"x"`
	Y         [2]int `json:"y"`
	Processed [2]int `json:"processed"`
	Finished  bool   `json:"finished"`
	Winner    int    `json:"winner"`
}
type Game struct {
	mu           sync.Mutex
	state        Snapshot
	accepted     [2]int
	queue        [2][]Input
	durations    []float64
	maxQueue     int
	maxLag       time.Duration
	droppedTicks int
}

func New() *Game { return &Game{state: Snapshot{X: [2]int{0, 99}, Y: [2]int{0, 99}, Winner: -1}} }
func (g *Game) Input(player int, in Input) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if player < 0 || player > 1 {
		return "forbidden"
	}
	if g.state.Finished {
		return "finished"
	}
	if in.Seq <= g.accepted[player] {
		return "duplicate_input"
	}
	if in.DX < -1 || in.DX > 1 || in.DY < -1 || in.DY > 1 || abs(in.DX)+abs(in.DY) > 1 {
		return "invalid_input"
	}
	if in.Seq > g.accepted[player]+128 || len(g.queue[player]) >= 128 {
		return "input_window_exceeded"
	}
	g.accepted[player] = in.Seq
	g.queue[player] = append(g.queue[player], in)
	if len(g.queue[player]) > g.maxQueue {
		g.maxQueue = len(g.queue[player])
	}
	return ""
}
func (g *Game) Step() Snapshot {
	started := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.state.Finished {
		return g.state
	}
	g.state.Tick++
	for p := 0; p < 2; p++ {
		if len(g.queue[p]) > 0 {
			in := g.queue[p][0]
			g.queue[p] = g.queue[p][1:]
			g.state.X[p] = clamp(g.state.X[p] + in.DX)
			g.state.Y[p] = clamp(g.state.Y[p] + in.DY)
			g.state.Processed[p] = in.Seq
		}
	}
	if g.state.Tick == 600 {
		g.state.Finished = true
		d0 := abs(2*g.state.X[0]-99) + abs(2*g.state.Y[0]-99)
		d1 := abs(2*g.state.X[1]-99) + abs(2*g.state.Y[1]-99)
		if d0 < d1 {
			g.state.Winner = 0
		} else if d1 < d0 {
			g.state.Winner = 1
		}
	}
	g.durations = append(g.durations, float64(time.Since(started).Nanoseconds())/1e6)
	return g.state
}
func (g *Game) Snapshot() Snapshot { g.mu.Lock(); defer g.mu.Unlock(); return g.state }
func (g *Game) Stats() map[string]any {
	g.mu.Lock()
	defer g.mu.Unlock()
	d := append([]float64(nil), g.durations...)
	sort.Float64s(d)
	p99 := 0.0
	if len(d) > 0 {
		p99 = d[int(math.Ceil(.99*float64(len(d))))-1]
	}
	return map[string]any{"completed_ticks": g.state.Tick, "tick_p99_ms": p99, "max_pending_inputs": g.maxQueue, "scheduler_max_lag_ms": float64(g.maxLag) / float64(time.Millisecond), "dropped_schedule_ticks": g.droppedTicks}
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func clamp(n int) int {
	if n < 0 {
		return 0
	}
	if n > 99 {
		return 99
	}
	return n
}

// Reconciler is the bot/client-side prediction reference, not server authority.
type Reconciler struct {
	X, Y, LastTick int
	Pending        []Input
	MaxPending     int
}

func (r *Reconciler) Predict(in Input) {
	r.Pending = append(r.Pending, in)
	r.X = clamp(r.X + in.DX)
	r.Y = clamp(r.Y + in.DY)
	if len(r.Pending) > r.MaxPending {
		r.MaxPending = len(r.Pending)
	}
}
func (r *Reconciler) Install(s Snapshot, player int) {
	if s.Tick < r.LastTick {
		return
	}
	r.LastTick = s.Tick
	r.X = s.X[player]
	r.Y = s.Y[player]
	pending := r.Pending[:0]
	for _, in := range r.Pending {
		if in.Seq > s.Processed[player] {
			pending = append(pending, in)
			r.X = clamp(r.X + in.DX)
			r.Y = clamp(r.Y + in.DY)
		}
	}
	r.Pending = pending
}
