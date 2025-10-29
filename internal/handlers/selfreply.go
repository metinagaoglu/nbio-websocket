package handlers

import (
	"encoding/json"

	"go.uber.org/zap"

	"nbio-websocket/internal"
)

// SelfReplyHandler: Sadece mesajı gönderen kullanıcıya yanıt döner.
func SelfReplyHandler(client *internal.Client, req *internal.JsonRPCRequest) {
	// Validate params
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		resp := internal.NewErrorResponse(
			req.ID,
			internal.InvalidParams,
			"Invalid params",
			map[string]string{"expected": "object"},
		)
		sendErrorResponse(client, resp)
		return
	}

	// Validate required fields
	text, ok := params["text"].(string)
	if !ok || text == "" {
		resp := internal.NewErrorResponse(
			req.ID,
			internal.InvalidParams,
			"Invalid params",
			map[string]string{"detail": "text field is required and must be a non-empty string"},
		)
		sendErrorResponse(client, resp)
		return
	}

	// Send echo response
	resp := internal.NewSuccessResponse(req.ID, map[string]interface{}{
		"echo":        text,
		"text_length": len(text),
	})

	msg, err := json.Marshal(resp)
	if err != nil {
		internal.Error("SelfReplyHandler: Marshal error", zap.Error(err))
		errResp := internal.NewErrorResponse(
			req.ID,
			internal.InternalError,
			"Internal error",
			map[string]string{"detail": err.Error()},
		)
		sendErrorResponse(client, errResp)
		return
	}

	client.Send(msg)
}
