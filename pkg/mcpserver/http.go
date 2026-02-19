package mcpserver

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// HTTPServer wraps the MCP Server to serve over HTTP with SSE support.
type HTTPServer struct {
	server    *Server
	addr      string
	authToken string
	logger    *slog.Logger
}

// RunHTTP starts the MCP server on an HTTP endpoint.
func (s *Server) RunHTTP(addr string) error {
	hs := &HTTPServer{
		server: s,
		addr:   addr,
		logger: s.logger,
	}
	return hs.ListenAndServe()
}

// SetHTTPAuthToken sets a Bearer token for HTTP authentication.
func (s *Server) SetHTTPAuthToken(token string) {
	// Will be used by RunHTTP
}

// ListenAndServe starts the HTTP server.
func (hs *HTTPServer) ListenAndServe() error {
	mux := http.NewServeMux()

	// MCP protocol endpoint (JSON-RPC 2.0)
	mux.HandleFunc("/mcp", hs.handleMCPRequest)

	// RESTful endpoints
	mux.HandleFunc("/api/tools", hs.handleToolsList)
	mux.HandleFunc("/api/tools/", hs.handleToolCall)

	// Health check
	mux.HandleFunc("/health", hs.handleHealth)

	hs.logger.Info("starting HTTP server", "addr", hs.addr, "tools", len(hs.server.tools))

	return http.ListenAndServe(hs.addr, hs.corsMiddleware(mux))
}

func (hs *HTTPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Session-Id")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (hs *HTTPServer) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.writeError(w, -32700, "Parse error")
		return
	}

	// Validate session for non-initialize requests
	if req.Method != "initialize" {
		sessionID := r.Header.Get("Mcp-Session-Id")
		if sessionID == "" || !hs.server.CheckSession(sessionID) {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
	}

	resp := hs.server.HandleRequest(&req)

	// Set session ID header for initialize response
	if req.Method == "initialize" && resp != nil && resp.Error == nil {
		if result, ok := resp.Result.(*InitializeResult); ok && result.SessionID != "" {
			w.Header().Set("Mcp-Session-Id", result.SessionID)
		}
	}

	// Choose response format based on Accept header
	if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		hs.sendSSE(w, resp)
	} else {
		hs.sendJSON(w, resp)
	}
}

func (hs *HTTPServer) sendJSON(w http.ResponseWriter, resp *JSONRPCResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (hs *HTTPServer) sendSSE(w http.ResponseWriter, resp *JSONRPCResponse) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		hs.sendJSON(w, resp)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", "/mcp")
	flusher.Flush()

	respBytes, _ := json.Marshal(resp)
	fmt.Fprintf(w, "data: %s\n\n", string(respBytes))
	flusher.Flush()
}

func (hs *HTTPServer) handleToolsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := hs.server.handleToolsList()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (hs *HTTPServer) handleToolCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	toolName := strings.TrimPrefix(r.URL.Path, "/api/tools/")
	if toolName == "" {
		http.Error(w, "Tool name required", http.StatusBadRequest)
		return
	}

	var args map[string]any
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	callParams := map[string]any{
		"name":      toolName,
		"arguments": args,
	}
	result := hs.server.handleToolCall(callParams)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (hs *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"server":    hs.server.name,
		"version":   hs.server.version,
	})
}

func (hs *HTTPServer) writeError(w http.ResponseWriter, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Error:   &RPCError{Code: code, Message: message},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
