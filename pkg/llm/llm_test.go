package llm

import (
	"context"
	"testing"
)

func TestNewClient_InvalidProvider(t *testing.T) {
	_, err := NewClient(Config{Provider: "invalid", APIKey: "test"})
	if err == nil {
		t.Fatal("expected error for invalid provider")
	}
}

func TestNewClient_MissingAPIKey(t *testing.T) {
	for _, p := range []Provider{OpenAI, Gemini, Claude} {
		_, err := NewClient(Config{Provider: p})
		if err == nil {
			t.Fatalf("expected error for %s without API key", p)
		}
	}
}

func TestNewClient_Ollama(t *testing.T) {
	client, err := NewClient(Config{Provider: Ollama, BaseURL: "http://localhost:11434"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.Provider() != Ollama {
		t.Fatalf("expected Ollama provider, got %s", client.Provider())
	}
	client.Close()
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Provider != OpenAI {
		t.Fatalf("expected OpenAI, got %s", cfg.Provider)
	}
	if cfg.Model != "gpt-4o-mini" {
		t.Fatalf("expected gpt-4o-mini, got %s", cfg.Model)
	}
}

func TestEstimateCost(t *testing.T) {
	cost := EstimateCost("gpt-4o-mini", 1000, 500)
	if cost <= 0 {
		t.Fatalf("expected positive cost, got %f", cost)
	}
	// gpt-4o-mini: $0.15/1M in, $0.60/1M out
	// 1000 in = 0.00015, 500 out = 0.0003 => total ~0.00045
	expected := 0.00015 + 0.0003
	if cost < expected*0.9 || cost > expected*1.1 {
		t.Fatalf("cost %f not in expected range around %f", cost, expected)
	}
}

func TestEstimateCost_UnknownModel(t *testing.T) {
	cost := EstimateCost("unknown-model", 1000, 500)
	if cost != 0 {
		t.Fatalf("expected 0 cost for unknown model, got %f", cost)
	}
}

// TestRetryClient_NoRetryOnSuccess verifies no retry happens on success.
func TestRetryClient_NoRetryOnSuccess(t *testing.T) {
	calls := 0
	mock := &mockClient{
		generateFn: func(ctx context.Context, req *Request) (*Response, error) {
			calls++
			return &Response{Content: "hello"}, nil
		},
	}
	rc := wrapWithRetry(mock, 3)
	resp, err := rc.Generate(context.Background(), &Request{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hello" {
		t.Fatalf("expected 'hello', got '%s'", resp.Content)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

type mockClient struct {
	generateFn func(ctx context.Context, req *Request) (*Response, error)
}

func (m *mockClient) Generate(ctx context.Context, req *Request) (*Response, error) {
	return m.generateFn(ctx, req)
}
func (m *mockClient) GenerateJSON(ctx context.Context, req *Request, out any) error {
	return nil
}
func (m *mockClient) Provider() Provider { return "mock" }
func (m *mockClient) Close() error       { return nil }

func TestNewClient_MiniMax(t *testing.T) {
	client, err := NewClient(Config{Provider: MiniMax, APIKey: "test-key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer client.Close()
	// MiniMax reuses OpenAI client, reports as OpenAI provider
	if client.Provider() != OpenAI {
		t.Fatalf("expected OpenAI provider (via MiniMax), got %s", client.Provider())
	}
}

func TestEstimateCost_MiniMax(t *testing.T) {
	cost := EstimateCost("MiniMax-M2.5", 1000, 500)
	if cost <= 0 {
		t.Fatalf("expected positive cost for MiniMax-M2.5, got %f", cost)
	}
}

func TestStripThinkTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no tags", "Hello world", "Hello world"},
		{"with think tags", "<think>reasoning here</think>Actual response", "Actual response"},
		{"multiline think", "<think>\nstep 1\nstep 2\n</think>\nFinal answer", "Final answer"},
		{"empty content", "", ""},
		{"only think", "<think>only thinking</think>", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripThinkTags(tt.input)
			if got != tt.expected {
				t.Errorf("stripThinkTags(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
