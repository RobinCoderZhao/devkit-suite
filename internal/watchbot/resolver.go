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

	"github.com/RobinCoderZhao/devkit-suite/pkg/llm"
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

// TargetSuggestion represents an LLM-filtered result from the Discovery Agent.
type TargetSuggestion struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Confidence int    `json:"confidence"`
	Reasoning  string `json:"reasoning"`
}

// SearchResult holds Bing Web Search raw results.
type SearchResult struct {
	URL     string `json:"url"`
	Name    string `json:"name"`
	Snippet string `json:"snippet"`
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
- 只返回你有绝对高置信度的 URL，不要编造。
- 绝对禁止返回其他类似竞品的URL（例如用户搜 A，你千万不能返回 B 的 URL）。
- 宁可 urls 留空，也不要猜错。
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

// searchBingAdvanced is a robust helper for the discovery agent generating structured results.
func (r *Resolver) searchBingAdvanced(ctx context.Context, query string, count int) ([]SearchResult, error) {
	apiURL := fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?q=%s&count=%d",
		url.QueryEscape(query), count)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", r.bingAPIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bing API returned %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var bingResp struct {
		WebPages struct {
			Value []struct {
				URL     string `json:"url"`
				Name    string `json:"name"`
				Snippet string `json:"snippet"`
			} `json:"value"`
		} `json:"webPages"`
	}
	if err := json.Unmarshal(body, &bingResp); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, v := range bingResp.WebPages.Value {
		results = append(results, SearchResult{
			URL:     v.URL,
			Name:    v.Name,
			Snippet: v.Snippet,
		})
	}
	return results, nil
}

// DiscoverDomainTargets accepts a raw domain name, concurrently spins out multiple precise Bing queries,
// aggregates results, and evaluates commercial value tightly against an LLM.
func (r *Resolver) DiscoverDomainTargets(ctx context.Context, domain string) ([]TargetSuggestion, error) {
	// Sanitize domain
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimSuffix(domain, "/")
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}

	if r.bingAPIKey == "" {
		r.logger.Warn("Bing API key not found, falling back to pure LLM URL guessing for discovery", "domain", domain)
		return r.fallbackLLMDiscovery(ctx, domain)
	}

	queries := []string{
		fmt.Sprintf("site:%s 价格 OR 定价 OR pricing OR prices", domain),
		fmt.Sprintf("site:%s 产品动态 OR 发版说明 OR release notes OR changelog", domain),
		fmt.Sprintf("site:%s API参考 OR 开发者文档 OR API reference", domain),
	}

	type searchRes struct {
		Results []SearchResult
		Err     error
	}

	resChan := make(chan searchRes, len(queries))
	for _, q := range queries {
		go func(query string) {
			r.logger.Info("discovery agent searching bing", "query", query)
			res, err := r.searchBingAdvanced(ctx, query, 4)
			resChan <- searchRes{Results: res, Err: err}
		}(q)
	}

	var allResults []SearchResult
	seenURLs := make(map[string]bool)

	for i := 0; i < len(queries); i++ {
		sr := <-resChan
		if sr.Err != nil {
			r.logger.Warn("bing search error in discovery", "error", sr.Err)
			continue
		}
		for _, res := range sr.Results {
			cleanURL := strings.Split(res.URL, "?")[0]
			cleanURL = strings.Split(cleanURL, "#")[0]

			if !seenURLs[cleanURL] {
				seenURLs[cleanURL] = true
				allResults = append(allResults, res)
			}
		}
	}

	if len(allResults) == 0 {
		return nil, fmt.Errorf("no results found for domain %s", domain)
	}

	// Limit to top 15 candidates to save LLM tokens and keep context manageable
	if len(allResults) > 15 {
		allResults = allResults[:15]
	}

	var itemsBuilder strings.Builder
	for i, res := range allResults {
		itemsBuilder.WriteString(fmt.Sprintf("[%d] URL: %s\nTitle: %s\nSnippet: %s\n\n", i+1, res.URL, res.Name, res.Snippet))
	}

	prompt := fmt.Sprintf(`你是一名严苛的 B2B 商业情报数据分析师。你的任务是审查由爬虫从同一家互联网公司抓取回来的网页列表，精准挑选出真正具备订阅监控价值的商业页面。

要求模型遇到以下特征的页面时，即使看起来相关也一律排除：
- 排除所有的博客文章 (Blog posts)、媒体通稿 (PR News)。
- 排除官网大首页 (homepage)、关于我们、联系我们。
- 排除第三方评测网站、黄牛代理商页面（非官方域名）。
- 排除具体的某一篇单独的 API 文档（我们要的是 API Release Notes 的汇总页或概览目录）。
- 确保所有返回的 URL 都属于域名 %s，不要编造外部链接。

候选网页列表：
%s

请分析上述网页，仅提取出高监控价值的页面。
强制按照以下严格的 JSON 数组格式返回（不要包含任何 Markdown 代码块标签，如 %s，直接输出数组）：
[
  {
    "url": "网址",
    "title": "网页标题",
    "category": "pricer" (注: 只能是 pricer/changelog/docs),
    "confidence": 95,
    "reasoning": "入选理由"
  }
]`, domain, itemsBuilder.String(), "```json")

	resp, err := r.llmClient.Generate(ctx, &llm.Request{
		System:      "Output raw JSON array only, without markdown formatting.",
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   2000,
		Temperature: 0.1,
		JSONMode:    true,
	})
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var suggestions []TargetSuggestion
	if err := json.Unmarshal([]byte(content), &suggestions); err != nil {
		r.logger.Error("failed to parse discovery agent json response", "error", err, "content", content)
		// Try to fallback
		return nil, fmt.Errorf("JSON parse failed: %v", err)
	}

	var finalSuggestions []TargetSuggestion
	for _, sug := range suggestions {
		if sug.Confidence >= 80 && strings.Contains(sug.URL, domain) {
			finalSuggestions = append(finalSuggestions, sug)
		}
	}

	return finalSuggestions, nil
}

// fallbackLLMDiscovery generates domain monitoring targets purely using LLM when no Search APIs are available.
func (r *Resolver) fallbackLLMDiscovery(ctx context.Context, domain string) ([]TargetSuggestion, error) {
	prompt := fmt.Sprintf(`你是一名资深的 B2B 商业情报数据分析师。由于搜索引擎不可用，你需要凭借你的知识，推测并补全目标域名下的监控价值页面。

目标域名：%s

请推测出该域名最可能的 3 个关键监控页面（发版日志 Changelog、定价页面 Pricing、开发者API文档 API Docs）。
请严格按照以下 JSON 数组格式返回（不要包含任何 Markdown 代码块标签，直接输出数组）：
[
  {
    "url": "网址(必须以 https://%s 开头)",
    "title": "网页标题",
    "category": "pricer" // 只能是 pricer/changelog/docs
    "confidence": 80, // 置信度 0-100
    "reasoning": "推测理由"
  }
]`, domain, domain)

	resp, err := r.llmClient.Generate(ctx, &llm.Request{
		System:      "Output raw JSON array only, without markdown formatting.",
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   2000,
		Temperature: 0.1,
		JSONMode:    true,
	})
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var suggestions []TargetSuggestion
	if err := json.Unmarshal([]byte(content), &suggestions); err != nil {
		r.logger.Error("failed to parse fallback LLM discovery response", "error", err, "content", content)
		return nil, fmt.Errorf("JSON parse failed: %v", err)
	}

	return suggestions, nil
}
