//go:build reference

package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JiangZhan-s/ArenaLab/internal/reference/dedicated"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/realtime"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/store"
	"github.com/JiangZhan-s/ArenaLab/web"
	"github.com/gorilla/websocket"
)

type Identity struct {
	Player, Epoch int
	Expires       int64
}
type Service struct {
	Dedicated       *dedicated.Control
	Game            *realtime.Game
	RealClock       atomic.Bool
	Store           *store.Store
	Secret          string
	Origin          string
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.Mutex
	rooms           map[string]*room
	peers           map[*peer]bool
	owned           map[string]int
	Drain           atomic.Bool
	Maintenance     sync.Mutex
	Quiesced        atomic.Bool
	MaxQueue        atomic.Int64
	SlowDisconnects atomic.Int64
}
type request struct {
	ctx    context.Context
	who    Identity
	cmd    store.Command
	result chan store.Reply
}
type room struct {
	queue chan request
	done  chan struct{}
}
type peer struct {
	ws    *websocket.Conn
	who   Identity
	queue chan any
	done  chan struct{}
	once  sync.Once
	match string
}

func (p *peer) close() { p.once.Do(func() { close(p.done); p.ws.Close() }) }
func New(st *store.Store, secret, origin string) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{Store: st, Secret: secret, Origin: origin, ctx: ctx, cancel: cancel, rooms: map[string]*room{}, peers: map[*peer]bool{}, owned: map[string]int{}}
	go s.background()
	return s
}
func (s *Service) Close() {
	s.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	for p := range s.peers {
		p.close()
	}
}
func (s *Service) token(i Identity) string {
	b, _ := json.Marshal(i)
	payload := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, []byte(s.Secret))
	mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (s *Service) identity(token string) (Identity, error) {
	var i Identity
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return i, errors.New("unauthorized")
	}
	mac := hmac.New(sha256.New, []byte(s.Secret))
	mac.Write([]byte(parts[0]))
	sig, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		return i, errors.New("unauthorized")
	}
	b, e := base64.RawURLEncoding.DecodeString(parts[0])
	if e != nil {
		return i, e
	}
	if e = json.Unmarshal(b, &i); e != nil {
		return i, e
	}
	if i.Expires <= time.Now().Unix() {
		return i, errors.New("unauthorized")
	}
	return i, nil
}
func (s *Service) auth(r *http.Request) (Identity, error) {
	v := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if v == "" {
		for _, p := range websocket.Subprotocols(r) {
			if strings.HasPrefix(p, "ticket.") {
				v = strings.TrimPrefix(p, "ticket.")
				break
			}
		}
	}
	if v == "" {
		if c, e := r.Cookie("arena_session"); e == nil {
			v = c.Value
		}
	}
	return s.identity(v)
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /", web.Handler())
	mux.HandleFunc("GET /v1/score", func(w http.ResponseWriter, r *http.Request) {
		i, e := s.auth(r)
		if e != nil {
			write(w, 401, map[string]any{"error": "unauthorized"})
			return
		}
		var score int
		e = s.Store.DB.QueryRowContext(r.Context(), "SELECT COALESCE(SUM(delta),0) FROM ledger WHERE player_id=?", i.Player).Scan(&score)
		if e != nil {
			write(w, 500, map[string]any{"error": "internal"})
			return
		}
		write(w, 200, map[string]any{"score": score})
	})
	if s.Dedicated != nil {
		s.dedicatedRoutes(mux)
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]any{"ready": true}) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		code := 200
		if s.Drain.Load() {
			code = 503
		}
		write(w, code, map[string]any{"ready": !s.Drain.Load()})
	})
	mux.HandleFunc("POST /v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && o != s.Origin {
			write(w, 403, map[string]any{"error": "origin_denied"})
			return
		}
		var v struct {
			Player     int    `json:"player"`
			Credential string `json:"credential"`
		}
		if decode(w, r, &v) != nil || !hmac.Equal([]byte(v.Credential), []byte(s.Secret+":"+strconv.Itoa(v.Player))) {
			write(w, 401, map[string]any{"error": "unauthorized"})
			return
		}
		epoch, e := s.Store.Login(r.Context(), v.Player)
		if e != nil {
			write(w, 401, map[string]any{"error": "unauthorized"})
			return
		}
		token := s.token(Identity{v.Player, epoch, time.Now().Add(10 * time.Minute).Unix()})
		http.SetCookie(w, &http.Cookie{Name: "arena_session", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 600})
		write(w, 200, map[string]any{"error": "", "token": token, "epoch": epoch})
	})
	mux.HandleFunc("GET /v1/ws", s.websocket)
	mux.HandleFunc("GET /v1/realtime", func(w http.ResponseWriter, r *http.Request) {
		if _, e := s.auth(r); e != nil {
			write(w, 401, map[string]any{"error": "unauthorized"})
			return
		}
		if s.Game == nil {
			write(w, 404, map[string]any{"error": "not_found"})
			return
		}
		write(w, 200, s.Game.Snapshot())
	})
	mux.HandleFunc("GET /v1/matches/{id}", func(w http.ResponseWriter, r *http.Request) {
		i, e := s.auth(r)
		if e != nil {
			write(w, 401, map[string]any{"error": "unauthorized"})
			return
		}
		m, e := s.Store.Get(r.Context(), r.PathValue("id"))
		if e != nil {
			write(w, 404, map[string]any{"error": "not_found"})
			return
		}
		if i.Player != m.Players[0] && i.Player != m.Players[1] {
			write(w, 403, map[string]any{"error": "forbidden"})
			return
		}
		write(w, 200, map[string]any{"version": m.State.Version, "state": m.State, "status": m.Status})
	})
	mux.HandleFunc("POST /v1/tickets", func(w http.ResponseWriter, r *http.Request) {
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
		}
		if decode(w, r, &v) != nil {
			write(w, 400, map[string]any{"error": "invalid_request"})
			return
		}
		out, e := s.Store.Enqueue(r.Context(), i.Player, i.Epoch, v.Request)
		if e != nil {
			write(w, 500, map[string]any{"error": "internal"})
			return
		}
		write(w, 200, out)
	})
	mux.HandleFunc("DELETE /v1/tickets", func(w http.ResponseWriter, r *http.Request) {
		i, e := s.auth(r)
		if e != nil {
			write(w, 401, map[string]any{"error": "unauthorized"})
			return
		}
		state, e := s.Store.Cancel(r.Context(), i.Player, i.Epoch)
		if e != nil {
			write(w, 500, map[string]any{"error": "internal"})
			return
		}
		write(w, 200, map[string]any{"state": state})
	})
	mux.HandleFunc("GET /v1/tickets", func(w http.ResponseWriter, r *http.Request) {
		i, e := s.auth(r)
		if e != nil {
			write(w, 401, map[string]any{"error": "unauthorized"})
			return
		}
		var state, id string
		e = s.Store.DB.QueryRowContext(r.Context(), "SELECT status,match_id FROM tickets WHERE player_id=? ORDER BY id DESC LIMIT 1", i.Player).Scan(&state, &id)
		if e != nil {
			write(w, 404, map[string]any{"error": "not_found"})
			return
		}
		write(w, 200, map[string]any{"state": state, "match_id": id})
	})
	mux.HandleFunc("POST /v1/results", func(w http.ResponseWriter, r *http.Request) { write(w, 403, map[string]any{"error": "forbidden"}) })
	return mux
}
func (s *Service) submit(ctx context.Context, i Identity, c store.Command) store.Reply {
	s.mu.Lock()
	rm := s.rooms[c.MatchID]
	if rm == nil {
		rm = &room{make(chan request, 64), make(chan struct{})}
		s.rooms[c.MatchID] = rm
		go s.runRoom(c.MatchID, rm)
	}
	req := request{ctx, i, c, make(chan store.Reply, 1)}
	select {
	case <-rm.done:
		s.mu.Unlock()
		return store.Reply{Error: "retry"}
	case <-ctx.Done():
		s.mu.Unlock()
		return store.Reply{Error: "outcome_unknown"}
	case rm.queue <- req:
	default:
		s.mu.Unlock()
		return store.Reply{Error: "overloaded"}
	}
	s.mu.Unlock()
	s.recordDepth(len(rm.queue))
	select {
	case r := <-req.result:
		return r
	case <-ctx.Done():
		return store.Reply{Error: "outcome_unknown"}
	case <-s.ctx.Done():
		return store.Reply{Error: "outcome_unknown"}
	}
}
func (s *Service) recordDepth(n int) {
	for {
		old := s.MaxQueue.Load()
		if int64(n) <= old || s.MaxQueue.CompareAndSwap(old, int64(n)) {
			return
		}
	}
}
func (s *Service) runRoom(id string, rm *room) {
	defer func() {
		s.mu.Lock()
		if s.rooms[id] == rm {
			delete(s.rooms, id)
		}
		select {
		case <-rm.done:
		default:
			close(rm.done)
		}
		s.mu.Unlock()
	}()
	idle := time.NewTimer(time.Second)
	defer idle.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-idle.C:
			s.mu.Lock()
			if len(rm.queue) > 0 {
				s.mu.Unlock()
				idle.Reset(time.Second)
				continue
			}
			delete(s.rooms, id)
			close(rm.done)
			s.mu.Unlock()
			return
		case r := <-rm.queue:
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(time.Second)
			s.mu.Lock()
			epoch := s.owned[id]
			s.mu.Unlock()
			if epoch == 0 {
				if m, e := s.Store.Get(r.ctx, id); e == nil && m.Owner == s.Store.Node {
					epoch = m.Epoch
				}
			}
			out, e := s.Store.Action(r.ctx, r.who.Player, r.who.Epoch, epoch, r.cmd)
			if e != nil {
				out = store.Reply{Error: "outcome_unknown"}
			}
			r.result <- out
			if out.Error == "" {
				s.broadcast(id, map[string]any{"type": "snapshot", "version": out.Version, "state": out.State})
			}
		}
	}
}
func (s *Service) broadcast(id string, v any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for p := range s.peers {
		if p.match == id {
			select {
			case p.queue <- v:
				s.recordDepth(len(p.queue))
			default:
				s.SlowDisconnects.Add(1)
				s.recordDepth(cap(p.queue))
				p.close()
			}
		}
	}
}
func (s *Service) background() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.Maintenance.Lock()
			if s.Quiesced.Load() {
				s.Maintenance.Unlock()
				continue
			}
			ctx, cancel := context.WithTimeout(s.ctx, 2*time.Second)
			s.mu.Lock()
			known := map[string]int{}
			for id, e := range s.owned {
				known[id] = e
			}
			s.mu.Unlock()
			owned, e := s.Store.Own(ctx, known)
			if e == nil {
				s.mu.Lock()
				for id, epoch := range owned {
					s.owned[id] = epoch
				}
				s.mu.Unlock()
				for id, epoch := range owned {
					changed, e := s.Store.Timeout(ctx, id, epoch)
					if e == nil && changed {
						if m, e := s.Store.Get(ctx, id); e == nil {
							s.broadcast(id, map[string]any{"type": "snapshot", "version": m.State.Version, "state": m.State})
						}
					}
				}
			}
			if !s.Drain.Load() {
				_ = s.Store.MatchQueued(ctx)
			}
			_ = s.Store.ExpireReady(ctx)
			cancel()
			s.Maintenance.Unlock()
		}
	}
}
func (s *Service) websocket(w http.ResponseWriter, r *http.Request) {
	i, e := s.auth(r)
	if e != nil {
		write(w, 401, map[string]any{"error": "unauthorized"})
		return
	}
	if o := r.Header.Get("Origin"); o != "" && o != s.Origin {
		write(w, 403, map[string]any{"error": "origin_denied"})
		return
	}
	u := websocket.Upgrader{Subprotocols: []string{"arena"}, CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == "" || r.Header.Get("Origin") == s.Origin }}
	if e = s.Store.ClaimConnection(r.Context(), i.Player, i.Epoch); e != nil {
		write(w, 403, map[string]any{"error": e.Error()})
		return
	}
	ws, e := u.Upgrade(w, r, nil)
	if e != nil {
		return
	}
	ws.SetReadLimit(1 << 20)
	if tcp, ok := ws.UnderlyingConn().(*net.TCPConn); ok {
		_ = tcp.SetWriteBuffer(4096)
	}
	p := &peer{ws: ws, who: i, queue: make(chan any, 64), done: make(chan struct{})}
	s.mu.Lock()
	s.peers[p] = true
	s.mu.Unlock()
	defer func() { p.close(); s.mu.Lock(); delete(s.peers, p); s.mu.Unlock() }()
	go func() {
		defer p.close()
		for {
			select {
			case <-p.done:
				return
			case v := <-p.queue:
				_ = ws.SetWriteDeadline(time.Now().Add(2 * time.Second))
				if ws.WriteJSON(v) != nil {
					return
				}
			}
		}
	}()
	budget := time.Now()
	count := 0
	inflight := make(chan struct{}, 64)
	for {
		_ = ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		var msg struct {
			Seq       int    `json:"seq"`
			Type      string `json:"type"`
			RequestID string `json:"request_id"`
			store.Command
			Last int `json:"last_seen_version"`
		}
		if e = ws.ReadJSON(&msg); e != nil {
			return
		}
		if time.Since(budget) >= time.Second {
			budget = time.Now()
			count = 0
		}
		count++
		if count > 200 {
			return
		}
		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		out := map[string]any{"request_id": msg.RequestID, "type": "response"}
		async := false
		switch msg.Type {
		case "input":
			if s.Game == nil {
				out["error"] = "not_found"
			} else {
				var epoch int
				err := s.Store.DB.QueryRowContext(ctx, "SELECT epoch FROM players WHERE id=?", i.Player).Scan(&epoch)
				if err != nil || epoch != i.Epoch {
					out["error"] = "stale_session"
					out["accepted"] = false
				} else {
					code := s.Game.Input(i.Player, realtime.Input{Seq: msg.Seq, DX: msg.DX, DY: msg.DY})
					out["error"] = code
					out["accepted"] = code == ""
				}
			}
		case "action":
			select {
			case inflight <- struct{}{}:
				async = true
				go func(command store.Command, rid string) {
					defer func() { <-inflight }()
					actionCtx, stop := context.WithTimeout(s.ctx, 5*time.Second)
					defer stop()
					reply := s.submit(actionCtx, i, command)
					response := map[string]any{"request_id": rid, "type": "response", "error": reply.Error, "version": reply.Version, "state": reply.State, "committed": reply.Committed}
					select {
					case p.queue <- response:
					case <-p.done:
					default:
						p.close()
					}
				}(msg.Command, msg.RequestID)
			default:
				out["error"] = "overloaded"
			}
		case "join", "resume":
			if s.Game != nil && msg.MatchID == "realtime" {
				s.mu.Lock()
				p.match = "realtime"
				s.mu.Unlock()
				snap := s.Game.Snapshot()
				out["state"] = snap
				out["version"] = snap.Tick
				out["error"] = ""
				break
			}
			m, err := s.Store.Get(ctx, msg.MatchID)
			var current int
			authErr := s.Store.DB.QueryRowContext(ctx, "SELECT epoch FROM players WHERE id=?", i.Player).Scan(&current)
			if authErr != nil || current != i.Epoch {
				out["error"] = "stale_session"
			} else if err != nil {
				out["error"] = "not_found"
			} else if i.Player != m.Players[0] && i.Player != m.Players[1] {
				out["error"] = "forbidden"
			} else {
				s.mu.Lock()
				p.match = m.ID
				s.mu.Unlock()
				out["error"] = ""
				out["mode"] = "snapshot"
				out["version"] = m.State.Version
				out["state"] = m.State
				out["players"] = m.Players
				out["player_slot"] = 0
				if i.Player == m.Players[1] {
					out["player_slot"] = 1
				}
			}
		case "ready":
			state, err := s.Store.Ready(ctx, i.Player, i.Epoch, msg.MatchID)
			if err != nil {
				out["error"] = "internal"
			} else {
				out["state"] = state
				out["error"] = ""
			}
		case "lookup":
			reply, err := s.Store.Lookup(ctx, i.Player, i.Epoch, msg.MatchID, msg.ID)
			if err != nil {
				out["error"] = "internal"
			} else {
				out["error"] = reply.Error
				out["committed"] = reply.Committed
				out["version"] = reply.Version
			}
		default:
			out["error"] = "invalid_message"
		}
		cancel()
		if async {
			continue
		}
		select {
		case p.queue <- out:
			s.recordDepth(len(p.queue))
		case <-p.done:
			return
		default:
			s.SlowDisconnects.Add(1)
			s.recordDepth(cap(p.queue))
			return
		}
	}
}
func (s *Service) Stats() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]any{"live_rooms": len(s.rooms), "connections": len(s.peers), "max_queue_depth": s.MaxQueue.Load(), "slow_disconnects": s.SlowDisconnects.Load()}
}
func (s *Service) String() string { return fmt.Sprintf("arena %s", s.Store.Node) }
