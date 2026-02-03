package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/playok/only1mon/internal/model"
	"nhooyr.io/websocket"
)

const (
	pingInterval = 30 * time.Second
	readTimeout  = 60 * time.Second
)

// Hub manages WebSocket connections and broadcasts.
type Hub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
	reg     chan *wsClient
	unreg   chan *wsClient
}

type wsClient struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	subs map[string]bool // subscribed metric prefixes
	mu   sync.Mutex
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*wsClient]struct{}),
		reg:     make(chan *wsClient, 16),
		unreg:   make(chan *wsClient, 16),
	}
}

// Run processes register/unregister events.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.reg:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()
		case c := <-h.unreg:
			h.mu.Lock()
			delete(h.clients, c)
			h.mu.Unlock()
			close(c.send)
		}
	}
}

// Broadcast sends samples to all connected clients that have matching subscriptions.
func (h *Hub) Broadcast(samples []model.MetricSample) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.clients) == 0 {
		return
	}

	// Group by potential subscription prefixes
	data, err := json.Marshal(map[string]interface{}{
		"type":    "metrics",
		"samples": samples,
	})
	if err != nil {
		return
	}

	for c := range h.clients {
		// If client has no subs, send everything; otherwise filter
		if c.hasMatchingSub(samples) {
			select {
			case c.send <- data:
			default:
				// client too slow, skip
			}
		}
	}
}

// BroadcastAlerts sends alerts to all connected clients.
func (h *Hub) BroadcastAlerts(alerts []model.Alert) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.clients) == 0 {
		return
	}

	data, err := json.Marshal(map[string]interface{}{
		"type":   "alerts",
		"alerts": alerts,
	})
	if err != nil {
		return
	}

	for c := range h.clients {
		select {
		case c.send <- data:
		default:
		}
	}
}

func (c *wsClient) hasMatchingSub(samples []model.MetricSample) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.subs) == 0 {
		return true // no filter = receive all
	}
	for _, s := range samples {
		for prefix := range c.subs {
			if len(s.MetricName) >= len(prefix) && s.MetricName[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

func (c *wsClient) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.conn.Ping(ctx); err != nil {
				return
			}
		}
	}
}

// HandleWS handles WebSocket upgrade and manages the connection.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // allow any origin for local tool
	})
	if err != nil {
		log.Printf("[ws] accept error: %v", err)
		return
	}

	client := &wsClient{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 64),
		subs: make(map[string]bool),
	}

	h.reg <- client

	ctx := r.Context()
	go client.pingLoop(ctx)
	go client.writePump(ctx)
	client.readPump(ctx)
}

func (c *wsClient) readPump(ctx context.Context) {
	defer func() {
		c.hub.unreg <- c
		c.conn.Close(websocket.StatusNormalClosure, "bye")
	}()

	for {
		readCtx, cancel := context.WithTimeout(ctx, readTimeout)
		_, data, err := c.conn.Read(readCtx)
		cancel()
		if err != nil {
			return
		}
		// Parse subscription messages
		var msg struct {
			Type    string   `json:"type"`
			Metrics []string `json:"metrics"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "subscribe":
			c.mu.Lock()
			for _, m := range msg.Metrics {
				c.subs[m] = true
			}
			c.mu.Unlock()
		case "unsubscribe":
			c.mu.Lock()
			for _, m := range msg.Metrics {
				delete(c.subs, m)
			}
			c.mu.Unlock()
		}
	}
}

func (c *wsClient) writePump(ctx context.Context) {
	for data := range c.send {
		if err := c.conn.Write(ctx, websocket.MessageText, data); err != nil {
			return
		}
	}
}
