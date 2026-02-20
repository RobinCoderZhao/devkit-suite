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

// geminiClient implements the Client interface for Google Gemini API.
type geminiClient struct {
	cfg    Config
	http   *http.Client
	apiKey string
	base   string
}

func newGeminiClient(cfg Config) (Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}
	base := "https://generativelanguage.googleapis.com/v1beta"
	if cfg.BaseURL != "" {
		base = cfg.BaseURL
	}
	client := &geminiClient{
		cfg:    cfg,
		apiKey: cfg.APIKey,
		base:   base,
		http:   &http.Client{Timeout: cfg.Timeout},
	}
	return wrapWithRetry(client, cfg.MaxRetries), nil
}

type geminiRequest struct {
	Contents          []geminiContent  `json:"contents"`
	SystemInstruction *geminiContent   `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	MaxOutputTokens  int             `json:"maxOutputTokens,omitempty"`
	Temperature      float64         `json:"temperature,omitempty"`
	ResponseMimeType string          `json:"responseMimeType,omitempty"`
	ThinkingConfig   *thinkingConfig `json:"thinkingConfig,omitempty"`
}

type thinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

func (c *geminiClient) Generate(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	gReq := geminiRequest{}

	if req.System != "" {
		gReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.System}},
		}
	}

	for _, m := range req.Messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		gReq.Contents = append(gReq.Contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	gReq.GenerationConfig = &geminiGenConfig{
		// Disable thinking mode for Gemini 2.5+ models.
		// Without this, thinking tokens consume maxOutputTokens budget,
		// leaving only ~40 tokens for actual content output.
		ThinkingConfig: &thinkingConfig{ThinkingBudget: 0},
	}
	if req.MaxTokens > 0 {
		gReq.GenerationConfig.MaxOutputTokens = req.MaxTokens
	} else if c.cfg.MaxTokens > 0 {
		gReq.GenerationConfig.MaxOutputTokens = c.cfg.MaxTokens
	}
	if req.Temperature > 0 {
		gReq.GenerationConfig.Temperature = req.Temperature
	} else if c.cfg.Temperature > 0 {
		gReq.GenerationConfig.Temperature = c.cfg.Temperature
	}
	if req.JSONMode {
		gReq.GenerationConfig.ResponseMimeType = "application/json"
	}

	body, err := json.Marshal(gReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.base, c.cfg.Model, c.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
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

	var gResp geminiResponse
	if err := json.Unmarshal(respBody, &gResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if gResp.Error != nil {
		return nil, fmt.Errorf("Gemini API error (%d): %s", gResp.Error.Code, gResp.Error.Message)
	}

	if len(gResp.Candidates) == 0 || len(gResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in Gemini response")
	}

	latency := time.Since(start).Milliseconds()
	return &Response{
		Content:      gResp.Candidates[0].Content.Parts[0].Text,
		FinishReason: gResp.Candidates[0].FinishReason,
		TokensIn:     gResp.UsageMetadata.PromptTokenCount,
		TokensOut:    gResp.UsageMetadata.CandidatesTokenCount,
		Cost:         EstimateCost(c.cfg.Model, gResp.UsageMetadata.PromptTokenCount, gResp.UsageMetadata.CandidatesTokenCount),
		Model:        c.cfg.Model,
		LatencyMs:    latency,
	}, nil
}

func (c *geminiClient) GenerateJSON(ctx context.Context, req *Request, out any) error {
	req.JSONMode = true
	resp, err := c.Generate(ctx, req)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(resp.Content), out)
}

func (c *geminiClient) Provider() Provider { return Gemini }
func (c *geminiClient) Close() error       { return nil }
