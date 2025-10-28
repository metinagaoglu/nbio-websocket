package internal

import (
	"encoding/json"
	"fmt"
	"log"
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

// Handle processes incoming JSON-RPC messages with comprehensive validation
func (r *Router) Handle(client *Client, msg []byte) {
	var req JsonRPCRequest

	// Step 1: Parse JSON
	if err := json.Unmarshal(msg, &req); err != nil {
		resp := NewErrorResponse(nil, ParseError, "Parse error", map[string]string{
			"detail": err.Error(),
		})
		r.sendResponse(client, resp)
		return
	}

	// Step 2: Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		resp := NewErrorResponse(req.ID, InvalidRequest, "Invalid Request", map[string]string{
			"detail": "jsonrpc version must be '2.0'",
		})
		r.sendResponse(client, resp)
		return
	}

	// Step 3: Validate method
	if req.Method == "" {
		resp := NewErrorResponse(req.ID, InvalidRequest, "Invalid Request", map[string]string{
			"detail": "method is required",
		})
		r.sendResponse(client, resp)
		return
	}

	// Step 4: Find handler
	handler, ok := r.handlers[req.Method]
	if !ok {
		resp := NewErrorResponse(req.ID, MethodNotFound, "Method not found", map[string]string{
			"method": req.Method,
		})
		r.sendResponse(client, resp)
		return
	}

	// Step 5: Execute handler with panic recovery
	func() {
		defer func() {
			if panicErr := recover(); panicErr != nil {
				log.Printf("Panic in handler '%s': %v", req.Method, panicErr)
				resp := NewErrorResponse(req.ID, InternalError, "Internal error", map[string]interface{}{
					"panic": fmt.Sprintf("%v", panicErr),
				})
				r.sendResponse(client, resp)
			}
		}()

		handler(client, &req)
	}()
}

// sendResponse marshals and sends a JSON-RPC response to the client
func (router *Router) sendResponse(client *Client, resp *JsonRPCResponse) {
	msg, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	if err := client.Send(msg); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}
