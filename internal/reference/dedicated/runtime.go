//go:build reference

package dedicated

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/realtime"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"time"
)

type Runtime struct {
	Origin                                 string
	ID, Base, Endpoint, AuthKey, TicketKey string
	mu                                     sync.Mutex
	Allocation                             Allocation
	Game                                   *realtime.Game
	used                                   map[string]bool
	joined                                 map[int]bool
	peers                                  map[*websocket.Conn]chan any
	ctx                                    context.Context
	cancel                                 context.CancelFunc
}

func NewRuntime(id, base, endpoint, auth, ticket string) *Runtime {
	ctx, cancel := context.WithCancel(context.Background())
	return &Runtime{Origin: "http://localhost:8080", ID: id, Base: base, Endpoint: endpoint, AuthKey: auth, TicketKey: ticket, used: map[string]bool{}, joined: map[int]bool{}, peers: map[*websocket.Conn]chan any{}, ctx: ctx, cancel: cancel}
}
func (r *Runtime) Close() {
	r.cancel()
	r.mu.Lock()
	defer r.mu.Unlock()
	for c := range r.peers {
		c.Close()
	}
}
func (r *Runtime) Control(ctx context.Context, path string, v any) (map[string]any, error) {
	b, _ := json.Marshal(v)
	req, e := http.NewRequestWithContext(ctx, "POST", r.Base+path, bytes.NewReader(b))
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+r.AuthKey)
	req.Header.Set("X-DS-ID", r.ID)
	resp, e := (&http.Client{Timeout: time.Second}).Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	var out map[string]any
	e = json.NewDecoder(resp.Body).Decode(&out)
	return out, e
}
func (r *Runtime) Start() error {
	out, e := r.Control(r.ctx, "/internal/ds/register", map[string]any{"endpoint": r.Endpoint, "build": "ds-v1", "region": "local"})
	if e != nil {
		return e
	}
	if out["ready"] != true {
		return fmt.Errorf("registration rejected")
	}
	go r.loop()
	return nil
}
func (r *Runtime) Submit(ctx context.Context, id string, generation, winner int) (map[string]any, error) {
	return r.Control(ctx, "/internal/ds/result", map[string]any{"allocation_id": id, "generation": generation, "winner": winner})
}
func (r *Runtime) loop() {
	go r.controlLoop()
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	var schedule realtime.Scheduler
	schedule.Reset(time.Now())
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-tick.C:
			r.mu.Lock()
			game := r.Game
			started := len(r.joined) == 2
			r.mu.Unlock()
			if game == nil || !started {
				schedule.Reset(time.Now())
				continue
			}
			steps, lag, dropped := schedule.Due(time.Now())
			game.RecordSchedule(lag, dropped)
			for i := 0; i < steps; i++ {
				s := game.Step()
				if s.Tick%2 == 0 {
					r.mu.Lock()
					for c, q := range r.peers {
						select {
						case q <- map[string]any{"type": "snapshot", "version": s.Tick, "state": s}:
						default:
							c.Close()
						}
					}
					r.mu.Unlock()
				}
			}
		}
	}
}
func (r *Runtime) controlLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	durable := false
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			_, _ = r.Control(r.ctx, "/internal/ds/heartbeat", nil)
			r.mu.Lock()
			game, a := r.Game, r.Allocation
			r.mu.Unlock()
			if game != nil && !durable {
				snap := game.Snapshot()
				if snap.Finished {
					out, e := r.Submit(r.ctx, a.ID, a.Generation, snap.Winner)
					if e == nil && out["error"] == "" {
						durable = true
					}
				}
			}
		}
	}
}
func (r *Runtime) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, q *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ready": true})
	})
	mux.HandleFunc("POST /internal/start", func(w http.ResponseWriter, q *http.Request) {
		if !hmac.Equal([]byte(q.Header.Get("Authorization")), []byte("Bearer "+r.AuthKey)) {
			http.Error(w, "forbidden", 403)
			return
		}
		var a Allocation
		if json.NewDecoder(http.MaxBytesReader(w, q.Body, 8192)).Decode(&a) != nil || a.Instance != r.ID {
			http.Error(w, "invalid", 400)
			return
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.Game != nil && r.Allocation.ID != a.ID {
			http.Error(w, "occupied", 409)
			return
		}
		if r.Game == nil {
			r.Game = realtime.New()
			r.Allocation = a
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"error": ""})
	})
	mux.HandleFunc("GET /v1/ws", r.websocket)
	return mux
}
func (r *Runtime) websocket(w http.ResponseWriter, q *http.Request) {
	// Browser clients send the one-use ticket as the first frame, not in the URL.
	origin := q.Header.Get("Origin")
	if origin != "" && origin != r.Origin {
		http.Error(w, "origin_denied", 403)
		return
	}
	u := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	ws, e := u.Upgrade(w, q, nil)
	if e != nil {
		return
	}
	defer ws.Close()
	ws.SetReadLimit(8192)
	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	var hello struct {
		Ticket string `json:"ticket"`
	}
	if ws.ReadJSON(&hello) != nil {
		return
	}
	ticket, e := Verify(r.TicketKey, hello.Ticket)
	code := ""
	if e != nil {
		code = e.Error()
	}
	r.mu.Lock()
	a := r.Allocation
	if code == "" && (ticket.Audience != r.ID || ticket.Allocation != a.ID || ticket.Generation != a.Generation || ticket.Build != a.Build || ticket.Match != a.Match) {
		code = "invalid_ticket"
	}
	if code == "" && r.used[ticket.JTI] {
		code = "ticket_used"
	}
	if code == "" {
		r.used[ticket.JTI] = true
		r.joined[ticket.Player] = true
	}
	game := r.Game
	r.mu.Unlock()
	if e = ws.WriteJSON(map[string]any{"accepted": code == "", "error": code}); e != nil || code != "" {
		return
	}
	if game == nil {
		return
	}
	sends := make(chan any, 64)
	done := make(chan struct{})
	defer close(done)
	r.mu.Lock()
	r.peers[ws] = sends
	r.mu.Unlock()
	defer func() { r.mu.Lock(); delete(r.peers, ws); r.mu.Unlock() }()
	go func() {
		defer ws.Close()
		for {
			select {
			case <-done:
				return
			case v := <-sends:
				_ = ws.SetWriteDeadline(time.Now().Add(2 * time.Second))
				if ws.WriteJSON(v) != nil {
					return
				}
			}
		}
	}()
	for {
		_ = ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		var in struct {
			Type      string `json:"type"`
			RequestID string `json:"request_id"`
			realtime.Input
		}
		if ws.ReadJSON(&in) != nil {
			return
		}
		code := game.Input(ticket.Player%2, in.Input)
		out := map[string]any{"type": "response", "request_id": in.RequestID, "error": code, "accepted": code == ""}
		select {
		case sends <- out:
		default:
			return
		}
	}
}
