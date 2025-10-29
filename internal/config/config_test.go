package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Server defaults
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Wrong default host: got %s, want 0.0.0.0", cfg.Server.Host)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Wrong default port: got %d, want 8080", cfg.Server.Port)
	}

	// Client defaults
	if cfg.Client.SendBufferSize != 256 {
		t.Errorf("Wrong default buffer size: got %d, want 256", cfg.Client.SendBufferSize)
	}

	if cfg.Client.PingInterval != 10*time.Second {
		t.Errorf("Wrong default ping interval: got %v, want 10s", cfg.Client.PingInterval)
	}

	// Log defaults
	if cfg.Log.Level != "info" {
		t.Errorf("Wrong default log level: got %s, want info", cfg.Log.Level)
	}

	if cfg.Log.Format != "text" {
		t.Errorf("Wrong default log format: got %s, want text", cfg.Log.Format)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("WS_HOST", "127.0.0.1")
	os.Setenv("WS_PORT", "9090")
	os.Setenv("WS_SEND_BUFFER", "512")
	os.Setenv("WS_LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("WS_HOST")
		os.Unsetenv("WS_PORT")
		os.Unsetenv("WS_SEND_BUFFER")
		os.Unsetenv("WS_LOG_LEVEL")
	}()

	cfg := LoadConfig()

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Failed to load host from env: got %s, want 127.0.0.1", cfg.Server.Host)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Failed to load port from env: got %d, want 9090", cfg.Server.Port)
	}

	if cfg.Client.SendBufferSize != 512 {
		t.Errorf("Failed to load buffer size from env: got %d, want 512", cfg.Client.SendBufferSize)
	}

	if cfg.Log.Level != "debug" {
		t.Errorf("Failed to load log level from env: got %s, want debug", cfg.Log.Level)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "invalid port (too low)",
			modify: func(c *Config) {
				c.Server.Port = 0
			},
			wantErr: true,
		},
		{
			name: "invalid port (too high)",
			modify: func(c *Config) {
				c.Server.Port = 70000
			},
			wantErr: true,
		},
		{
			name: "invalid buffer size",
			modify: func(c *Config) {
				c.Client.SendBufferSize = 0
			},
			wantErr: true,
		},
		{
			name: "invalid ping interval",
			modify: func(c *Config) {
				c.Client.PingInterval = 500 * time.Millisecond
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			modify: func(c *Config) {
				c.Log.Level = "invalid"
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			modify: func(c *Config) {
				c.Log.Format = "invalid"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerAddr(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 8080

	addr := cfg.ServerAddr()
	expected := "127.0.0.1:8080"

	if addr != expected {
		t.Errorf("ServerAddr() = %s, want %s", addr, expected)
	}
}
