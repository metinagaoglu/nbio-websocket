package pubsub

import (
	"context"
	"sync"
	"time"
)

// LocalAdapter implements PubSubAdapter for in-memory pub/sub (single instance)
type LocalAdapter struct {
	mu           sync.RWMutex
	subscribers  map[string][]MessageHandler
	connected    bool
	shutdownChan chan struct{}
}

// NewLocalAdapter creates a new local (in-memory) adapter
func NewLocalAdapter() *LocalAdapter {
	return &LocalAdapter{
		subscribers:  make(map[string][]MessageHandler),
		shutdownChan: make(chan struct{}),
	}
}

// Connect establishes the "connection" (no-op for local)
func (l *LocalAdapter) Connect(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.connected = true
	return nil
}

// Close closes the adapter
func (l *LocalAdapter) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.connected {
		close(l.shutdownChan)
		l.connected = false
		l.subscribers = make(map[string][]MessageHandler)
	}
	return nil
}

// Publish sends a message to all local subscribers
func (l *LocalAdapter) Publish(ctx context.Context, channel string, data []byte) error {
	l.mu.RLock()
	handlers := l.subscribers[channel]
	l.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	msg := &Message{
		Data:      data,
		Channel:   channel,
		Timestamp: time.Now().Unix(),
	}

	// Call all handlers asynchronously
	for _, handler := range handlers {
		go func(h MessageHandler) {
			_ = h(msg)
		}(handler)
	}

	return nil
}

// Subscribe adds a handler for a channel
func (l *LocalAdapter) Subscribe(ctx context.Context, channel string, handler MessageHandler) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.subscribers[channel] = append(l.subscribers[channel], handler)
	return nil
}

// Unsubscribe removes all handlers for a channel
func (l *LocalAdapter) Unsubscribe(ctx context.Context, channel string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.subscribers, channel)
	return nil
}

// IsConnected returns connection status
func (l *LocalAdapter) IsConnected() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.connected
}

// Name returns the adapter name
func (l *LocalAdapter) Name() string {
	return string(AdapterTypeLocal)
}
