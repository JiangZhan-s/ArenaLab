//go:build reference

// Package client is a real protocol client shared by bots and the test driver.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	WS       *websocket.Conn
	Token    string
	write    sync.Mutex
	mu       sync.Mutex
	pending  map[string]chan map[string]any
	seq      atomic.Int64
	done     chan struct{}
	close    sync.Once
	snapshot map[string]any
}

func Dial(url, token, origin string) (*Client, *http.Response, error) {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+token)
	if origin != "" {
		h.Set("Origin", origin)
	}
	ws, resp, e := websocket.DefaultDialer.Dial(url, h)
	if e != nil {
		return nil, resp, e
	}
	c := Adopt(ws, token)
	return c, resp, nil
}
func Adopt(ws *websocket.Conn, token string) *Client {
	c := &Client{WS: ws, Token: token, pending: map[string]chan map[string]any{}, done: make(chan struct{})}
	go c.read()
	return c
}
func (c *Client) Close() { c.close.Do(func() { close(c.done); c.WS.Close() }) }
func (c *Client) read() {
	defer c.Close()
	for {
		var msg map[string]any
		if c.WS.ReadJSON(&msg) != nil {
			return
		}
		c.mu.Lock()
		if msg["state"] != nil {
			old, _ := c.snapshot["version"].(float64)
			next, _ := msg["version"].(float64)
			if next >= old {
				// The caller owns the response map after delivery. Keep a deep
				// copy so adding classifications cannot race the snapshot reader.
				b, _ := json.Marshal(msg)
				_ = json.Unmarshal(b, &c.snapshot)
			}
		}
		if id, ok := msg["request_id"].(string); ok {
			if ch := c.pending[id]; ch != nil {
				select {
				case ch <- msg:
				default:
				}
			}
		}
		c.mu.Unlock()
	}
}
func (c *Client) Call(ctx context.Context, msg map[string]any) (map[string]any, error) {
	id := strconv.FormatInt(c.seq.Add(1), 10)
	msg["request_id"] = id
	ch := make(chan map[string]any, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	defer func() { c.mu.Lock(); delete(c.pending, id); c.mu.Unlock() }()
	c.write.Lock()
	_ = c.WS.SetWriteDeadline(time.Now().Add(5 * time.Second))
	e := c.WS.WriteJSON(msg)
	c.write.Unlock()
	if e != nil {
		return nil, e
	}
	select {
	case out := <-ch:
		return out, nil
	case <-c.done:
		return nil, fmt.Errorf("connection_closed")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (c *Client) Oversize(n int) error {
	c.write.Lock()
	defer c.write.Unlock()
	_ = c.WS.SetWriteDeadline(time.Now().Add(3 * time.Second))
	return c.WS.WriteMessage(websocket.TextMessage, make([]byte, n))
}
func (c *Client) WaitClosed(ctx context.Context) bool {
	select {
	case <-c.done:
		return true
	case <-ctx.Done():
		return false
	}
}
func (c *Client) Snapshot() map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, _ := json.Marshal(c.snapshot)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}
