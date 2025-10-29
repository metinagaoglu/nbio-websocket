package transport

import (
	"net/http"

	"go.uber.org/zap"

	"nbio-websocket/internal/config"
	"nbio-websocket/internal/core"
	"nbio-websocket/internal/observability"
	"nbio-websocket/internal/protocol"
)

// StartServer initializes and starts the WebSocket server with clean routing
func StartServer(hub *core.Hub, router *protocol.Router, cfg *config.Config) error {
	// Create routes
	routes := NewRoutes(hub, router, cfg)

	// Create HTTP multiplexer
	mux := http.NewServeMux()

	// Register all routes
	routes.RegisterRoutes(mux)

	// Start server
	addr := cfg.ServerAddr()
	observability.Info("WebSocket server starting",
		zap.String("address", addr),
		zap.Bool("auth_enabled", cfg.Security.AuthEnabled),
		zap.Bool("rate_limit_enabled", cfg.Security.RateLimitEnabled),
	)

	return http.ListenAndServe(addr, mux)
}

// StartMetricsServer starts a separate HTTP server for metrics
func StartMetricsServer(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", observability.MetricsHandler)
	mux.HandleFunc("/health", observability.HealthHandler)

	observability.Info("Metrics server starting", zap.String("address", addr))
	return http.ListenAndServe(addr, mux)
}
