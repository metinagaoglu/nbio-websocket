package pubsub

import (
	"fmt"

	"go.uber.org/zap"

	"nbio-websocket/internal/observability"
)

// NewAdapter creates a new PubSubAdapter based on the adapter type
func NewAdapter(adapterType AdapterType, config map[string]string) (PubSubAdapter, error) {
	switch adapterType {
	case AdapterTypeLocal:
		observability.Info("Creating local pub/sub adapter")
		return NewLocalAdapter(), nil

	case AdapterTypeRedis:
		url, ok := config["url"]
		if !ok || url == "" {
			return nil, fmt.Errorf("redis adapter requires 'url' in config")
		}
		observability.Info("Creating Redis pub/sub adapter",
			zap.String("url", url))
		return NewRedisAdapter(url), nil

	case AdapterTypeNATS:
		url, ok := config["url"]
		if !ok || url == "" {
			return nil, fmt.Errorf("nats adapter requires 'url' in config")
		}
		observability.Info("Creating NATS pub/sub adapter",
			zap.String("url", url))
		return NewNATSAdapter(url), nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidAdapter, adapterType)
	}
}
