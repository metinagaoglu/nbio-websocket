package pubsub

import (
	"context"
)

// Message represents a message to be published/subscribed
type Message struct {
	Data      []byte
	Channel   string
	Timestamp int64
}

// MessageHandler is called when a message is received
type MessageHandler func(msg *Message) error

// PubSubAdapter defines the interface for pub/sub implementations
type PubSubAdapter interface {
	// Connect establishes connection to the pub/sub backend
	Connect(ctx context.Context) error

	// Close closes the connection
	Close() error

	// Publish sends a message to a channel
	Publish(ctx context.Context, channel string, data []byte) error

	// Subscribe subscribes to a channel and calls handler for each message
	Subscribe(ctx context.Context, channel string, handler MessageHandler) error

	// Unsubscribe unsubscribes from a channel
	Unsubscribe(ctx context.Context, channel string) error

	// IsConnected returns true if the adapter is connected
	IsConnected() bool

	// Name returns the adapter name
	Name() string
}

// AdapterType represents the type of pub/sub adapter
type AdapterType string

const (
	AdapterTypeLocal AdapterType = "local"
	AdapterTypeRedis AdapterType = "redis"
	AdapterTypeNATS  AdapterType = "nats"
)

// DefaultChannel is the default channel name for broadcasting
const DefaultChannel = "websocket.broadcast"
