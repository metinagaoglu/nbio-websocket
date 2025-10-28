package internal

import (
	"log"
	"net/http"

	"github.com/lesismal/nbio/nbhttp/websocket"
)

func StartServerWithRouter(hub *Hub, router *Router, cfg *Config) error {
	upgrader := websocket.NewUpgrader()

	// Set max message size (nbio uses MessageLengthLimit)
	upgrader.MessageLengthLimit = int(cfg.Client.MaxMessageSize)

	upgrader.OnOpen(func(c *websocket.Conn) {
		client := NewClient(hub, c, cfg.Client.SendBufferSize)
		hub.register <- client
		go client.WritePump()
		go client.StartPing(cfg.Client.PingInterval)
	})

	upgrader.OnMessage(func(c *websocket.Conn, msgType websocket.MessageType, data []byte) {
		if msgType == websocket.TextMessage {
			// ✅ O(1) lookup instead of O(n) iteration
			client, ok := hub.GetClientByConn(c)
			if ok {
				router.Handle(client, data)
			}
		}
	})

	upgrader.OnClose(func(c *websocket.Conn, err error) {
		log.Printf("Client disconnected: %v", err)

		client, ok := hub.GetClientByConn(c)
		if ok {
			hub.unregister <- client
		}
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		_, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Upgrade error: %v", err)
		}
	})

	addr := cfg.ServerAddr()
	log.Printf("WebSocket server starting on %s/ws", addr)
	return http.ListenAndServe(addr, nil)
}
