package handlers

import (
	"encoding/json"

	"go.uber.org/zap"

	"nbio-websocket/internal/core"
	"nbio-websocket/internal/observability"
	"nbio-websocket/internal/protocol"
)

// BroadcastHandler: Gelen mesajı tüm clientlara iletir.
func BroadcastHandler(client *core.Client, req *protocol.JsonRPCRequest) {
	// Validate params
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		sendErrorResponse(client, protocol.NewErrorResponse(
			req.ID,
			protocol.InvalidParams,
			"Invalid params",
			map[string]string{"expected": "object"},
		))
		return
	}

	// Validate required fields
	text, ok := params["text"].(string)
	if !ok || text == "" {
		sendErrorResponse(client, protocol.NewErrorResponse(
			req.ID,
			protocol.InvalidParams,
			"Invalid params",
			map[string]string{"detail": "text field is required and must be a non-empty string"},
		))
		return
	}

	// Marshal request for broadcasting
	msg, err := json.Marshal(req)
	if err != nil {
		observability.Error("BroadcastHandler: Marshal error", zap.Error(err))
		sendErrorResponse(client, protocol.NewErrorResponse(
			req.ID,
			protocol.InternalError,
			"Internal error",
			map[string]string{"detail": err.Error()},
		))
		return
	}

	// Broadcast to all clients
	client.Hub().Broadcast() <- msg

	// Send success response to sender
	resp := protocol.NewSuccessResponse(req.ID, map[string]interface{}{
		"status":      "broadcasted",
		"text_length": len(text),
	})
	sendSuccessResponse(client, resp)
}

// sendErrorResponse sends an error response to the client
func sendErrorResponse(client *core.Client, resp *protocol.JsonRPCResponse) {
	msg, err := json.Marshal(resp)
	if err != nil {
		observability.Error("Failed to marshal error response", zap.Error(err))
		return
	}
	client.Send(msg)
}

// sendSuccessResponse sends a success response to the client
func sendSuccessResponse(client *core.Client, resp *protocol.JsonRPCResponse) {
	msg, err := json.Marshal(resp)
	if err != nil {
		observability.Error("Failed to marshal success response", zap.Error(err))
		return
	}
	client.Send(msg)
}
