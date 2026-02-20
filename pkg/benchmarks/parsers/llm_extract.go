package parsers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/benchmarks"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/llm"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/scraper"
)

// LLMExtractor extracts benchmark scores from vendor blog pages using LLM.
// For benchmarks not available on aggregation leaderboards (HLE, Terminal-Bench, etc.)
type LLMExtractor struct {
	llmClient llm.Client
	fetcher   scraper.Fetcher
	models    []benchmarks.ModelConfig
}

// VendorPage represents a vendor evaluation page to extract data from.
type VendorPage struct {
	URL      string
	Provider string
}

// DefaultVendorPages returns the standard set of vendor evaluation pages.
func DefaultVendorPages() []VendorPage {
	return []VendorPage{
		{"https://deepmind.google/models/evals-methodology/gemini-3-1-pro", "google"},
		// Add more vendor pages as they publish evaluation results
	}
}

func NewLLMExtractor(llmClient llm.Client, fetcher scraper.Fetcher, models []benchmarks.ModelConfig) *LLMExtractor {
	return &LLMExtractor{
		llmClient: llmClient,
		fetcher:   fetcher,
		models:    models,
	}
}

func (e *LLMExtractor) Name() string { return "llm-extractor" }

func (e *LLMExtractor) Parse(ctx context.Context, client *http.Client) ([]benchmarks.BenchmarkScore, error) {
	if e.llmClient == nil {
		return nil, fmt.Errorf("LLM client required for vendor page extraction")
	}

	var allScores []benchmarks.BenchmarkScore
	var errors []string

	for _, page := range DefaultVendorPages() {
		scores, err := e.extractFromPage(ctx, page)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", page.URL, err))
			continue
		}
		allScores = append(allScores, scores...)
	}

	if len(errors) > 0 && len(allScores) == 0 {
		return nil, fmt.Errorf("all extractions failed: %s", strings.Join(errors, "; "))
	}
	return allScores, nil
}

func (e *LLMExtractor) extractFromPage(ctx context.Context, page VendorPage) ([]benchmarks.BenchmarkScore, error) {
	// Fetch page content via Jina Reader
	result, err := e.fetcher.Fetch(ctx, page.URL, &scraper.FetchOptions{
		Timeout: 30 * 1e9,
	})
	if err != nil {
		return nil, err
	}

	// Truncate to avoid excessive LLM cost
	content := result.CleanText
	if len(content) > 15000 {
		content = content[:15000]
	}

	// Build benchmark name list for constrained extraction
	var benchNames []string
	for _, b := range benchmarks.AllBenchmarks {
		benchNames = append(benchNames, b.Name)
	}

	prompt := fmt.Sprintf(`Extract AI model benchmark test scores from the following article.

Only extract scores for these benchmarks (ignore others):
%s

Output a JSON array of objects with these fields:
- "benchmark": exact benchmark name from the list above
- "variant": sub-test name if applicable (e.g., "No tools", "Search+Code"), empty string if none
- "model": model name as written in the article
- "score": numeric score value
- "unit": "%%" or "Elo"

If you cannot find any benchmark data, output an empty array [].
Only output valid JSON, no explanation.

Article content:
%s`, strings.Join(benchNames, ", "), content)

	resp, err := e.llmClient.Generate(ctx, &llm.Request{
		System:      "You are a data extraction assistant. Extract structured benchmark data from articles. Output valid JSON only.",
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   4096,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM extraction: %w", err)
	}

	// Parse LLM response
	jsonStr := extractJSONArray(resp.Content)
	var extracted []struct {
		Benchmark string  `json:"benchmark"`
		Variant   string  `json:"variant"`
		Model     string  `json:"model"`
		Score     float64 `json:"score"`
		Unit      string  `json:"unit"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &extracted); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}

	// Convert to BenchmarkScore, matching model names
	var scores []benchmarks.BenchmarkScore
	for _, item := range extracted {
		// Find benchmark ID by name
		benchID := findBenchmarkID(item.Benchmark)
		if benchID == "" {
			continue
		}

		scores = append(scores, benchmarks.BenchmarkScore{
			BenchmarkID:   benchID,
			ModelName:     item.Model,
			ModelProvider: page.Provider,
			Variant:       item.Variant,
			Score:         item.Score,
			SourceURL:     page.URL,
		})
	}

	return scores, nil
}

// findBenchmarkID finds the benchmark ID from its display name.
func findBenchmarkID(name string) string {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	for _, b := range benchmarks.AllBenchmarks {
		if strings.ToLower(b.Name) == nameLower {
			return b.ID
		}
		// Partial match
		if strings.Contains(nameLower, strings.ToLower(b.Name)) ||
			strings.Contains(strings.ToLower(b.Name), nameLower) {
			return b.ID
		}
	}
	return ""
}

// extractJSONArray extracts a JSON array from text that may contain markdown fences.
func extractJSONArray(s string) string {
	s = strings.TrimSpace(s)
	// Remove markdown fences
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}
	// Find array bounds
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return "[]"
}
