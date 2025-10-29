package protocol

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"nbio-websocket/internal/core"
	"nbio-websocket/internal/observability"
)

// TestMain initializes logger before running tests
func TestMain(m *testing.M) {
	cfg := &observability.LogConfig{
		Level:  "error", // Minimal logging in tests
		Format: "text",
	}
	if err := observability.InitLogger(cfg); err != nil {
		panic(err)
	}

	code := m.Run()
	observability.Sync()
	os.Exit(code)
}

func TestNewRouter(t *testing.T) {
	router := NewRouter()

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}

	if router.handlers == nil {
		t.Error("Router handlers map not initialized")
	}
}

func TestRouterRegisterHandler(t *testing.T) {
	router := NewRouter()

	handler := func(client *core.Client, req *JsonRPCRequest) {
		// Test handler
	}

	router.Register("test", handler)

	// Verify handler was registered
	if _, exists := router.handlers["test"]; !exists {
		t.Error("Handler not registered")
	}
}

func TestRouterHandleInvalidJSON(t *testing.T) {
	router := NewRouter()
	hub := core.NewHub()
	client := core.NewTestClient(hub, 1)

	// Send invalid JSON
	router.Handle(client, []byte("{invalid json"))

	// Should receive error response
	select {
	case response := <-client.GetSendChannel():
		var resp JsonRPCResponse
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}

		if resp.Error == nil {
			t.Error("Expected error response, got success")
		}

		if resp.Error.Code != ParseError {
			t.Errorf("Wrong error code: got %d, want %d", resp.Error.Code, ParseError)
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("No response received")
	}
}

func TestRouterHandleInvalidVersion(t *testing.T) {
	router := NewRouter()
	hub := core.NewHub()
	client := core.NewTestClient(hub, 1)

	// Send request with wrong JSON-RPC version
	req := map[string]interface{}{
		"jsonrpc": "1.0",
		"method":  "test",
		"id":      1,
	}
	msg, _ := json.Marshal(req)

	router.Handle(client, msg)

	// Should receive error response
	select {
	case response := <-client.GetSendChannel():
		var resp JsonRPCResponse
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}

		if resp.Error == nil {
			t.Error("Expected error response, got success")
		}

		if resp.Error.Code != InvalidRequest {
			t.Errorf("Wrong error code: got %d, want %d", resp.Error.Code, InvalidRequest)
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("No response received")
	}
}

func TestRouterHandleEmptyMethod(t *testing.T) {
	router := NewRouter()
	hub := core.NewHub()
	client := core.NewTestClient(hub, 1)

	// Send request with empty method
	req := JsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "",
		ID:      1,
	}
	msg, _ := json.Marshal(req)

	router.Handle(client, msg)

	// Should receive error response
	select {
	case response := <-client.GetSendChannel():
		var resp JsonRPCResponse
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}

		if resp.Error == nil {
			t.Error("Expected error response, got success")
		}

		if resp.Error.Code != InvalidRequest {
			t.Errorf("Wrong error code: got %d, want %d", resp.Error.Code, InvalidRequest)
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("No response received")
	}
}

func TestRouterHandleMethodNotFound(t *testing.T) {
	router := NewRouter()
	hub := core.NewHub()
	client := core.NewTestClient(hub, 1)

	// Send request with unregistered method
	req := JsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "nonexistent",
		ID:      1,
	}
	msg, _ := json.Marshal(req)

	router.Handle(client, msg)

	// Should receive error response
	select {
	case response := <-client.GetSendChannel():
		var resp JsonRPCResponse
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}

		if resp.Error == nil {
			t.Error("Expected error response, got success")
		}

		if resp.Error.Code != MethodNotFound {
			t.Errorf("Wrong error code: got %d, want %d", resp.Error.Code, MethodNotFound)
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("No response received")
	}
}

func TestRouterHandleSuccess(t *testing.T) {
	router := NewRouter()
	hub := core.NewHub()
	client := core.NewTestClient(hub, 1)

	// Register test handler
	handlerCalled := false
	router.Register("test", func(c *core.Client, req *JsonRPCRequest) {
		handlerCalled = true

		// Send success response
		resp := NewSuccessResponse(req.ID, map[string]string{"result": "ok"})
		c.SendJSON(resp)
	})

	// Send valid request
	req := JsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "test",
		ID:      1,
	}
	msg, _ := json.Marshal(req)

	router.Handle(client, msg)

	// Verify handler was called
	time.Sleep(10 * time.Millisecond)
	if !handlerCalled {
		t.Error("Handler was not called")
	}

	// Should receive success response
	select {
	case response := <-client.GetSendChannel():
		var resp JsonRPCResponse
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Error != nil {
			t.Errorf("Expected success response, got error: %v", resp.Error)
		}

		if resp.Result == nil {
			t.Error("Success response has nil result")
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("No response received")
	}
}

func TestRouterHandlePanic(t *testing.T) {
	router := NewRouter()
	hub := core.NewHub()
	client := core.NewTestClient(hub, 1)

	// Register handler that panics
	router.Register("panic", func(c *core.Client, req *JsonRPCRequest) {
		panic("test panic")
	})

	// Send request
	req := JsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "panic",
		ID:      1,
	}
	msg, _ := json.Marshal(req)

	router.Handle(client, msg)

	// Should receive error response (panic recovered)
	select {
	case response := <-client.GetSendChannel():
		var resp JsonRPCResponse
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}

		if resp.Error == nil {
			t.Error("Expected error response after panic, got success")
		}

		if resp.Error.Code != InternalError {
			t.Errorf("Wrong error code: got %d, want %d", resp.Error.Code, InternalError)
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("No response received after panic")
	}
}
