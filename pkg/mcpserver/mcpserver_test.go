package mcpserver_test

import (
	"testing"

	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/mcpserver"
)

// EchoTool is a simple tool for testing that echoes back its input.
type EchoTool struct {
	mcpserver.BaseTool
}

func NewEchoTool() *EchoTool {
	return &EchoTool{
		BaseTool: mcpserver.BaseTool{
			ToolName:        "echo",
			ToolDescription: "Echoes back the input message",
			ToolSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message": map[string]any{
						"type":        "string",
						"description": "Message to echo",
					},
				},
				"required": []string{"message"},
			},
		},
	}
}

func (t *EchoTool) Execute(args map[string]any) (*mcpserver.ToolCallResult, error) {
	msg, _ := args["message"].(string)
	return mcpserver.TextResult("Echo: " + msg), nil
}

func TestServer_Initialize(t *testing.T) {
	s := mcpserver.New("test-server", "1.0.0")
	s.RegisterTool(NewEchoTool())

	resp := s.HandleRequest(&mcpserver.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	})

	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	result, ok := resp.Result.(*mcpserver.InitializeResult)
	if !ok {
		t.Fatal("expected InitializeResult")
	}
	if result.ServerInfo.Name != "test-server" {
		t.Fatalf("expected 'test-server', got '%s'", result.ServerInfo.Name)
	}
	if result.SessionID == "" {
		t.Fatal("expected non-empty session ID")
	}
}

func TestServer_ToolsList(t *testing.T) {
	s := mcpserver.New("test-server", "1.0.0")
	s.RegisterTool(NewEchoTool())

	resp := s.HandleRequest(&mcpserver.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	})

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	result, ok := resp.Result.(*mcpserver.ToolsListResult)
	if !ok {
		t.Fatal("expected ToolsListResult")
	}
	if len(result.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result.Tools))
	}
	if result.Tools[0].Name != "echo" {
		t.Fatalf("expected 'echo', got '%s'", result.Tools[0].Name)
	}
}

func TestServer_ToolCall(t *testing.T) {
	s := mcpserver.New("test-server", "1.0.0")
	s.RegisterTool(NewEchoTool())

	resp := s.HandleRequest(&mcpserver.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "echo",
			"arguments": map[string]any{"message": "hello world"},
		},
	})

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(*mcpserver.ToolCallResult)
	if !ok {
		t.Fatal("expected ToolCallResult")
	}
	if result.IsError {
		t.Fatal("expected no error")
	}
	if len(result.Content) != 1 || result.Content[0].Text != "Echo: hello world" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestServer_ToolNotFound(t *testing.T) {
	s := mcpserver.New("test-server", "1.0.0")

	resp := s.HandleRequest(&mcpserver.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "nonexistent",
			"arguments": map[string]any{},
		},
	})

	result, ok := resp.Result.(*mcpserver.ToolCallResult)
	if !ok {
		t.Fatal("expected ToolCallResult")
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
}

func TestServer_MethodNotFound(t *testing.T) {
	s := mcpserver.New("test-server", "1.0.0")

	resp := s.HandleRequest(&mcpserver.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "unknown/method",
	})

	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Fatalf("expected code -32601, got %d", resp.Error.Code)
	}
}

func TestServer_Middleware(t *testing.T) {
	s := mcpserver.New("test-server", "1.0.0")
	s.RegisterTool(NewEchoTool())

	calls := 0
	s.Use(func(next mcpserver.HandlerFunc) mcpserver.HandlerFunc {
		return func(req *mcpserver.JSONRPCRequest) *mcpserver.JSONRPCResponse {
			calls++
			return next(req)
		}
	})

	s.HandleRequest(&mcpserver.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      6,
		Method:  "tools/list",
	})

	if calls != 1 {
		t.Fatalf("expected middleware to be called once, got %d", calls)
	}
}

func TestServer_Session(t *testing.T) {
	s := mcpserver.New("test-server", "1.0.0")

	resp := s.HandleRequest(&mcpserver.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      7,
		Method:  "initialize",
	})

	result := resp.Result.(*mcpserver.InitializeResult)
	if !s.CheckSession(result.SessionID) {
		t.Fatal("expected session to be valid")
	}
	if s.CheckSession("invalid-session") {
		t.Fatal("expected invalid session to fail")
	}
}
