package internal

import (
	"context"
	"log"
	"sync"
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
	log.Println("Hub: Started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Hub: Shutdown signal received")
			h.closeAllClients()
			log.Println("Hub: All clients closed")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.connToClient[client.conn] = client // ✅ Add reverse mapping
			count := len(h.clients)
			h.mu.Unlock()
			log.Printf("Hub: Client registered (total: %d)", count)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.connToClient, client.conn) // ✅ Remove reverse mapping
				close(client.send)
				count := len(h.clients)
				log.Printf("Hub: Client unregistered (remaining: %d)", count)
			}
			h.mu.Unlock()

		case client := <-h.disconnected:
			// ✅ NEW CASE: Handle cleanup from failed sends
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.connToClient, client.conn) // ✅ Remove reverse mapping
				close(client.send)
				log.Printf("Hub: Client disconnected due to send failure")
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
					// ✅ Message sent successfully
				default:
					// ✅ FIXED: Signal cleanup instead of direct delete under RLock
					go func(c *Client) {
						h.disconnected <- c
					}(client)
				}
			}
			h.mu.RUnlock()
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
		// Note: client.Close() will be available after Task 1.3
	}

	// Clear maps
	h.clients = make(map[*Client]bool)
	h.connToClient = make(map[interface{}]*Client)
	log.Printf("Hub: Closed %d clients", count)
}
