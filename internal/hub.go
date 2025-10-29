package internal

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type Hub struct {
	clients      map[*Client]bool
	connToClient map[interface{}]*Client // ✅ NEW: O(1) lookup by connection
	broadcast    chan []byte
	register     chan *Client
	unregister   chan *Client
	disconnected chan *Client
	mu           sync.RWMutex
}

func (h *Hub) Broadcast() chan []byte {
	return h.broadcast
}

func NewHub() *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		connToClient: make(map[interface{}]*Client), // ✅ Initialize reverse map
		broadcast:    make(chan []byte),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		disconnected: make(chan *Client),
	}
}

func (h *Hub) Run(ctx context.Context) {
	Info("Hub started")

	for {
		select {
		case <-ctx.Done():
			Info("Hub shutdown initiated")
			h.closeAllClients()
			Info("Hub shutdown complete")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.connToClient[client.conn] = client
			count := len(h.clients)
			h.mu.Unlock()
			GetMetrics().IncrementConnections()
			Info("Client registered",
				zap.Int("total_clients", count),
			)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.connToClient, client.conn)
				close(client.send)
				count := len(h.clients)
				GetMetrics().IncrementDisconnects()
				Info("Client unregistered",
					zap.Int("remaining_clients", count),
				)
			}
			h.mu.Unlock()

		case client := <-h.disconnected:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.connToClient, client.conn)
				close(client.send)
				GetMetrics().IncrementDisconnects()
				Warn("Client disconnected due to send failure")
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			GetMetrics().IncrementBroadcasts()
			h.mu.RLock()
			clientCount := 0
			for client := range h.clients {
				select {
				case client.send <- message:
					// ✅ Message sent successfully
					clientCount++
				default:
					// ✅ FIXED: Signal cleanup instead of direct delete under RLock
					go func(c *Client) {
						h.disconnected <- c
					}(client)
				}
			}
			h.mu.RUnlock()
			// Track actual messages sent
			for i := 0; i < clientCount; i++ {
				GetMetrics().IncrementMessagesSent()
			}
		}
	}
}

// GetClientByConn returns the client associated with the connection (O(1) lookup)
func (h *Hub) GetClientByConn(conn interface{}) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	client, ok := h.connToClient[conn]
	return client, ok
}

// closeAllClients gracefully closes all connected clients
func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	count := len(h.clients)
	for client := range h.clients {
		close(client.send)
	}

	// Clear maps
	h.clients = make(map[*Client]bool)
	h.connToClient = make(map[interface{}]*Client)
	Info("Closed all clients", zap.Int("count", count))
}
