package protocol

// JsonRPCRequest represents a JSON-RPC 2.0 request
type JsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

// JsonRPCResponse represents a JSON-RPC 2.0 response
type JsonRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JsonRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id"`
}

// JsonRPCError represents a JSON-RPC 2.0 error object
type JsonRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes
const (
	ParseError     = -32700 // Invalid JSON
	InvalidRequest = -32600 // Invalid Request object
	MethodNotFound = -32601 // Method does not exist
	InvalidParams  = -32602 // Invalid method parameters
	InternalError  = -32603 // Internal JSON-RPC error
)

// NewErrorResponse creates a new error response
func NewErrorResponse(id interface{}, code int, message string, data interface{}) *JsonRPCResponse {
	return &JsonRPCResponse{
		JSONRPC: "2.0",
		Error: &JsonRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(id interface{}, result interface{}) *JsonRPCResponse {
	return &JsonRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
}
