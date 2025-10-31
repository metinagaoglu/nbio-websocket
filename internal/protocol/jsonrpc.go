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

const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
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
