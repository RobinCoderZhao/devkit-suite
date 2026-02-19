package mcpserver

import "log/slog"

// LoggingMiddleware logs all incoming requests and their results.
func LoggingMiddleware(logger *slog.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(req *JSONRPCRequest) *JSONRPCResponse {
			logger.Info("mcp request", "method", req.Method, "id", req.ID)
			resp := next(req)
			if resp != nil && resp.Error != nil {
				logger.Error("mcp error", "method", req.Method, "code", resp.Error.Code, "message", resp.Error.Message)
			}
			return resp
		}
	}
}

// RecoveryMiddleware catches panics and returns a JSON-RPC error.
func RecoveryMiddleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(req *JSONRPCRequest) (resp *JSONRPCResponse) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic in MCP handler", "method", req.Method, "panic", r)
					resp = &JSONRPCResponse{
						JSONRPC: "2.0",
						ID:      req.ID,
						Error: &RPCError{
							Code:    -32603,
							Message: "Internal error",
						},
					}
				}
			}()
			return next(req)
		}
	}
}
