package mcpserver

// ToolHandler is the interface for MCP tools.
// This is the generic version of npinterface-mcp's tools.ToolHandler,
// decoupled from any specific API client.
type ToolHandler interface {
	// Name returns the unique tool name.
	Name() string

	// Description returns a human-readable description.
	Description() string

	// InputSchema returns the JSON Schema for the tool's input.
	InputSchema() map[string]any

	// Execute runs the tool with the given arguments.
	Execute(args map[string]any) (*ToolCallResult, error)
}

// BaseTool provides a base implementation for common tool fields.
// Embed this in your tool structs and implement Execute().
type BaseTool struct {
	ToolName        string
	ToolDescription string
	ToolSchema      map[string]any

	// Metadata for tool discovery / recommendation
	Category string
	Tags     []string
	Keywords []string
}

func (t *BaseTool) Name() string                { return t.ToolName }
func (t *BaseTool) Description() string         { return t.ToolDescription }
func (t *BaseTool) InputSchema() map[string]any { return t.ToolSchema }

// Middleware is a function that wraps a request handler.
type Middleware func(next HandlerFunc) HandlerFunc

// HandlerFunc is a function that handles a JSON-RPC request.
type HandlerFunc func(req *JSONRPCRequest) *JSONRPCResponse
