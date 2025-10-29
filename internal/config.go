package internal

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the WebSocket server
type Config struct {
	Server   ServerConfig
	Client   ClientConfig
	Log      LogConfig
	Security SecurityConfig
}

// ServerConfig contains server-specific settings
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// ClientConfig contains client-specific settings
type ClientConfig struct {
	SendBufferSize int
	PingInterval   time.Duration
	PongTimeout    time.Duration
	MaxMessageSize int64
}

// LogConfig contains logging settings
type LogConfig struct {
	Level  string // debug, info, warn, error
	Format string // text, json
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	AuthEnabled        bool
	BearerTokens       []string
	RateLimitEnabled   bool
	RateLimitRate      int           // requests per interval
	RateLimitInterval  time.Duration // interval for rate limit
	RateLimitBurst     int           // max burst size
	TLSEnabled         bool
	TLSCertFile        string
	TLSKeyFile         string
	MaxMessageSize     int64 // Maximum message size in bytes
	AllowedOrigins     []string
}

// DefaultConfig returns configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
		},
		Client: ClientConfig{
			SendBufferSize: 256,
			PingInterval:   10 * time.Second,
			PongTimeout:    60 * time.Second,
			MaxMessageSize: 1024 * 1024, // 1MB
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		Security: SecurityConfig{
			AuthEnabled:       false,
			BearerTokens:      []string{},
			RateLimitEnabled:  false,
			RateLimitRate:     100,               // 100 requests
			RateLimitInterval: 1 * time.Minute,   // per minute
			RateLimitBurst:    10,                // burst of 10
			TLSEnabled:        false,
			TLSCertFile:       "",
			TLSKeyFile:        "",
			MaxMessageSize:    1024 * 1024,       // 1MB
			AllowedOrigins:    []string{"*"},
		},
	}
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	cfg := DefaultConfig()

	// Server configuration
	if host := os.Getenv("WS_HOST"); host != "" {
		cfg.Server.Host = host
	}

	if portStr := os.Getenv("WS_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.Server.Port = port
		}
	}

	if readTimeoutStr := os.Getenv("WS_READ_TIMEOUT"); readTimeoutStr != "" {
		if timeout, err := time.ParseDuration(readTimeoutStr); err == nil {
			cfg.Server.ReadTimeout = timeout
		}
	}

	if writeTimeoutStr := os.Getenv("WS_WRITE_TIMEOUT"); writeTimeoutStr != "" {
		if timeout, err := time.ParseDuration(writeTimeoutStr); err == nil {
			cfg.Server.WriteTimeout = timeout
		}
	}

	// Client configuration
	if bufferStr := os.Getenv("WS_SEND_BUFFER"); bufferStr != "" {
		if size, err := strconv.Atoi(bufferStr); err == nil {
			cfg.Client.SendBufferSize = size
		}
	}

	if intervalStr := os.Getenv("WS_PING_INTERVAL"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			cfg.Client.PingInterval = interval
		}
	}

	if timeoutStr := os.Getenv("WS_PONG_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			cfg.Client.PongTimeout = timeout
		}
	}

	if sizeStr := os.Getenv("WS_MAX_MESSAGE_SIZE"); sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
			cfg.Client.MaxMessageSize = size
		}
	}

	// Log configuration
	if level := os.Getenv("WS_LOG_LEVEL"); level != "" {
		cfg.Log.Level = level
	}

	if format := os.Getenv("WS_LOG_FORMAT"); format != "" {
		cfg.Log.Format = format
	}

	// Security configuration
	if authEnabledStr := os.Getenv("WS_AUTH_ENABLED"); authEnabledStr != "" {
		cfg.Security.AuthEnabled = authEnabledStr == "true" || authEnabledStr == "1"
	}

	if tokens := os.Getenv("WS_BEARER_TOKENS"); tokens != "" {
		cfg.Security.BearerTokens = []string{}
		for _, token := range strings.Split(tokens, ",") {
			if t := strings.TrimSpace(token); t != "" {
				cfg.Security.BearerTokens = append(cfg.Security.BearerTokens, t)
			}
		}
	}

	if rateLimitEnabledStr := os.Getenv("WS_RATE_LIMIT_ENABLED"); rateLimitEnabledStr != "" {
		cfg.Security.RateLimitEnabled = rateLimitEnabledStr == "true" || rateLimitEnabledStr == "1"
	}

	if rateStr := os.Getenv("WS_RATE_LIMIT_RATE"); rateStr != "" {
		if rate, err := strconv.Atoi(rateStr); err == nil {
			cfg.Security.RateLimitRate = rate
		}
	}

	if intervalStr := os.Getenv("WS_RATE_LIMIT_INTERVAL"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			cfg.Security.RateLimitInterval = interval
		}
	}

	if burstStr := os.Getenv("WS_RATE_LIMIT_BURST"); burstStr != "" {
		if burst, err := strconv.Atoi(burstStr); err == nil {
			cfg.Security.RateLimitBurst = burst
		}
	}

	if tlsEnabledStr := os.Getenv("WS_TLS_ENABLED"); tlsEnabledStr != "" {
		cfg.Security.TLSEnabled = tlsEnabledStr == "true" || tlsEnabledStr == "1"
	}

	if certFile := os.Getenv("WS_TLS_CERT"); certFile != "" {
		cfg.Security.TLSCertFile = certFile
	}

	if keyFile := os.Getenv("WS_TLS_KEY"); keyFile != "" {
		cfg.Security.TLSKeyFile = keyFile
	}

	return cfg
}

// ServerAddr returns the formatted server address
func (c *Config) ServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Server.Port)
	}

	if c.Client.SendBufferSize < 1 {
		return fmt.Errorf("invalid send buffer size: %d (must be > 0)", c.Client.SendBufferSize)
	}

	if c.Client.PingInterval < time.Second {
		return fmt.Errorf("invalid ping interval: %v (must be >= 1s)", c.Client.PingInterval)
	}

	if c.Client.MaxMessageSize < 1024 {
		return fmt.Errorf("invalid max message size: %d (must be >= 1024)", c.Client.MaxMessageSize)
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Log.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Log.Level)
	}

	validFormats := map[string]bool{"text": true, "json": true}
	if !validFormats[c.Log.Format] {
		return fmt.Errorf("invalid log format: %s (must be text or json)", c.Log.Format)
	}

	return nil
}
