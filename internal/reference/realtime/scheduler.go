//go:build reference

package realtime

import "time"

// Scheduler is confined to the tick goroutine. Production time.Time values
// retain Go's monotonic clock; tests supply an explicit timeline.
type Scheduler struct{ next time.Time }

func (s *Scheduler) Reset(now time.Time) { s.next = now.Add(50 * time.Millisecond) }
func (s *Scheduler) Due(now time.Time) (steps int, lag time.Duration, dropped int) {
	if s.next.IsZero() {
		s.Reset(now)
		return
	}
	if now.Before(s.next) {
		return
	}
	lag = now.Sub(s.next)
	due := int(lag/(50*time.Millisecond)) + 1
	steps = due
	if steps > 3 {
		steps = 3
		dropped = due - 3
		s.Reset(now)
	} else {
		s.next = s.next.Add(time.Duration(steps) * 50 * time.Millisecond)
	}
	return
}

func (g *Game) RecordSchedule(lag time.Duration, dropped int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if lag > g.maxLag {
		g.maxLag = lag
	}
	g.droppedTicks += dropped
}
