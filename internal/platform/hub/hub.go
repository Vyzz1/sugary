// Package hub manages WebSocket client connections and broadcasts messages
// to all connected clients.
package hub

import (
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Client wraps a single WebSocket connection.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// Hub holds the set of active clients and broadcasts messages to all of them.
// It implements usecase.MealAnalysisPublisher via its Broadcast method.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

const sendBufferSize = 256

// New creates a Hub and starts its internal event loop in the background.
func New() *Hub {
	h := &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, sendBufferSize),
	}
	go h.run()
	return h
}

// Broadcast enqueues msg for delivery to all connected clients.
// It is non-blocking and safe to call from any goroutine.
// Implements usecase.MealAnalysisPublisher.
func (h *Hub) Broadcast(msg []byte) {
	select {
	case h.broadcast <- msg:
	default:
		zap.L().Warn("hub_broadcast_dropped", zap.Int("msg_len", len(msg)))
	}
}

// Register adds a new client to the hub.
func (h *Hub) Register(c *Client) {
	h.register <- c
}

// Unregister removes a client from the hub and closes its send channel.
func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

// NewClient creates a Client with the provided WebSocket connection and
// starts its write pump goroutine. The caller must also call Register.
func NewClient(h *Hub, conn *websocket.Conn) *Client {
	c := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, sendBufferSize),
	}
	go c.writePump()
	return c
}

// run is the hub's event loop — it serialises register/unregister/broadcast
// operations so no locking is needed on the clients map itself.
func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			zap.L().Info("ws_client_connected")

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
			zap.L().Info("ws_client_disconnected")

		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// Slow client — drop and let writePump close on its own.
					zap.L().Warn("ws_client_send_dropped")
					go h.Unregister(c)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// writePump drains the client's send channel and writes to the WebSocket.
// It exits (and signals unregister) when the channel is closed or a write fails.
func (c *Client) writePump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			zap.L().Warn("ws_write_error", zap.Error(err))
			return
		}
	}
}
