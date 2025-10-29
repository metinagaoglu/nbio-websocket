package protocol

import (
	"encoding/json"
	"testing"
)

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse(1, ParseError, "Parse error", map[string]string{"detail": "invalid JSON"})

	if resp == nil {
		t.Fatal("NewErrorResponse returned nil")
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Wrong JSONRPC version: got %s, want 2.0", resp.JSONRPC)
	}

	if resp.Error == nil {
		t.Fatal("Error field is nil")
	}

	if resp.Error.Code != ParseError {
		t.Errorf("Wrong error code: got %d, want %d", resp.Error.Code, ParseError)
	}

	if resp.Error.Message != "Parse error" {
		t.Errorf("Wrong error message: got %s, want 'Parse error'", resp.Error.Message)
	}

	if resp.ID != 1 {
		t.Errorf("Wrong ID: got %v, want 1", resp.ID)
	}
}

func TestNewSuccessResponse(t *testing.T) {
	result := map[string]string{"status": "ok"}
	resp := NewSuccessResponse(1, result)

	if resp == nil {
		t.Fatal("NewSuccessResponse returned nil")
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Wrong JSONRPC version: got %s, want 2.0", resp.JSONRPC)
	}

	if resp.Result == nil {
		t.Fatal("Result field is nil")
	}

	if resp.Error != nil {
		t.Error("Error field should be nil for success response")
	}

	if resp.ID != 1 {
		t.Errorf("Wrong ID: got %v, want 1", resp.ID)
	}
}

func TestErrorResponseMarshaling(t *testing.T) {
	resp := NewErrorResponse(1, MethodNotFound, "Method not found", map[string]string{"method": "unknown"})

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal error response: %v", err)
	}

	var decoded JsonRPCResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if decoded.JSONRPC != "2.0" {
		t.Error("JSONRPC version lost in marshal/unmarshal")
	}

	if decoded.Error == nil {
		t.Fatal("Error field is nil after unmarshal")
	}

	if decoded.Error.Code != MethodNotFound {
		t.Errorf("Error code changed: got %d, want %d", decoded.Error.Code, MethodNotFound)
	}
}

func TestSuccessResponseMarshaling(t *testing.T) {
	result := map[string]interface{}{
		"status": "success",
		"count":  42,
	}
	resp := NewSuccessResponse("test-id", result)

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal success response: %v", err)
	}

	var decoded JsonRPCResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal success response: %v", err)
	}

	if decoded.Result == nil {
		t.Fatal("Result field is nil after unmarshal")
	}

	resultMap, ok := decoded.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map after unmarshal")
	}

	if resultMap["status"] != "success" {
		t.Error("Result data lost in marshal/unmarshal")
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		code int
		name string
	}{
		{ParseError, "ParseError"},
		{InvalidRequest, "InvalidRequest"},
		{MethodNotFound, "MethodNotFound"},
		{InvalidParams, "InvalidParams"},
		{InternalError, "InternalError"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewErrorResponse(1, tt.code, tt.name, nil)
			if resp.Error.Code != tt.code {
				t.Errorf("Error code mismatch: got %d, want %d", resp.Error.Code, tt.code)
			}
		})
	}
}
