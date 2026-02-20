package parsers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/RobinCoderZhao/devkit-suite/pkg/benchmarks"
	"github.com/RobinCoderZhao/devkit-suite/pkg/scraper"
)

// LLMStatsParser fetches benchmark data from llm-stats.com.
// Each benchmark has a dedicated page with a leaderboard table.
type LLMStatsParser struct {
	fetcher scraper.Fetcher
	models  []benchmarks.ModelConfig
}

// llm-stats.com benchmark URLs
var llmStatsURLs = map[string]string{
	"gpqa_diamond":       "https://llm-stats.com/benchmarks/gpqa",
	"swe_bench_verified": "https://llm-stats.com/benchmarks/swe-bench-verified",
	"mmmlu":              "https://llm-stats.com/benchmarks/mmlu",
	"mmmu_pro":           "https://llm-stats.com/benchmarks/mmmu",
	"livecodebench_pro":  "https://llm-stats.com/benchmarks/livecodebench",
	"arc_agi_2":          "https://llm-stats.com/benchmarks/arc-agi",
}

// NewLLMStatsParser creates a parser for llm-stats.com.
func NewLLMStatsParser(fetcher scraper.Fetcher, models []benchmarks.ModelConfig) *LLMStatsParser {
	return &LLMStatsParser{fetcher: fetcher, models: models}
}

func (p *LLMStatsParser) Name() string { return "llm-stats.com" }

func (p *LLMStatsParser) Parse(ctx context.Context, client *http.Client) ([]benchmarks.BenchmarkScore, error) {
	var allScores []benchmarks.BenchmarkScore
	var errors []string

	for benchID, url := range llmStatsURLs {
		scores, err := p.parseBenchmarkPage(ctx, benchID, url)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", benchID, err))
			continue
		}
		allScores = append(allScores, scores...)
	}

	if len(errors) > 0 && len(allScores) == 0 {
		return nil, fmt.Errorf("all pages failed: %s", strings.Join(errors, "; "))
	}
	return allScores, nil
}

func (p *LLMStatsParser) parseBenchmarkPage(ctx context.Context, benchID, url string) ([]benchmarks.BenchmarkScore, error) {
	// Fetch via Jina Reader (JS-rendered pages)
	result, err := p.fetcher.Fetch(ctx, url, &scraper.FetchOptions{
		Timeout: 30 * 1e9, // 30 seconds
	})
	if err != nil {
		return nil, err
	}

	content := result.CleanText
	if content == "" {
		return nil, fmt.Errorf("empty content from %s", url)
	}

	// Extract markdown table
	rows := ExtractMarkdownTable(content)
	if len(rows) < 2 {
		return nil, fmt.Errorf("no table found in %s", url)
	}

	// Find the score column index
	header := rows[0]
	scoreCol := findScoreColumn(header)
	modelCol := findModelColumn(header)

	if scoreCol < 0 || modelCol < 0 {
		// Fallback: assume col 0 = model, col 1 = score
		modelCol = 0
		scoreCol = 1
	}

	// Parse data rows
	var scores []benchmarks.BenchmarkScore
	for _, row := range rows[1:] {
		if modelCol >= len(row) || scoreCol >= len(row) {
			continue
		}

		rawModelName := row[modelCol]
		model, found := MatchModelName(rawModelName, p.models)
		if !found {
			continue // Skip models not in our tracking list
		}

		score, ok := ParseScore(row[scoreCol])
		if !ok {
			continue
		}

		scores = append(scores, benchmarks.BenchmarkScore{
			BenchmarkID:   benchID,
			ModelName:     model.Name,
			ModelProvider: model.Provider,
			Score:         score,
			SourceURL:     url,
		})
	}

	return scores, nil
}

// findScoreColumn finds the column index containing scores.
func findScoreColumn(header []string) int {
	scoreKeywords := []string{"score", "accuracy", "pass", "resolved", "elo", "rating", "%"}
	for i, h := range header {
		lower := strings.ToLower(h)
		for _, kw := range scoreKeywords {
			if strings.Contains(lower, kw) {
				return i
			}
		}
	}
	return -1
}

// findModelColumn finds the column index containing model names.
func findModelColumn(header []string) int {
	modelKeywords := []string{"model", "name", "system", "agent"}
	for i, h := range header {
		lower := strings.ToLower(h)
		for _, kw := range modelKeywords {
			if strings.Contains(lower, kw) {
				return i
			}
		}
	}
	return -1
}
