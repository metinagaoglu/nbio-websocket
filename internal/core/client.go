package core

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/lesismal/nbio/nbhttp/websocket"
	"go.uber.org/zap"

	"nbio-websocket/internal/observability"
)

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	closed atomic.Bool
	done   chan struct{}
}

func NewClient(hub *Hub, conn *websocket.Conn, bufferSize int) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, bufferSize),
		done: make(chan struct{}),
	}
}

// NewTestClient creates a client for testing purposes (without conn requirement)
func NewTestClient(hub *Hub, bufferSize int) *Client {
	return &Client{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, bufferSize),
		done: make(chan struct{}),
	}
}

// GetSendChannel returns the send channel for testing purposes
func (c *Client) GetSendChannel() <-chan []byte {
	return c.send
}

func (c *Client) Hub() *Hub {
	return c.hub
}

func (c *Client) Close() {
	if c.closed.CompareAndSwap(false, true) {
		close(c.done)
		c.hub.unregister <- c
		c.conn.Close()
	}
}

// Send safely sends a message to the client
func (c *Client) Send(msg []byte) error {
	if c.closed.Load() {
		return nil
	}

	select {
	case c.send <- msg:
		return nil
	case <-c.done:
		return nil
	default:
		return nil
	}
}

// SendJSON marshals and sends a JSON response to the client
func (c *Client) SendJSON(v interface{}) error {
	msg, err := json.Marshal(v)
	if err != nil {
		observability.Error("Failed to marshal JSON", zap.Error(err))
		return err
	}
	return c.Send(msg)
}

// WritePump handles sending messages to the WebSocket connection
func (c *Client) WritePump() {
	defer c.Close()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-c.done:
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
				c.Close()
				return
			}

		case <-c.done:
			return
		}
	}
}
