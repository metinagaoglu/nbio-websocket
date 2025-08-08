package handlers

import (
	"encoding/json"
	"nbio-websocket/internal"
)

// SelfReplyHandler: Sadece mesajı gönderen kullanıcıya yanıt döner.
func SelfReplyHandler(client *internal.Client, req *internal.JsonRPCRequest) {
	resp := internal.JsonRPCResponse{
		JSONRPC: "2.0",
		Result:  map[string]interface{}{"echo": req.Params},
		ID:      req.ID,
	}
	msg, err := json.Marshal(resp)
	if err != nil {
		// TODO: Phase 2 - Send error response to client
		return
	}
	client.Send(msg)
}
