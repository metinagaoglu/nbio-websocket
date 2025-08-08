package handlers

import (
	"encoding/json"
	"nbio-websocket/internal"
)

// BroadcastHandler: Gelen mesajı tüm clientlara iletir.
func BroadcastHandler(client *internal.Client, req *internal.JsonRPCRequest) {
	msg, err := json.Marshal(req)
	if err != nil {
		// TODO: Phase 2 - Send error response to client
		return
	}
	client.Hub().Broadcast() <- msg
}
