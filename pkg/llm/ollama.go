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

// ollamaClient implements the Client interface for local Ollama models.
type ollamaClient struct {
	cfg  Config
	http *http.Client
	base string
}

func newOllamaClient(cfg Config) (Client, error) {
	base := "http://localhost:11434"
	if cfg.BaseURL != "" {
		base = cfg.BaseURL
	}
	if cfg.Model == "" {
		cfg.Model = "llama3.2"
	}
	client := &ollamaClient{
		cfg:  cfg,
		base: base,
		http: &http.Client{Timeout: cfg.Timeout},
	}
	// No retry for local models
	return client, nil
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   string          `json:"format,omitempty"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaResponse struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done            bool  `json:"done"`
	TotalDuration   int64 `json:"total_duration"`
	PromptEvalCount int   `json:"prompt_eval_count"`
	EvalCount       int   `json:"eval_count"`
}

func (c *ollamaClient) Generate(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	messages := make([]ollamaMessage, 0, len(req.Messages)+1)
	if req.System != "" {
		messages = append(messages, ollamaMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		messages = append(messages, ollamaMessage{Role: m.Role, Content: m.Content})
	}

	oReq := ollamaRequest{
		Model:    c.cfg.Model,
		Messages: messages,
		Stream:   false,
	}

	if req.JSONMode {
		oReq.Format = "json"
	}

	opts := &ollamaOptions{}
	if req.Temperature > 0 {
		opts.Temperature = req.Temperature
	} else if c.cfg.Temperature > 0 {
		opts.Temperature = c.cfg.Temperature
	}
	if req.MaxTokens > 0 {
		opts.NumPredict = req.MaxTokens
	} else if c.cfg.MaxTokens > 0 {
		opts.NumPredict = c.cfg.MaxTokens
	}
	oReq.Options = opts

	body, err := json.Marshal(oReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.base+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama error (%d): %s", httpResp.StatusCode, string(respBody))
	}

	var oResp ollamaResponse
	if err := json.Unmarshal(respBody, &oResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	latency := time.Since(start).Milliseconds()
	return &Response{
		Content:   oResp.Message.Content,
		TokensIn:  oResp.PromptEvalCount,
		TokensOut: oResp.EvalCount,
		Cost:      0, // Local models are free
		Model:     c.cfg.Model,
		LatencyMs: latency,
	}, nil
}

func (c *ollamaClient) GenerateJSON(ctx context.Context, req *Request, out any) error {
	req.JSONMode = true
	resp, err := c.Generate(ctx, req)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(resp.Content), out)
}

func (c *ollamaClient) Provider() Provider { return Ollama }
func (c *ollamaClient) Close() error       { return nil }
