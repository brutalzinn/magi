package main

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

// hub is a minimal websocket broadcast fan-out: every connected dashboard
// receives every telemetry event. Writes are serialised by mu.
type hub struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]struct{}
}

func newHub() *hub {
	return &hub{conns: make(map[*websocket.Conn]struct{})}
}

func (h *hub) add(c *websocket.Conn) {
	h.mu.Lock()
	h.conns[c] = struct{}{}
	h.mu.Unlock()
}

func (h *hub) remove(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.conns, c)
	h.mu.Unlock()
}

// broadcast marshals v to JSON and sends it to every connected client, dropping
// any connection that errors.
func (h *hub) broadcast(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.conns {
		if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
			_ = c.Close()
			delete(h.conns, c)
		}
	}
}
