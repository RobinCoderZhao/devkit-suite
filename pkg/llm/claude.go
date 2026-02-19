package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// claudeClient implements the Client interface for Anthropic Claude API.
type claudeClient struct {
	cfg    Config
	http   *http.Client
	apiKey string
	base   string
}

func newClaudeClient(cfg Config) (Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Claude API key is required")
	}
	base := "https://api.anthropic.com/v1"
	if cfg.BaseURL != "" {
		base = cfg.BaseURL
	}
	client := &claudeClient{
		cfg:    cfg,
		apiKey: cfg.APIKey,
		base:   base,
		http:   &http.Client{Timeout: cfg.Timeout},
	}
	return wrapWithRetry(client, cfg.MaxRetries), nil
}

type claudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system,omitempty"`
	Messages    []claudeMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *claudeClient) Generate(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	messages := make([]claudeMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role != "system" {
			messages = append(messages, claudeMessage{Role: m.Role, Content: m.Content})
		}
	}

	maxTokens := c.cfg.MaxTokens
	if req.MaxTokens > 0 {
		maxTokens = req.MaxTokens
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	cReq := claudeRequest{
		Model:     c.cfg.Model,
		MaxTokens: maxTokens,
		System:    req.System,
		Messages:  messages,
	}

	if req.Temperature > 0 {
		cReq.Temperature = req.Temperature
	} else if c.cfg.Temperature > 0 {
		cReq.Temperature = c.cfg.Temperature
	}

	body, err := json.Marshal(cReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.base+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var cResp claudeResponse
	if err := json.Unmarshal(respBody, &cResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if cResp.Error != nil {
		return nil, fmt.Errorf("Claude API error (%s): %s", cResp.Error.Type, cResp.Error.Message)
	}

	if len(cResp.Content) == 0 {
		return nil, fmt.Errorf("no content in Claude response")
	}

	var text string
	for _, block := range cResp.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}

	latency := time.Since(start).Milliseconds()
	return &Response{
		Content:   text,
		TokensIn:  cResp.Usage.InputTokens,
		TokensOut: cResp.Usage.OutputTokens,
		Cost:      EstimateCost(cResp.Model, cResp.Usage.InputTokens, cResp.Usage.OutputTokens),
		Model:     cResp.Model,
		LatencyMs: latency,
	}, nil
}

func (c *claudeClient) GenerateJSON(ctx context.Context, req *Request, out any) error {
	req.JSONMode = true
	if req.System == "" {
		req.System = "Always respond with valid JSON only."
	} else {
		req.System += "\n\nAlways respond with valid JSON only."
	}
	resp, err := c.Generate(ctx, req)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(resp.Content), out)
}

func (c *claudeClient) Provider() Provider { return Claude }
func (c *claudeClient) Close() error       { return nil }
