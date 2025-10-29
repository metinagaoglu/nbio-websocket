package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"nbio-websocket/internal/config"
	"nbio-websocket/internal/core"
	"nbio-websocket/internal/handlers"
	"nbio-websocket/internal/observability"
	"nbio-websocket/internal/protocol"
	"nbio-websocket/internal/security"
	"nbio-websocket/internal/transport"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		panic("Invalid configuration: " + err.Error())
	}

	// Initialize logger
	if err := observability.InitLogger(&cfg.Log); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer observability.Sync()

	// Initialize metrics
	observability.InitMetrics()

	// Initialize authentication
	security.InitAuth(cfg.Security.AuthEnabled, cfg.Security.BearerTokens)

	observability.Info("Configuration loaded",
		zap.String("server", cfg.ServerAddr()),
		zap.String("log_level", cfg.Log.Level),
		zap.Int("buffer_size", cfg.Client.SendBufferSize),
		zap.Duration("ping_interval", cfg.Client.PingInterval),
		zap.Bool("auth_enabled", cfg.Security.AuthEnabled),
		zap.Bool("rate_limit_enabled", cfg.Security.RateLimitEnabled),
		zap.Bool("tls_enabled", cfg.Security.TLSEnabled),
	)

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create hub and start with context
	hub := core.NewHub()
	go hub.Run(ctx)

	// Setup router
	router := protocol.NewRouter()
	router.Register("broadcast", handlers.BroadcastHandler)
	router.Register("self.reply", handlers.SelfReplyHandler)

	// Start metrics server in goroutine
	go func() {
		metricsAddr := cfg.Server.Host + ":9090"
		if err := transport.StartMetricsServer(metricsAddr); err != nil {
			observability.Error("Metrics server error", zap.Error(err))
		}
	}()

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := transport.StartServer(hub, router, cfg); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		observability.Info("Shutdown signal received, cleaning up...")
		cancel() // Cancel context to stop hub
		observability.Info("Server stopped gracefully")

	case err := <-errChan:
		observability.Fatal("Server error", zap.Error(err))
	}
}
