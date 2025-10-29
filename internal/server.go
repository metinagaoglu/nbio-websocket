package internal

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/lesismal/nbio/nbhttp/websocket"
)

func StartServerWithRouter(hub *Hub, router *Router, cfg *Config) error {
	upgrader := websocket.NewUpgrader()

	// Set max message size (nbio uses MessageLengthLimit)
	upgrader.MessageLengthLimit = int(cfg.Security.MaxMessageSize)

	// Initialize rate limiter if enabled
	var rateLimiter *RateLimiter
	if cfg.Security.RateLimitEnabled {
		rateLimiter = NewRateLimiter(
			cfg.Security.RateLimitRate,
			cfg.Security.RateLimitInterval,
			cfg.Security.RateLimitBurst,
		)
		Info("Rate limiting enabled",
			zap.Int("rate", cfg.Security.RateLimitRate),
			zap.Duration("interval", cfg.Security.RateLimitInterval),
			zap.Int("burst", cfg.Security.RateLimitBurst),
		)
	}

	upgrader.OnOpen(func(c *websocket.Conn) {
		client := NewClient(hub, c, cfg.Client.SendBufferSize)
		hub.register <- client
		go client.WritePump()
		go client.StartPing(cfg.Client.PingInterval)
	})

	upgrader.OnMessage(func(c *websocket.Conn, msgType websocket.MessageType, data []byte) {
		if msgType == websocket.TextMessage {
			// ✅ O(1) lookup instead of O(n) iteration
			client, ok := hub.GetClientByConn(c)
			if !ok {
				return
			}

			// Rate limiting check
			if rateLimiter != nil && !rateLimiter.Allow(c) {
				resp := NewErrorResponse(nil, InternalError, "Rate limit exceeded", map[string]string{
					"retry_after": cfg.Security.RateLimitInterval.String(),
				})
				client.SendJSON(resp)
				return
			}

			router.Handle(client, data)
		}
	})

	upgrader.OnClose(func(c *websocket.Conn, err error) {
		if err != nil {
			Debug("Client disconnected with error", zap.Error(err))
		} else {
			Debug("Client disconnected")
		}

		client, ok := hub.GetClientByConn(c)
		if ok {
			hub.unregister <- client
		}
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Authentication check
		if GlobalAuth != nil && GlobalAuth.IsEnabled() {
			if err := GlobalAuth.ValidateRequest(r); err != nil {
				Error("Authentication failed",
					zap.Error(err),
					zap.String("remote_addr", r.RemoteAddr),
				)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		_, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			Error("WebSocket upgrade failed",
				zap.Error(err),
				zap.String("remote_addr", r.RemoteAddr),
			)
		}
	})

	addr := cfg.ServerAddr()
	Info("WebSocket server starting", zap.String("address", addr))
	return http.ListenAndServe(addr, nil)
}
