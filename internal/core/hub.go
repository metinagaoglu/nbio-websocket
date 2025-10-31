package core

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"nbio-websocket/internal/observability"
	"nbio-websocket/internal/pubsub"
)

type Hub struct {
	clients      map[*Client]bool
	connToClient map[interface{}]*Client // ✅ NEW: O(1) lookup by connection
	broadcast    chan []byte
	register     chan *Client
	unregister   chan *Client
	disconnected chan *Client
	mu           sync.RWMutex
	pubsub       pubsub.PubSubAdapter // PubSub adapter for horizontal scaling
	pubsubChan   string               // PubSub channel name
}

func (h *Hub) Broadcast() chan []byte {
	return h.broadcast
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
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

// SetPubSub configures the hub to use a PubSub adapter for horizontal scaling
func (h *Hub) SetPubSub(ctx context.Context, adapter pubsub.PubSubAdapter, channel string) error {
	if adapter == nil {
		observability.Warn("PubSub adapter is nil, scaling disabled")
		return nil
	}

	h.pubsub = adapter
	h.pubsubChan = channel

	// Connect to the adapter
	if err := adapter.Connect(ctx); err != nil {
		observability.Error("Failed to connect PubSub adapter", zap.Error(err))
		return err
	}

	// Subscribe to receive messages from other instances
	err := adapter.Subscribe(ctx, channel, func(msg *pubsub.Message) error {
		// When we receive a message from pubsub, broadcast it to local clients
		h.broadcastToLocalClients(msg.Data)
		return nil
	})

	if err != nil {
		observability.Error("Failed to subscribe to PubSub channel",
			zap.String("channel", channel),
			zap.Error(err))
		return err
	}

	observability.Info("PubSub adapter configured",
		zap.String("adapter", adapter.Name()),
		zap.String("channel", channel))

	return nil
}

// broadcastToLocalClients sends a message to all local clients without re-publishing to pubsub
func (h *Hub) broadcastToLocalClients(message []byte) {
	h.mu.RLock()
	clientCount := 0
	for client := range h.clients {
		select {
		case client.send <- message:
			clientCount++
		default:
			// Signal cleanup if send channel is full
			go func(c *Client) {
				h.disconnected <- c
			}(client)
		}
	}
	h.mu.RUnlock()

	// Track messages sent
	for i := 0; i < clientCount; i++ {
		observability.GetMetrics().IncrementMessagesSent()
	}
}

func (h *Hub) Run(ctx context.Context) {
	observability.Info("Hub started")

	for {
		select {
		case <-ctx.Done():
			observability.Info("Hub shutdown initiated")
			h.closeAllClients()
			observability.Info("Hub shutdown complete")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.connToClient[client.conn] = client
			count := len(h.clients)
			h.mu.Unlock()
			observability.GetMetrics().IncrementConnections()
			observability.Info("Client registered",
				zap.Int("total_clients", count),
			)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.connToClient, client.conn)
				close(client.send)
				count := len(h.clients)
				observability.GetMetrics().IncrementDisconnects()
				observability.Info("Client unregistered",
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
				observability.GetMetrics().IncrementDisconnects()
				observability.Warn("Client disconnected due to send failure")
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			observability.GetMetrics().IncrementBroadcasts()

			// If PubSub is enabled, publish to the adapter (all instances will receive)
			if h.pubsub != nil && h.pubsub.IsConnected() {
				if err := h.pubsub.Publish(ctx, h.pubsubChan, message); err != nil {
					observability.Error("Failed to publish to PubSub",
						zap.String("channel", h.pubsubChan),
						zap.Error(err))
					// Fallback to local broadcast on error
					h.broadcastToLocalClients(message)
				}
				// Message will be received via subscription and distributed to local clients
			} else {
				// No PubSub or not connected, broadcast directly to local clients
				h.broadcastToLocalClients(message)
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
	observability.Info("Closed all clients", zap.Int("count", count))
}
