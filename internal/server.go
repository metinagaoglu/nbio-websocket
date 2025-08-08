package internal

import (
	"log"
	"net/http"
	"time"

	"github.com/lesismal/nbio/nbhttp/websocket"
)

func StartServerWithRouter(hub *Hub, router *Router) error {
	upgrader := websocket.NewUpgrader()

	upgrader.OnOpen(func(c *websocket.Conn) {
		client := NewClient(hub, c)
		hub.register <- client
		go client.WritePump()
		go client.StartPing(10 * time.Second)
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

	log.Println("WebSocket server started on :8080/ws")
	return http.ListenAndServe(":8080", nil)
}
