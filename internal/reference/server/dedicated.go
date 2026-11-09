//go:build reference

package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/dedicated"
	"net/http"
	"net/url"
	"time"
)

func (s *Service) EnableDedicated() {
	s.Dedicated = &dedicated.Control{Store: s.Store, Secret: s.Secret}
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(s.ctx, time.Second)
				_ = s.Dedicated.Expire(ctx)
				cancel()
			}
		}
	}()
}
func (s *Service) dsIdentity(r *http.Request) (string, bool) {
	id := r.Header.Get("X-DS-ID")
	expected := "Bearer " + dedicated.Key(s.Secret, "ds", id)
	return id, id != "" && hmac.Equal([]byte(expected), []byte(r.Header.Get("Authorization")))
}
func (s *Service) dedicatedRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/allocations/current", func(w http.ResponseWriter, r *http.Request) {
		i, e := s.auth(r)
		if e != nil {
			write(w, 401, map[string]any{"error": "unauthorized"})
			return
		}
		var id int
		e = s.Store.DB.QueryRowContext(r.Context(), "SELECT a.id FROM allocations a JOIN matches m ON a.match_id=m.id WHERE a.status='allocated' AND (m.p0=? OR m.p1=?) LIMIT 1", i.Player, i.Player).Scan(&id)
		if e != nil {
			write(w, 404, map[string]any{"error": "not_found"})
			return
		}
		a, e := s.Dedicated.Find(r.Context(), fmt.Sprintf("a%d", id))
		if e != nil {
			write(w, 500, map[string]any{"error": "internal"})
			return
		}
		write(w, 200, a)
	})
	mux.HandleFunc("POST /internal/ds/register", func(w http.ResponseWriter, r *http.Request) {
		id, ok := s.dsIdentity(r)
		if !ok {
			write(w, 403, map[string]any{"error": "forbidden"})
			return
		}
		var v struct {
			Endpoint string `json:"endpoint"`
			Build    string `json:"build"`
			Region   string `json:"region"`
		}
		if decode(w, r, &v) != nil {
			write(w, 400, map[string]any{"error": "invalid_request"})
			return
		}
		u, e := url.Parse(v.Endpoint)
		if e != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" {
			write(w, 400, map[string]any{"error": "invalid_endpoint"})
			return
		}
		if e = s.Dedicated.Register(r.Context(), id, v.Endpoint, v.Build, v.Region); e != nil {
			write(w, 500, map[string]any{"error": "registration_failed"})
			return
		}
		write(w, 200, map[string]any{"ready": true})
	})
	mux.HandleFunc("POST /internal/ds/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		id, ok := s.dsIdentity(r)
		if !ok {
			write(w, 403, map[string]any{"error": "forbidden"})
			return
		}
		if e := s.Dedicated.Heartbeat(r.Context(), id); e != nil {
			write(w, 500, map[string]any{"error": "heartbeat_failed"})
			return
		}
		write(w, 200, map[string]any{"error": ""})
	})
	mux.HandleFunc("POST /internal/ds/drain", func(w http.ResponseWriter, r *http.Request) {
		id, ok := s.dsIdentity(r)
		if !ok {
			write(w, 403, map[string]any{"error": "forbidden"})
			return
		}
		_, e := s.Store.DB.ExecContext(r.Context(), "UPDATE ds_instances SET status='draining' WHERE id=?", id)
		if e != nil {
			write(w, 500, map[string]any{"error": "internal"})
			return
		}
		write(w, 200, map[string]any{"draining": true})
	})
	mux.HandleFunc("POST /v1/allocations", func(w http.ResponseWriter, r *http.Request) {
		i, e := s.auth(r)
		if e != nil {
			write(w, 401, map[string]any{"error": "unauthorized"})
			return
		}
		if s.Drain.Load() {
			write(w, 409, map[string]any{"error": "draining"})
			return
		}
		var v struct {
			Request string `json:"request_id"`
			Build   string `json:"build"`
			Region  string `json:"region"`
		}
		if decode(w, r, &v) != nil {
			write(w, 400, map[string]any{"error": "invalid_request"})
			return
		}
		a, e := s.Dedicated.Allocate(r.Context(), i.Player, i.Epoch, v.Request, v.Build, v.Region)
		if e != nil {
			write(w, 500, map[string]any{"error": "outcome_unknown"})
			return
		}
		if a.Error == "" && a.State == "allocated" {
			b, _ := json.Marshal(a)
			req, _ := http.NewRequestWithContext(r.Context(), "POST", a.Endpoint+"/internal/start", bytes.NewReader(b))
			req.Header.Set("Authorization", "Bearer "+dedicated.Key(s.Secret, "ds", a.Instance))
			resp, e := (&http.Client{Timeout: 2 * time.Second}).Do(req)
			if e != nil {
				a.Error = "outcome_unknown"
			} else {
				resp.Body.Close()
				if resp.StatusCode != 200 {
					a.Error = "outcome_unknown"
				}
			}
		}
		write(w, 200, a)
	})
	mux.HandleFunc("POST /v1/allocations/{id}/ticket", func(w http.ResponseWriter, r *http.Request) {
		i, e := s.auth(r)
		if e != nil {
			write(w, 401, map[string]any{"error": "unauthorized"})
			return
		}
		var epoch int
		if e = s.Store.DB.QueryRowContext(r.Context(), "SELECT epoch FROM players WHERE id=?", i.Player).Scan(&epoch); e != nil || epoch != i.Epoch {
			write(w, 403, map[string]any{"error": "stale_session"})
			return
		}
		ticket, e := s.Dedicated.Ticket(r.Context(), i.Player, r.PathValue("id"))
		if e != nil {
			write(w, 403, map[string]any{"error": "forbidden"})
			return
		}
		write(w, 200, map[string]any{"ticket": ticket})
	})
	mux.HandleFunc("POST /internal/ds/result", func(w http.ResponseWriter, r *http.Request) {
		id, ok := s.dsIdentity(r)
		if !ok {
			write(w, 403, map[string]any{"error": "forbidden"})
			return
		}
		var v struct {
			Allocation string `json:"allocation_id"`
			Generation int    `json:"generation"`
			Winner     int    `json:"winner"`
		}
		if decode(w, r, &v) != nil {
			write(w, 400, map[string]any{"error": "invalid_request"})
			return
		}
		e := s.Dedicated.Result(r.Context(), id, v.Allocation, v.Generation, v.Winner)
		code := ""
		if e != nil {
			code = e.Error()
		}
		write(w, 200, map[string]any{"error": code})
	})
}
