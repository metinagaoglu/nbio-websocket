package internal

import (
	"encoding/json"
)

type HandlerFunc func(*Client, *JsonRPCRequest)

type Router struct {
	handlers map[string]HandlerFunc
}

func NewRouter() *Router {
	return &Router{handlers: make(map[string]HandlerFunc)}
}

func (r *Router) Register(event string, handler HandlerFunc) {
	r.handlers[event] = handler
}

func (r *Router) Handle(client *Client, msg []byte) {
	var req JsonRPCRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return
	}
	if handler, ok := r.handlers[req.Method]; ok {
		handler(client, &req)
	}
}
// Handler fonksiyonları internal/handlers klasöründe olacak.