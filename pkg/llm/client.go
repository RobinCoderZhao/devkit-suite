// Package llm provides a unified interface for interacting with multiple LLM providers.
// It supports OpenAI, Gemini, Claude, and Ollama with automatic retries and cost tracking.
package llm

import (
	"context"
	"fmt"
	"time"
)

// Provider represents an LLM provider.
type Provider string

const (
	OpenAI  Provider = "openai"
	Gemini  Provider = "gemini"
	Claude  Provider = "claude"
	Ollama  Provider = "ollama"
	MiniMax Provider = "minimax"
)

// Config holds configuration for an LLM client.
type Config struct {
	Provider    Provider      `yaml:"provider" json:"provider"`
	Model       string        `yaml:"model" json:"model"`
	APIKey      string        `yaml:"api_key" json:"api_key"`
	BaseURL     string        `yaml:"base_url" json:"base_url"`
	MaxRetries  int           `yaml:"max_retries" json:"max_retries"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
	MaxTokens   int           `yaml:"max_tokens" json:"max_tokens"`
	Temperature float64       `yaml:"temperature" json:"temperature"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Provider:    OpenAI,
		Model:       "gpt-4o-mini",
		MaxRetries:  3,
		Timeout:     30 * time.Second,
		MaxTokens:   4096,
		Temperature: 0.7,
	}
}

// Client is the unified interface for LLM interactions.
type Client interface {
	// Generate sends a prompt and returns the LLM response.
	Generate(ctx context.Context, req *Request) (*Response, error)

	// GenerateJSON sends a prompt and unmarshals the JSON response into out.
	GenerateJSON(ctx context.Context, req *Request, out any) error

	// Provider returns the name of the provider.
	Provider() Provider

	// Close releases any resources held by the client.
	Close() error
}

// Message represents a single message in a conversation.
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// Request holds the parameters for an LLM generation request.
type Request struct {
	System      string    `json:"system,omitempty"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	JSONMode    bool      `json:"json_mode,omitempty"`
}

// Response holds the result of an LLM generation.
type Response struct {
	Content      string  `json:"content"`
	FinishReason string  `json:"finish_reason,omitempty"` // "STOP", "MAX_TOKENS", "SAFETY", etc.
	TokensIn     int     `json:"tokens_in"`
	TokensOut    int     `json:"tokens_out"`
	Cost         float64 `json:"cost"`
	Model        string  `json:"model"`
	LatencyMs    int64   `json:"latency_ms"`
}

// NewClient creates a new LLM client based on the provided config.
func NewClient(cfg Config) (Client, error) {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	switch cfg.Provider {
	case OpenAI:
		return newOpenAIClient(cfg)
	case Gemini:
		return newGeminiClient(cfg)
	case Claude:
		return newClaudeClient(cfg)
	case Ollama:
		return newOllamaClient(cfg)
	case MiniMax:
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.minimax.io/v1"
		}
		return newOpenAIClient(cfg)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}

// SimpleGenerate is a convenience function for quick one-shot generation.
func SimpleGenerate(ctx context.Context, cfg Config, prompt string) (string, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return "", fmt.Errorf("create client: %w", err)
	}
	defer client.Close()

	resp, err := client.Generate(ctx, &Request{
		Messages: []Message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
