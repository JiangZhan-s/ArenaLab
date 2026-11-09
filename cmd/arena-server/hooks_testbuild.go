//go:build reference && testhooks

package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/server"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

func installHooks(s *server.Service) (func(), error) {
	addr := os.Getenv("ARENALAB_CONTROL_ADDR")
	if addr == "" {
		return func() {}, nil
	}
	host, _, e := net.SplitHostPort(addr)
	if e != nil || host != "127.0.0.1" {
		return nil, fmt.Errorf("test control requires literal loopback")
	}
	token := os.Getenv("ARENALAB_TEST_TOKEN")
	if len(token) < 16 {
		return nil, fmt.Errorf("test token too short")
	}
	var mu sync.Mutex
	armed, reached := "", ""
	stop := make(chan struct{})
	s.Store.Hook = func(point string) {
		mu.Lock()
		hit := armed == point
		if hit {
			armed = ""
			reached = point
		}
		mu.Unlock()
		if hit {
			<-stop
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/control", func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
			http.Error(w, "unauthorized", 401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			var req struct {
				Action  string `json:"action"`
				Point   string `json:"point"`
				Ticks   int    `json:"ticks"`
				DelayMS int    `json:"delay_ms"`
			}
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil {
				http.Error(w, "invalid", 400)
				return
			}
			switch req.Action {
			case "arm":
				if req.Point != "before_commit" && req.Point != "after_commit_before_ack" {
					http.Error(w, "unknown point", 400)
					return
				}
				mu.Lock()
				armed = req.Point
				reached = ""
				mu.Unlock()
			case "drain":
				s.Drain.Store(true)
			case "quiesce":
				s.Quiesced.Store(true)
				s.Maintenance.Lock()
				s.Maintenance.Unlock()
			case "unquiesce":
				s.Quiesced.Store(false)
			case "ticks":
				if s.Game == nil || req.Ticks < 0 || req.Ticks > 600 {
					http.Error(w, "invalid clock", 400)
					return
				}
				for i := 0; i < req.Ticks; i++ {
					s.Game.Step()
				}
			case "realtime_on":
				s.RealClock.Store(true)
			case "realtime_off":
				s.RealClock.Store(false)
			case "database_delay":
				if req.DelayMS < 0 || req.DelayMS > 1000 {
					http.Error(w, "invalid delay", 400)
					return
				}
				s.Store.DelayMS.Store(int64(req.DelayMS))
			default:
				http.Error(w, "unknown operation", 400)
				return
			}
		}
		mu.Lock()
		point := reached
		mu.Unlock()
		out := s.Stats()
		out["reached"] = point
		out["armed"] = true
		out["draining"] = s.Drain.Load()
		out["goroutines"] = runtime.NumGoroutine()
		if s.Game != nil {
			out["snapshot"] = s.Game.Snapshot()
			out["metrics"] = s.Game.Stats()
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return nil, e
	}
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}
	go srv.Serve(l)
	return func() { close(stop); _ = srv.Close() }, nil
}
