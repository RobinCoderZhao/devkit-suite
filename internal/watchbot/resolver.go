package watchbot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/llm"
)

// Resolver resolves natural language input to monitoring URLs.
// Three-layer fallback: LLM recall → Google Custom Search → Bing Web Search.
type Resolver struct {
	llmClient    llm.Client
	googleAPIKey string
	googleCX     string // Custom Search Engine ID
	bingAPIKey   string
	logger       *slog.Logger
}

// ResolverConfig holds search API credentials.
type ResolverConfig struct {
	GoogleAPIKey string
	GoogleCX     string
	BingAPIKey   string
}

// NewResolver creates a new URL resolver.
func NewResolver(llmClient llm.Client, cfg ResolverConfig) *Resolver {
	return &Resolver{
		llmClient:    llmClient,
		googleAPIKey: cfg.GoogleAPIKey,
		googleCX:     cfg.GoogleCX,
		bingAPIKey:   cfg.BingAPIKey,
		logger:       slog.Default(),
	}
}

// ResolveResult holds the resolved URL info.
type ResolveResult struct {
	Name       string   `json:"name"`
	URLs       []string `json:"urls"`
	PageType   string   `json:"page_type"`
	Confidence string   `json:"confidence"` // "high" or "low"
	Source     string   // "llm", "google", "bing"
	Error      string   `json:"error,omitempty"`
}

// Resolve converts natural language input to monitoring target URLs.
func (r *Resolver) Resolve(ctx context.Context, input string) (*ResolveResult, error) {
	// Layer 1: LLM recall
	r.logger.Info("resolving via LLM", "input", input)
	result, err := r.resolveLLM(ctx, input)
	if err == nil && len(result.URLs) > 0 && result.Confidence == "high" {
		// Verify URL accessibility
		for _, u := range result.URLs {
			vr := ValidateURL(ctx, u)
			if vr.Valid {
				result.Source = "llm"
				return result, nil
			}
			r.logger.Warn("LLM URL failed validation", "url", u, "error", vr.Error)
		}
	}

	// Extract product name for search (from LLM result or input)
	productName := input
	if result != nil && result.Name != "" {
		productName = result.Name
	}

	// Check if LLM returned error (not a monitoring request)
	if result != nil && result.Error != "" {
		return result, nil
	}

	// Layer 2: Google Custom Search
	if r.googleAPIKey != "" && r.googleCX != "" {
		r.logger.Info("resolving via Google", "product", productName)
		searchURL, err := r.searchGoogle(ctx, productName)
		if err == nil && searchURL != "" {
			return &ResolveResult{
				Name:       productName,
				URLs:       []string{searchURL},
				PageType:   GuessPageType(searchURL),
				Confidence: "high",
				Source:     "google",
			}, nil
		}
		if err != nil {
			r.logger.Warn("Google search failed", "error", err)
		}
	}

	// Layer 3: Bing Web Search
	if r.bingAPIKey != "" {
		r.logger.Info("resolving via Bing", "product", productName)
		searchURL, err := r.searchBing(ctx, productName)
		if err == nil && searchURL != "" {
			return &ResolveResult{
				Name:       productName,
				URLs:       []string{searchURL},
				PageType:   GuessPageType(searchURL),
				Confidence: "high",
				Source:     "bing",
			}, nil
		}
		if err != nil {
			r.logger.Warn("Bing search failed", "error", err)
		}
	}

	// All layers failed
	if result != nil && result.Name != "" {
		result.Confidence = "low"
		result.Source = "llm"
		return result, nil
	}

	return &ResolveResult{
		Error: "无法识别监控目标",
	}, nil
}

// resolveLLM uses LLM to recall the URL from training data.
func (r *Resolver) resolveLLM(ctx context.Context, input string) (*ResolveResult, error) {
	if r.llmClient == nil {
		return nil, fmt.Errorf("no LLM client configured")
	}

	prompt := fmt.Sprintf(`你是竞品监控助手。用户想添加一个监控目标。

用户输入："%s"

你的任务：
1. 理解用户想监控哪个产品/公司的什么类型页面
2. 根据你的知识，给出该产品最可能的官方页面 URL
3. 如果你不确定 URL，在 urls 中留空，只返回 name

注意：
- 只返回你有高置信度的 URL，不要编造
- 如果用户输入与监控需求无关（如闲聊），返回 error

返回 JSON（不要包含其他文字）：
成功：{"name": "产品名", "urls": ["URL"], "page_type": "api_docs/pricing/changelog/blog/features", "confidence": "high/low"}
失败：{"error": "无法识别监控目标"}`, input)

	resp, err := r.llmClient.Generate(ctx, &llm.Request{
		System:      "You are a competitor monitoring assistant. Output valid JSON only.",
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   512,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, err
	}

	// Parse JSON from response
	content := strings.TrimSpace(resp.Content)
	// Strip markdown code fences if present
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result ResolveResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}
	return &result, nil
}

// searchGoogle uses Google Custom Search API to find a URL.
func (r *Resolver) searchGoogle(ctx context.Context, productName string) (string, error) {
	query := fmt.Sprintf("%s official documentation site", productName)
	apiURL := fmt.Sprintf("https://www.googleapis.com/customsearch/v1?q=%s&key=%s&cx=%s&num=3",
		url.QueryEscape(query), r.googleAPIKey, r.googleCX)

	return r.fetchSearchResult(ctx, apiURL, "google")
}

// searchBing uses Bing Web Search API to find a URL.
func (r *Resolver) searchBing(ctx context.Context, productName string) (string, error) {
	query := fmt.Sprintf("%s official documentation site", productName)
	apiURL := fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?q=%s&count=3",
		url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", r.bingAPIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("bing API returned %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var bingResp struct {
		WebPages struct {
			Value []struct {
				URL string `json:"url"`
			} `json:"value"`
		} `json:"webPages"`
	}
	if err := json.Unmarshal(body, &bingResp); err != nil {
		return "", err
	}
	if len(bingResp.WebPages.Value) > 0 {
		return bingResp.WebPages.Value[0].URL, nil
	}
	return "", nil
}

// fetchSearchResult performs a GET request and extracts the first URL from Google API response.
func (r *Resolver) fetchSearchResult(ctx context.Context, apiURL, source string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("%s API returned %d", source, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var googleResp struct {
		Items []struct {
			Link string `json:"link"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &googleResp); err != nil {
		return "", err
	}
	if len(googleResp.Items) > 0 {
		return googleResp.Items[0].Link, nil
	}
	return "", nil
}
