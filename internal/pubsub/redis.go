package pubsub

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"nbio-websocket/internal/observability"
)

// RedisAdapter implements PubSubAdapter using Redis pub/sub
type RedisAdapter struct {
	client       *redis.Client
	pubsub       *redis.PubSub
	mu           sync.RWMutex
	connected    bool
	url          string
	handlers     map[string]MessageHandler
	shutdownChan chan struct{}
}

// NewRedisAdapter creates a new Redis adapter
func NewRedisAdapter(url string) *RedisAdapter {
	return &RedisAdapter{
		url:          url,
		handlers:     make(map[string]MessageHandler),
		shutdownChan: make(chan struct{}),
	}
}

// Connect establishes connection to Redis
func (r *RedisAdapter) Connect(ctx context.Context) error {
	opt, err := redis.ParseURL(r.url)
	if err != nil {
		observability.Error("Failed to parse Redis URL", zap.Error(err))
		return err
	}

	r.client = redis.NewClient(opt)

	// Test connection
	if err := r.client.Ping(ctx).Err(); err != nil {
		observability.Error("Failed to connect to Redis", zap.Error(err))
		return err
	}

	r.mu.Lock()
	r.connected = true
	r.mu.Unlock()

	observability.Info("Connected to Redis",
		zap.String("adapter", "redis"),
		zap.String("url", r.url))

	return nil
}

// Close closes the Redis connection
func (r *RedisAdapter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.connected {
		return nil
	}

	close(r.shutdownChan)

	if r.pubsub != nil {
		if err := r.pubsub.Close(); err != nil {
			observability.Error("Failed to close Redis pubsub", zap.Error(err))
		}
	}

	if r.client != nil {
		if err := r.client.Close(); err != nil {
			observability.Error("Failed to close Redis client", zap.Error(err))
			return err
		}
	}

	r.connected = false
	r.handlers = make(map[string]MessageHandler)

	observability.Info("Disconnected from Redis")
	return nil
}

// Publish sends a message to a Redis channel
func (r *RedisAdapter) Publish(ctx context.Context, channel string, data []byte) error {
	if !r.IsConnected() {
		return ErrNotConnected
	}

	if err := r.client.Publish(ctx, channel, data).Err(); err != nil {
		observability.Error("Failed to publish to Redis",
			zap.String("channel", channel),
			zap.Error(err))
		return err
	}

	return nil
}

// Subscribe subscribes to a Redis channel
func (r *RedisAdapter) Subscribe(ctx context.Context, channel string, handler MessageHandler) error {
	if !r.IsConnected() {
		return ErrNotConnected
	}

	r.mu.Lock()
	r.handlers[channel] = handler

	// Create pubsub if not exists
	if r.pubsub == nil {
		r.pubsub = r.client.Subscribe(ctx, channel)
	} else {
		if err := r.pubsub.Subscribe(ctx, channel); err != nil {
			r.mu.Unlock()
			observability.Error("Failed to subscribe to Redis channel",
				zap.String("channel", channel),
				zap.Error(err))
			return err
		}
	}
	r.mu.Unlock()

	// Start listening in a goroutine
	go r.listen(ctx, channel)

	observability.Info("Subscribed to Redis channel",
		zap.String("channel", channel))

	return nil
}

// listen continuously receives messages from Redis
func (r *RedisAdapter) listen(ctx context.Context, channel string) {
	ch := r.pubsub.Channel()

	for {
		select {
		case <-r.shutdownChan:
			return
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			if msg.Channel != channel {
				continue
			}

			r.mu.RLock()
			handler, exists := r.handlers[channel]
			r.mu.RUnlock()

			if !exists {
				continue
			}

			message := &Message{
				Data:      []byte(msg.Payload),
				Channel:   msg.Channel,
				Timestamp: time.Now().Unix(),
			}

			if err := handler(message); err != nil {
				observability.Error("Handler error for Redis message",
					zap.String("channel", channel),
					zap.Error(err))
			}
		}
	}
}

// Unsubscribe unsubscribes from a Redis channel
func (r *RedisAdapter) Unsubscribe(ctx context.Context, channel string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.connected || r.pubsub == nil {
		return nil
	}

	delete(r.handlers, channel)

	if err := r.pubsub.Unsubscribe(ctx, channel); err != nil {
		observability.Error("Failed to unsubscribe from Redis channel",
			zap.String("channel", channel),
			zap.Error(err))
		return err
	}

	observability.Info("Unsubscribed from Redis channel",
		zap.String("channel", channel))

	return nil
}

// IsConnected returns connection status
func (r *RedisAdapter) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.connected
}

// Name returns the adapter name
func (r *RedisAdapter) Name() string {
	return string(AdapterTypeRedis)
}
