package internal

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"
)

var (
	ErrInvalidToken      = errors.New("invalid authentication token")
	ErrMissingToken      = errors.New("missing authentication token")
	ErrAuthNotConfigured = errors.New("authentication not configured")
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled      bool
	BearerTokens map[string]bool // Set of valid bearer tokens
	mu           sync.RWMutex
}

// GlobalAuth is the global authentication instance
var GlobalAuth *AuthConfig

// InitAuth initializes the global authentication system
func InitAuth(enabled bool, tokens []string) {
	GlobalAuth = &AuthConfig{
		Enabled:      enabled,
		BearerTokens: make(map[string]bool),
	}

	for _, token := range tokens {
		if token != "" {
			GlobalAuth.BearerTokens[token] = true
		}
	}

	if enabled {
		Info("Authentication enabled",
			zap.Int("token_count", len(GlobalAuth.BearerTokens)),
		)
	} else {
		Info("Authentication disabled")
	}
}

// ValidateRequest validates HTTP request authentication
func (ac *AuthConfig) ValidateRequest(r *http.Request) error {
	if !ac.Enabled {
		return nil
	}

	// Extract authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		// Check query parameter as fallback
		token := r.URL.Query().Get("token")
		if token == "" {
			Debug("Missing authentication token")
			return ErrMissingToken
		}
		return ac.ValidateToken(token)
	}

	// Parse Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		Debug("Invalid authorization header format")
		return ErrInvalidToken
	}

	return ac.ValidateToken(parts[1])
}

// ValidateToken validates a bearer token
func (ac *AuthConfig) ValidateToken(token string) error {
	if !ac.Enabled {
		return nil
	}

	if token == "" {
		return ErrMissingToken
	}

	ac.mu.RLock()
	defer ac.mu.RUnlock()

	// Use constant-time comparison to prevent timing attacks
	for validToken := range ac.BearerTokens {
		if subtle.ConstantTimeCompare([]byte(token), []byte(validToken)) == 1 {
			return nil
		}
	}

	Debug("Invalid authentication token attempted")
	GetMetrics().IncrementErrors()
	return ErrInvalidToken
}

// AddToken adds a new valid token (thread-safe)
func (ac *AuthConfig) AddToken(token string) {
	if token == "" {
		return
	}

	ac.mu.Lock()
	ac.BearerTokens[token] = true
	ac.mu.Unlock()

	Info("Authentication token added")
}

// RemoveToken removes a token (thread-safe)
func (ac *AuthConfig) RemoveToken(token string) {
	ac.mu.Lock()
	delete(ac.BearerTokens, token)
	ac.mu.Unlock()

	Info("Authentication token removed")
}

// IsEnabled returns whether authentication is enabled
func (ac *AuthConfig) IsEnabled() bool {
	return ac.Enabled
}

// TokenCount returns the number of valid tokens
func (ac *AuthConfig) TokenCount() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return len(ac.BearerTokens)
}
