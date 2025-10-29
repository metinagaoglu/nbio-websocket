package handlers

import (
	"encoding/json"

	"go.uber.org/zap"

	"nbio-websocket/internal/core"
	"nbio-websocket/internal/observability"
	"nbio-websocket/internal/protocol"
)

// SelfReplyHandler: Sadece mesajı gönderen kullanıcıya yanıt döner.
func SelfReplyHandler(client *core.Client, req *protocol.JsonRPCRequest) {
	// Validate params
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		resp := protocol.NewErrorResponse(
			req.ID,
			protocol.InvalidParams,
			"Invalid params",
			map[string]string{"expected": "object"},
		)
		sendErrorResponse(client, resp)
		return
	}

	// Validate required fields
	text, ok := params["text"].(string)
	if !ok || text == "" {
		resp := protocol.NewErrorResponse(
			req.ID,
			protocol.InvalidParams,
			"Invalid params",
			map[string]string{"detail": "text field is required and must be a non-empty string"},
		)
		sendErrorResponse(client, resp)
		return
	}

	// Send echo response
	resp := protocol.NewSuccessResponse(req.ID, map[string]interface{}{
		"echo":        text,
		"text_length": len(text),
	})

	msg, err := json.Marshal(resp)
	if err != nil {
		observability.Error("SelfReplyHandler: Marshal error", zap.Error(err))
		errResp := protocol.NewErrorResponse(
			req.ID,
			protocol.InternalError,
			"Internal error",
			map[string]string{"detail": err.Error()},
		)
		sendErrorResponse(client, errResp)
		return
	}

	client.Send(msg)
}
