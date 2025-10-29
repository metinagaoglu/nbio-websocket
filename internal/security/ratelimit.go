package security

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"nbio-websocket/internal/observability"
)

// RateLimiter implements token bucket rate limiting per client
type RateLimiter struct {
	clients map[interface{}]*bucket
	mu      sync.RWMutex

	// Configuration
	rate     int           // tokens per interval
	interval time.Duration // refill interval
	burst    int           // max tokens (bucket capacity)
}

// bucket holds rate limit state for a single client
type bucket struct {
	tokens    int
	lastRefill time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: maximum requests per interval
// interval: time window for rate limit
// burst: maximum burst size (allows temporary spikes)
func NewRateLimiter(rate int, interval time.Duration, burst int) *RateLimiter {
	return &RateLimiter{
		clients:  make(map[interface{}]*bucket),
		rate:     rate,
		interval: interval,
		burst:    burst,
	}
}

// Allow checks if a request from the client should be allowed
func (rl *RateLimiter) Allow(clientID interface{}) bool {
	rl.mu.RLock()
	b, exists := rl.clients[clientID]
	rl.mu.RUnlock()

	if !exists {
		// New client, create bucket
		b = &bucket{
			tokens:     rl.burst,
			lastRefill: time.Now(),
		}
		rl.mu.Lock()
		rl.clients[clientID] = b
		rl.mu.Unlock()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	tokensToAdd := int(elapsed / rl.interval * time.Duration(rl.rate))

	if tokensToAdd > 0 {
		b.tokens += tokensToAdd
		if b.tokens > rl.burst {
			b.tokens = rl.burst
		}
		b.lastRefill = now
	}

	// Check if we have tokens
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	observability.GetMetrics().IncrementErrors()
	observability.Debug("Rate limit exceeded",
		zap.Any("client", clientID),
		zap.Int("tokens", b.tokens),
	)
	return false
}

// Remove removes a client from rate limiting (cleanup)
func (rl *RateLimiter) Remove(clientID interface{}) {
	rl.mu.Lock()
	delete(rl.clients, clientID)
	rl.mu.Unlock()
}

// Stats returns current rate limiter statistics
func (rl *RateLimiter) Stats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"tracked_clients": len(rl.clients),
		"rate":           rl.rate,
		"interval":       rl.interval.String(),
		"burst":          rl.burst,
	}
}

// Cleanup removes clients that haven't been active for the specified duration
func (rl *RateLimiter) Cleanup(maxIdle time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for clientID, b := range rl.clients {
		b.mu.Lock()
		if now.Sub(b.lastRefill) > maxIdle {
			delete(rl.clients, clientID)
		}
		b.mu.Unlock()
	}
}

// StartCleanupRoutine starts a background goroutine to clean up idle clients
func (rl *RateLimiter) StartCleanupRoutine(interval, maxIdle time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.Cleanup(maxIdle)
		case <-done:
			return
		}
	}
}
