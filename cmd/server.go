package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"nbio-websocket/internal"
	"nbio-websocket/internal/handlers"
)

func main() {
	// ✅ Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ✅ Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create hub and start with context
	hub := internal.NewHub()
	go hub.Run(ctx)

	// Setup router
	router := internal.NewRouter()
	router.Register("broadcast", handlers.BroadcastHandler)
	router.Register("self.reply", handlers.SelfReplyHandler)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Println("Starting WebSocket server on :8080/ws")
		if err := internal.StartServerWithRouter(hub, router); err != nil {
			errChan <- err
		}
	}()

	// ✅ Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Println("Shutdown signal received, cleaning up...")
		cancel() // Cancel context to stop hub
		log.Println("Server stopped gracefully")

	case err := <-errChan:
		log.Fatalf("Server error: %v", err)
	}
}
