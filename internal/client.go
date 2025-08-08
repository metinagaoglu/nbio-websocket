package internal

import (
	"sync/atomic"
	"time"

	"github.com/lesismal/nbio/nbhttp/websocket"
)

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	closed atomic.Bool   // ✅ Thread-safe close flag
	done   chan struct{} // ✅ Coordination channel for shutdown
}

func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
		done: make(chan struct{}), // ✅ Initialize coordination channel
	}
}

func (c *Client) Hub() *Hub {
	return c.hub
}

// Close gracefully closes the client connection (thread-safe, idempotent)
func (c *Client) Close() {
	// ✅ Only close once using atomic compare-and-swap
	if c.closed.CompareAndSwap(false, true) {
		close(c.done)         // Signal all goroutines to stop
		c.hub.unregister <- c // Remove from hub
		c.conn.Close()        // Close the WebSocket connection
	}
}

// Send safely sends a message to the client
func (c *Client) Send(msg []byte) error {
	if c.closed.Load() {
		return nil // Client already closed
	}

	select {
	case c.send <- msg:
		return nil
	case <-c.done:
		return nil // Client closing
	default:
		// Buffer full, client too slow
		return nil
	}
}

// WritePump handles sending messages to the WebSocket connection
func (c *Client) WritePump() {
	defer c.Close() // ✅ Always cleanup on exit

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				// Channel closed by hub
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				// Write failed, cleanup and exit
				return
			}

		case <-c.done:
			// ✅ Graceful shutdown signal
			return
		}
	}
}

// StartPing sends periodic ping messages to keep the connection alive
func (c *Client) StartPing(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
				c.Close() // ✅ Proper cleanup on ping failure
				return
			}

		case <-c.done:
			// ✅ Graceful shutdown
			return
		}
	}
}
