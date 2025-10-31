package protocol

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"nbio-websocket/internal/core"
	"nbio-websocket/internal/observability"
)

type HandlerFunc func(*core.Client, *JsonRPCRequest)

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
func (r *Router) Handle(client *core.Client, msg []byte) {
	var req JsonRPCRequest
	observability.GetMetrics().IncrementMessagesReceived()

	if err := json.Unmarshal(msg, &req); err != nil {
		observability.GetMetrics().IncrementParseErrors()
		resp := NewErrorResponse(nil, ParseError, "Parse error", map[string]string{
			"detail": err.Error(),
		})
		r.sendResponse(client, resp)
		return
	}

	if req.JSONRPC != "2.0" {
		resp := NewErrorResponse(req.ID, InvalidRequest, "Invalid Request", map[string]string{
			"detail": "jsonrpc version must be '2.0'",
		})
		r.sendResponse(client, resp)
		return
	}

	if req.Method == "" {
		resp := NewErrorResponse(req.ID, InvalidRequest, "Invalid Request", map[string]string{
			"detail": "method is required",
		})
		r.sendResponse(client, resp)
		return
	}

	handler, ok := r.handlers[req.Method]
	if !ok {
		resp := NewErrorResponse(req.ID, MethodNotFound, "Method not found", map[string]string{
			"method": req.Method,
		})
		r.sendResponse(client, resp)
		return
	}

	func() {
		defer func() {
			if panicErr := recover(); panicErr != nil {
				observability.GetMetrics().IncrementHandlerPanics()
				observability.Error("Panic in handler",
					zap.String("method", req.Method),
					zap.Any("panic", panicErr),
				)
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
func (router *Router) sendResponse(client *core.Client, resp *JsonRPCResponse) {
	msg, err := json.Marshal(resp)
	if err != nil {
		observability.Error("Failed to marshal response", zap.Error(err))
		return
	}

	if err := client.Send(msg); err != nil {
		observability.Debug("Failed to send response", zap.Error(err))
	}
}
