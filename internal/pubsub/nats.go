package pubsub

import (
	"context"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"nbio-websocket/internal/observability"
)

// NATSAdapter implements PubSubAdapter using NATS
type NATSAdapter struct {
	conn         *nats.Conn
	mu           sync.RWMutex
	connected    bool
	url          string
	subscriptions map[string]*nats.Subscription
	shutdownChan chan struct{}
}

// NewNATSAdapter creates a new NATS adapter
func NewNATSAdapter(url string) *NATSAdapter {
	return &NATSAdapter{
		url:           url,
		subscriptions: make(map[string]*nats.Subscription),
		shutdownChan:  make(chan struct{}),
	}
}

// Connect establishes connection to NATS
func (n *NATSAdapter) Connect(ctx context.Context) error {
	conn, err := nats.Connect(n.url,
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				observability.Error("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			observability.Info("NATS reconnected",
				zap.String("url", nc.ConnectedUrl()))
		}),
	)

	if err != nil {
		observability.Error("Failed to connect to NATS", zap.Error(err))
		return err
	}

	n.mu.Lock()
	n.conn = conn
	n.connected = true
	n.mu.Unlock()

	observability.Info("Connected to NATS",
		zap.String("adapter", "nats"),
		zap.String("url", n.url))

	return nil
}

// Close closes the NATS connection
func (n *NATSAdapter) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.connected {
		return nil
	}

	close(n.shutdownChan)

	// Unsubscribe all
	for channel, sub := range n.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			observability.Error("Failed to unsubscribe from NATS channel",
				zap.String("channel", channel),
				zap.Error(err))
		}
	}

	n.subscriptions = make(map[string]*nats.Subscription)

	if n.conn != nil {
		n.conn.Close()
	}

	n.connected = false

	observability.Info("Disconnected from NATS")
	return nil
}

// Publish sends a message to a NATS subject
func (n *NATSAdapter) Publish(ctx context.Context, channel string, data []byte) error {
	if !n.IsConnected() {
		return ErrNotConnected
	}

	if err := n.conn.Publish(channel, data); err != nil {
		observability.Error("Failed to publish to NATS",
			zap.String("channel", channel),
			zap.Error(err))
		return err
	}

	return nil
}

// Subscribe subscribes to a NATS subject
func (n *NATSAdapter) Subscribe(ctx context.Context, channel string, handler MessageHandler) error {
	if !n.IsConnected() {
		return ErrNotConnected
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Check if already subscribed
	if _, exists := n.subscriptions[channel]; exists {
		observability.Warn("Already subscribed to NATS channel",
			zap.String("channel", channel))
		return nil
	}

	// Create subscription with message handler
	sub, err := n.conn.Subscribe(channel, func(msg *nats.Msg) {
		message := &Message{
			Data:      msg.Data,
			Channel:   msg.Subject,
			Timestamp: time.Now().Unix(),
		}

		if err := handler(message); err != nil {
			observability.Error("Handler error for NATS message",
				zap.String("channel", channel),
				zap.Error(err))
		}
	})

	if err != nil {
		observability.Error("Failed to subscribe to NATS channel",
			zap.String("channel", channel),
			zap.Error(err))
		return err
	}

	n.subscriptions[channel] = sub

	observability.Info("Subscribed to NATS channel",
		zap.String("channel", channel))

	return nil
}

// Unsubscribe unsubscribes from a NATS subject
func (n *NATSAdapter) Unsubscribe(ctx context.Context, channel string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.connected {
		return nil
	}

	sub, exists := n.subscriptions[channel]
	if !exists {
		return nil
	}

	if err := sub.Unsubscribe(); err != nil {
		observability.Error("Failed to unsubscribe from NATS channel",
			zap.String("channel", channel),
			zap.Error(err))
		return err
	}

	delete(n.subscriptions, channel)

	observability.Info("Unsubscribed from NATS channel",
		zap.String("channel", channel))

	return nil
}

// IsConnected returns connection status
func (n *NATSAdapter) IsConnected() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.connected && n.conn != nil && n.conn.IsConnected()
}

// Name returns the adapter name
func (n *NATSAdapter) Name() string {
	return string(AdapterTypeNATS)
}
