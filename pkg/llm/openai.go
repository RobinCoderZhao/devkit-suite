package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// openaiClient implements the Client interface for OpenAI-compatible APIs.
type openaiClient struct {
	cfg    Config
	http   *http.Client
	apiKey string
	base   string
}

func newOpenAIClient(cfg Config) (Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	base := "https://api.openai.com/v1"
	if cfg.BaseURL != "" {
		base = cfg.BaseURL
	}
	client := &openaiClient{
		cfg:    cfg,
		apiKey: cfg.APIKey,
		base:   base,
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
	return wrapWithRetry(client, cfg.MaxRetries), nil
}

type openaiRequest struct {
	Model          string                `json:"model"`
	Messages       []openaiMessage       `json:"messages"`
	MaxTokens      int                   `json:"max_tokens,omitempty"`
	Temperature    float64               `json:"temperature,omitempty"`
	ResponseFormat *openaiResponseFormat `json:"response_format,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponseFormat struct {
	Type string `json:"type"`
}

type openaiResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
}

type openaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (c *openaiClient) Generate(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	messages := make([]openaiMessage, 0, len(req.Messages)+1)
	if req.System != "" {
		messages = append(messages, openaiMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		messages = append(messages, openaiMessage{Role: m.Role, Content: m.Content})
	}

	oReq := openaiRequest{
		Model:    c.cfg.Model,
		Messages: messages,
	}

	if req.MaxTokens > 0 {
		oReq.MaxTokens = req.MaxTokens
	} else if c.cfg.MaxTokens > 0 {
		oReq.MaxTokens = c.cfg.MaxTokens
	}

	if req.Temperature > 0 {
		oReq.Temperature = req.Temperature
	} else if c.cfg.Temperature > 0 {
		oReq.Temperature = c.cfg.Temperature
	}

	if req.JSONMode {
		oReq.ResponseFormat = &openaiResponseFormat{Type: "json_object"}
	}

	body, err := json.Marshal(oReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		var errResp openaiErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("OpenAI API error (%d): %s", httpResp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("OpenAI API error (%d): %s", httpResp.StatusCode, string(respBody))
	}

	var oResp openaiResponse
	if err := json.Unmarshal(respBody, &oResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(oResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := oResp.Choices[0].Message.Content

	// Strip <think>...</think> tags from MiniMax responses
	content = stripThinkTags(content)

	latency := time.Since(start).Milliseconds()
	return &Response{
		Content:   content,
		TokensIn:  oResp.Usage.PromptTokens,
		TokensOut: oResp.Usage.CompletionTokens,
		Cost:      EstimateCost(oResp.Model, oResp.Usage.PromptTokens, oResp.Usage.CompletionTokens),
		Model:     oResp.Model,
		LatencyMs: latency,
	}, nil
}

func (c *openaiClient) GenerateJSON(ctx context.Context, req *Request, out any) error {
	req.JSONMode = true
	resp, err := c.Generate(ctx, req)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(resp.Content), out)
}

func (c *openaiClient) Provider() Provider {
	return OpenAI
}

func (c *openaiClient) Close() error {
	return nil
}

// thinkTagRe matches <think>...</think> blocks (including multiline).
var thinkTagRe = regexp.MustCompile(`(?s)<think>.*?</think>`)

// stripThinkTags removes <think>...</think> reasoning blocks from content.
// MiniMax M2.x models include chain-of-thought in <think> tags by default.
func stripThinkTags(content string) string {
	stripped := thinkTagRe.ReplaceAllString(content, "")
	return strings.TrimSpace(stripped)
}
