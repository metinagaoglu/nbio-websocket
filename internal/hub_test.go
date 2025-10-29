package internal

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestMain initializes logger before running tests
func TestMain(m *testing.M) {
	cfg := &LogConfig{
		Level:  "error", // Minimal logging in tests
		Format: "text",
	}
	if err := InitLogger(cfg); err != nil {
		panic(err)
	}

	code := m.Run()
	Sync()
	os.Exit(code)
}

func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}

	if hub.clients == nil {
		t.Error("Hub clients map not initialized")
	}

	if hub.connToClient == nil {
		t.Error("Hub connToClient map not initialized")
	}

	if hub.broadcast == nil {
		t.Error("Hub broadcast channel not initialized")
	}

	if hub.register == nil {
		t.Error("Hub register channel not initialized")
	}

	if hub.unregister == nil {
		t.Error("Hub unregister channel not initialized")
	}

	if hub.disconnected == nil {
		t.Error("Hub disconnected channel not initialized")
	}
}

func TestHubContextCancellation(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		hub.Run(ctx)
		done <- true
	}()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for hub to stop
	select {
	case <-done:
		// Hub stopped successfully
	case <-time.After(100 * time.Millisecond):
		t.Error("Hub did not stop after context cancellation")
	}
}

func TestHubGetClientByConn(t *testing.T) {
	hub := NewHub()

	// Create a dummy connection (interface{})
	dummyConn := "test-connection-1"

	// Verify non-existent connection returns false
	_, ok := hub.GetClientByConn(dummyConn)
	if ok {
		t.Error("Found client for non-existent connection")
	}

	// Verify empty map state
	hub.mu.RLock()
	if len(hub.connToClient) != 0 {
		t.Error("connToClient map should be empty initially")
	}
	hub.mu.RUnlock()
}

// Note: Full integration tests with actual websocket.Conn require
// either integration test setup or refactoring Client to use interfaces.
// These unit tests cover the Hub's internal logic and data structures.
