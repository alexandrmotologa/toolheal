package middleware

import (
	"encoding/json"
	"fmt"
)

// JSONRPCRequest models a standard JSON-RPC 2.0 request used by MCP and language servers.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse models a standard JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id,omitempty"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError defines standard JSON-RPC error codes.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// NewJSONRPCError creates a structured error object.
func NewJSONRPCError(code int, msg string, data any) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: msg,
			Data:    data,
		},
	}
}

// MCPToolCallParams models the parameters sent for method: "tools/call".
type MCPToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ParseMCPToolCall extracts the tool name and raw argument map from JSON-RPC params.
func ParseMCPToolCall(raw json.RawMessage) (string, map[string]any, error) {
	var callParams MCPToolCallParams
	if err := json.Unmarshal(raw, &callParams); err != nil {
		return "", nil, fmt.Errorf("failed to parse tools/call params: %w", err)
	}

	var args map[string]any
	if len(callParams.Arguments) > 0 {
		if err := json.Unmarshal(callParams.Arguments, &args); err != nil {
			return "", nil, fmt.Errorf("invalid json arguments in tool call: %w", err)
		}
	} else {
		args = make(map[string]any)
	}

	return callParams.Name, args, nil
}
