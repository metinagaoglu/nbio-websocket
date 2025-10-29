package transport

import (
	"net/http"

	"go.uber.org/zap"

	"nbio-websocket/internal/config"
	"nbio-websocket/internal/core"
	"nbio-websocket/internal/observability"
	"nbio-websocket/internal/protocol"
	"nbio-websocket/internal/security"

	"github.com/lesismal/nbio/nbhttp/websocket"
)

// Routes holds all HTTP route handlers
type Routes struct {
	hub         *core.Hub
	router      *protocol.Router
	upgrader    *websocket.Upgrader
	rateLimiter *security.RateLimiter
	config      *config.Config
}

// NewRoutes creates a new Routes instance with all dependencies
func NewRoutes(hub *core.Hub, router *protocol.Router, cfg *config.Config) *Routes {
	upgrader := websocket.NewUpgrader()
	upgrader.MessageLengthLimit = int(cfg.Security.MaxMessageSize)

	var rateLimiter *security.RateLimiter
	if cfg.Security.RateLimitEnabled {
		rateLimiter = security.NewRateLimiter(
			cfg.Security.RateLimitRate,
			cfg.Security.RateLimitInterval,
			cfg.Security.RateLimitBurst,
		)
		observability.Info("Rate limiting enabled",
			zap.Int("rate", cfg.Security.RateLimitRate),
			zap.Duration("interval", cfg.Security.RateLimitInterval),
			zap.Int("burst", cfg.Security.RateLimitBurst),
		)
	}

	return &Routes{
		hub:         hub,
		router:      router,
		upgrader:    upgrader,
		rateLimiter: rateLimiter,
		config:      cfg,
	}
}

// RegisterRoutes registers all HTTP routes
func (r *Routes) RegisterRoutes(mux *http.ServeMux) {
	// WebSocket endpoint
	mux.HandleFunc("/ws", r.handleWebSocket)

	// Health check endpoint
	mux.HandleFunc("/health", r.handleHealth)

	// Metrics endpoint
	mux.HandleFunc("/metrics", observability.MetricsHandler)

	observability.Info("HTTP routes registered",
		zap.Strings("endpoints", []string{"/ws", "/health", "/metrics"}),
	)
}

// handleWebSocket handles WebSocket upgrade and lifecycle
func (r *Routes) handleWebSocket(w http.ResponseWriter, req *http.Request) {
	// Authentication check
	if security.GlobalAuth != nil && security.GlobalAuth.IsEnabled() {
		if err := security.GlobalAuth.ValidateRequest(req); err != nil {
			observability.Error("Authentication failed",
				zap.Error(err),
				zap.String("remote_addr", req.RemoteAddr),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Setup WebSocket callbacks before upgrade
	r.upgrader.OnOpen(func(c *websocket.Conn) {
		client := core.NewClient(r.hub, c, r.config.Client.SendBufferSize)
		r.hub.Register(client)
		go client.WritePump()
		go client.StartPing(r.config.Client.PingInterval)
	})

	r.upgrader.OnMessage(func(c *websocket.Conn, msgType websocket.MessageType, data []byte) {
		if msgType == websocket.TextMessage {
			client, ok := r.hub.GetClientByConn(c)
			if !ok {
				return
			}

			// Rate limiting check
			if r.rateLimiter != nil && !r.rateLimiter.Allow(c) {
				resp := protocol.NewErrorResponse(nil, protocol.InternalError, "Rate limit exceeded", map[string]string{
					"retry_after": r.config.Security.RateLimitInterval.String(),
				})
				client.SendJSON(resp)
				return
			}

			r.router.Handle(client, data)
		}
	})

	r.upgrader.OnClose(func(c *websocket.Conn, err error) {
		if err != nil {
			observability.Debug("Client disconnected with error", zap.Error(err))
		} else {
			observability.Debug("Client disconnected")
		}

		client, ok := r.hub.GetClientByConn(c)
		if ok {
			r.hub.Unregister(client)
		}
	})

	// Upgrade to WebSocket
	_, err := r.upgrader.Upgrade(w, req, nil)
	if err != nil {
		observability.Error("WebSocket upgrade failed",
			zap.Error(err),
			zap.String("remote_addr", req.RemoteAddr),
		)
		return
	}
}

// handleHealth provides health check endpoint
func (r *Routes) handleHealth(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	observability.HealthHandler(w, req)
}
