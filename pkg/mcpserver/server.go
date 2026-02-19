// Package mcpserver provides a reusable MCP (Model Context Protocol) server framework.
//
// It extracts the core patterns from npinterface-mcp into a generic, embeddable package
// that supports stdio and HTTP/SSE transports, JSON-RPC 2.0, session management,
// middleware chains, and a clean tool registration interface.
//
// Quick Start:
//
//	server := mcpserver.New("my-server", "1.0.0")
//	server.RegisterTool(&MyTool{})
//	server.RunStdio() // or server.RunHTTP(":8080")
package mcpserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Server is the core MCP server that manages tools and handles JSON-RPC requests.
type Server struct {
	name            string
	version         string
	protocolVersion string
	tools           map[string]ToolHandler
	sessions        map[string]time.Time
	sessionMu       sync.RWMutex
	middleware      []Middleware
	logger          *slog.Logger
}

// New creates a new MCP server with the given name and version.
func New(name, version string) *Server {
	return &Server{
		name:            name,
		version:         version,
		protocolVersion: "2024-11-05",
		tools:           make(map[string]ToolHandler),
		sessions:        make(map[string]time.Time),
		logger:          slog.Default(),
	}
}

// RegisterTool adds a tool to the server.
func (s *Server) RegisterTool(tool ToolHandler) {
	s.tools[tool.Name()] = tool
	s.logger.Info("registered tool", "name", tool.Name())
}

// RegisterTools adds multiple tools to the server.
func (s *Server) RegisterTools(tools ...ToolHandler) {
	for _, tool := range tools {
		s.RegisterTool(tool)
	}
}

// Use adds middleware to the server's processing chain.
func (s *Server) Use(mw Middleware) {
	s.middleware = append(s.middleware, mw)
}

// RunStdio starts the server using stdin/stdout (stdio transport).
func (s *Server) RunStdio() error {
	s.logger.Info("starting MCP server (stdio)", "name", s.name, "version", s.version, "tools", len(s.tools))

	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var req JSONRPCRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode request: %w", err)
		}

		resp := s.HandleRequest(&req)
		if resp == nil {
			continue // Notification, no response needed
		}

		if err := encoder.Encode(resp); err != nil {
			return fmt.Errorf("encode response: %w", err)
		}
	}
	return nil
}

// HandleRequest processes a single JSON-RPC request and returns a response.
func (s *Server) HandleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	// Apply middleware chain
	handler := s.coreHandler
	for i := len(s.middleware) - 1; i >= 0; i-- {
		handler = s.middleware[i](handler)
	}
	return handler(req)
}

func (s *Server) coreHandler(req *JSONRPCRequest) *JSONRPCResponse {
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		resp.Result = s.handleInitialize(req.Params)
	case "notifications/initialized":
		s.logger.Info("client initialized")
		return nil
	case "tools/list":
		resp.Result = s.handleToolsList()
	case "tools/call":
		resp.Result = s.handleToolCall(req.Params)
	default:
		resp.Error = &RPCError{
			Code:    -32601,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		}
	}

	return resp
}

func (s *Server) handleInitialize(params any) *InitializeResult {
	return &InitializeResult{
		ProtocolVersion: s.protocolVersion,
		Capabilities: ServerCapabilities{
			Tools: ToolsCapability{ListChanged: false},
		},
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
		SessionID: s.createSession(),
	}
}

func (s *Server) handleToolsList() *ToolsListResult {
	tools := make([]ToolDef, 0, len(s.tools))
	for _, h := range s.tools {
		tools = append(tools, ToolDef{
			Name:        h.Name(),
			Description: h.Description(),
			InputSchema: h.InputSchema(),
		})
	}
	return &ToolsListResult{Tools: tools}
}

func (s *Server) handleToolCall(params any) any {
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return ErrorResult(fmt.Errorf("parse params: %w", err))
	}

	var callParams struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(paramsBytes, &callParams); err != nil {
		return ErrorResult(fmt.Errorf("unmarshal params: %w", err))
	}

	tool, ok := s.tools[callParams.Name]
	if !ok {
		return ErrorResult(fmt.Errorf("tool not found: %s", callParams.Name))
	}

	result, err := tool.Execute(callParams.Arguments)
	if err != nil {
		return ErrorResult(err)
	}
	return result
}

// Session management

func (s *Server) createSession() string {
	id := generateSessionID()
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.sessions[id] = time.Now()
	return id
}

// CheckSession verifies if a session ID is valid.
func (s *Server) CheckSession(id string) bool {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()
	_, ok := s.sessions[id]
	return ok
}

func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("sess-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
