//go:build reference

package server

import (
	"github.com/JiangZhan-s/ArenaLab/internal/reference/realtime"
	"time"
)

// EnableRealtime must be called before serving HTTP. Manual mode is for
// deterministic test clocks only; the normal executable uses monotonic time.
func (s *Service) EnableRealtime(manual bool) {
	s.Game = realtime.New()
	s.RealClock.Store(!manual)
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		var schedule realtime.Scheduler
		schedule.Reset(time.Now())
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				if s.RealClock.Load() {
					steps, lag, dropped := schedule.Due(time.Now())
					s.Game.RecordSchedule(lag, dropped)
					for i := 0; i < steps; i++ {
						snap := s.Game.Step()
						if snap.Tick%2 == 0 {
							s.broadcast("realtime", map[string]any{"type": "snapshot", "version": snap.Tick, "state": snap})
						}
					}
				} else {
					schedule.Reset(time.Now())
				}
			}
		}
	}()
}
