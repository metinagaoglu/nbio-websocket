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
	"nbio-websocket/internal/pubsub"
	"nbio-websocket/internal/security"
	"nbio-websocket/internal/transport"
)

func main() {
	cfg := config.LoadConfig()

	if err := cfg.Validate(); err != nil {
		panic("Invalid configuration: " + err.Error())
	}

	if err := observability.InitLogger(&cfg.Log); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer observability.Sync()

	observability.InitMetrics()

	security.InitAuth(cfg.Security.AuthEnabled, cfg.Security.BearerTokens)

	observability.Info("Configuration loaded",
		zap.String("server", cfg.ServerAddr()),
		zap.String("log_level", cfg.Log.Level),
		zap.Int("buffer_size", cfg.Client.SendBufferSize),
		zap.Duration("ping_interval", cfg.Client.PingInterval),
		zap.Bool("auth_enabled", cfg.Security.AuthEnabled),
		zap.Bool("rate_limit_enabled", cfg.Security.RateLimitEnabled),
		zap.Bool("tls_enabled", cfg.Security.TLSEnabled),
		zap.Bool("pubsub_enabled", cfg.PubSub.Enabled),
		zap.String("pubsub_adapter", cfg.PubSub.Adapter),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	hub := core.NewHub()

	if cfg.PubSub.Enabled {
		adapterConfig := map[string]string{}
		switch cfg.PubSub.Adapter {
		case "redis":
			adapterConfig["url"] = cfg.PubSub.RedisURL
		case "nats":
			adapterConfig["url"] = cfg.PubSub.NATSURL
		}

		adapter, err := pubsub.NewAdapter(pubsub.AdapterType(cfg.PubSub.Adapter), adapterConfig)
		if err != nil {
			observability.Fatal("Failed to create PubSub adapter", zap.Error(err))
		}

		if err := hub.SetPubSub(ctx, adapter, cfg.PubSub.Channel); err != nil {
			observability.Fatal("Failed to initialize PubSub", zap.Error(err))
		}

		observability.Info("PubSub scaling enabled",
			zap.String("adapter", cfg.PubSub.Adapter),
			zap.String("channel", cfg.PubSub.Channel))
	}

	go hub.Run(ctx)

	router := protocol.NewRouter()
	router.Register("broadcast", handlers.BroadcastHandler)
	router.Register("self.reply", handlers.SelfReplyHandler)

	go func() {
		metricsAddr := cfg.Server.Host + ":9090"
		if err := transport.StartMetricsServer(metricsAddr); err != nil {
			observability.Error("Metrics server error", zap.Error(err))
		}
	}()

	errChan := make(chan error, 1)
	go func() {
		if err := transport.StartServer(hub, router, cfg); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-sigChan:
		observability.Info("Shutdown signal received, cleaning up...")
		cancel()
		observability.Info("Server stopped gracefully")

	case err := <-errChan:
		observability.Fatal("Server error", zap.Error(err))
	}
}
