package mcpserver

import "encoding/json"

// JSON-RPC 2.0 protocol types

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCP protocol types

// InitializeResult is the response to an initialize request.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	SessionID       string             `json:"sessionId,omitempty"`
}

// ServerCapabilities describes the server's supported features.
type ServerCapabilities struct {
	Tools ToolsCapability `json:"tools"`
}

// ToolsCapability describes the tools capability.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// ServerInfo describes the server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolDef represents a tool definition for listing.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// ToolsListResult is the result of a tools/list request.
type ToolsListResult struct {
	Tools []ToolDef `json:"tools"`
}

// ToolCallResult is the standard result from executing a tool.
type ToolCallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents a piece of tool output.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data any    `json:"data,omitempty"`
}

// SuccessResult creates a successful ToolCallResult from any data.
func SuccessResult(data any) *ToolCallResult {
	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ErrorResult(err)
	}
	return &ToolCallResult{
		Content: []Content{{Type: "text", Text: string(dataJSON)}},
	}
}

// TextResult creates a ToolCallResult with a plain text message.
func TextResult(text string) *ToolCallResult {
	return &ToolCallResult{
		Content: []Content{{Type: "text", Text: text}},
	}
}

// ErrorResult creates an error ToolCallResult.
func ErrorResult(err error) *ToolCallResult {
	return &ToolCallResult{
		Content: []Content{{Type: "text", Text: err.Error()}},
		IsError: true,
	}
}
