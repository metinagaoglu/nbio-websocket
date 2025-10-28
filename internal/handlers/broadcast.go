package handlers

import (
	"encoding/json"
	"log"
	"nbio-websocket/internal"
)

// BroadcastHandler: Gelen mesajı tüm clientlara iletir.
func BroadcastHandler(client *internal.Client, req *internal.JsonRPCRequest) {
	// Validate params
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		sendErrorResponse(client, internal.NewErrorResponse(
			req.ID,
			internal.InvalidParams,
			"Invalid params",
			map[string]string{"expected": "object"},
		))
		return
	}

	// Validate required fields
	text, ok := params["text"].(string)
	if !ok || text == "" {
		sendErrorResponse(client, internal.NewErrorResponse(
			req.ID,
			internal.InvalidParams,
			"Invalid params",
			map[string]string{"detail": "text field is required and must be a non-empty string"},
		))
		return
	}

	// Marshal request for broadcasting
	msg, err := json.Marshal(req)
	if err != nil {
		log.Printf("BroadcastHandler: Marshal error: %v", err)
		sendErrorResponse(client, internal.NewErrorResponse(
			req.ID,
			internal.InternalError,
			"Internal error",
			map[string]string{"detail": err.Error()},
		))
		return
	}

	// Broadcast to all clients
	client.Hub().Broadcast() <- msg

	// Send success response to sender
	resp := internal.NewSuccessResponse(req.ID, map[string]interface{}{
		"status":      "broadcasted",
		"text_length": len(text),
	})
	sendSuccessResponse(client, resp)
}

// sendErrorResponse sends an error response to the client
func sendErrorResponse(client *internal.Client, resp *internal.JsonRPCResponse) {
	msg, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal error response: %v", err)
		return
	}
	client.Send(msg)
}

// sendSuccessResponse sends a success response to the client
func sendSuccessResponse(client *internal.Client, resp *internal.JsonRPCResponse) {
	msg, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal success response: %v", err)
		return
	}
	client.Send(msg)
}
