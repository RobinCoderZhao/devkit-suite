package benchmarks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Scraper coordinates benchmark data collection from multiple sources.
type Scraper struct {
	store   *Store
	client  *http.Client
	parsers []Parser
}

// Parser extracts benchmark scores from a data source.
type Parser interface {
	Name() string
	Parse(ctx context.Context, client *http.Client) ([]BenchmarkScore, error)
}

// NewScraper creates a scraper with the given store and parsers.
func NewScraper(store *Store, parsers ...Parser) *Scraper {
	return &Scraper{
		store:   store,
		client:  &http.Client{Timeout: 30 * time.Second},
		parsers: parsers,
	}
}

// ScrapeAll runs all parsers and stores the results. Returns total new/updated scores.
func (s *Scraper) ScrapeAll(ctx context.Context) (int, error) {
	total := 0
	var errors []string

	for _, p := range s.parsers {
		scores, err := p.Parse(ctx, s.client)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", p.Name(), err))
			continue
		}
		if len(scores) > 0 {
			if err := s.store.BulkUpsert(ctx, scores); err != nil {
				errors = append(errors, fmt.Sprintf("%s store: %v", p.Name(), err))
				continue
			}
			total += len(scores)
		}
	}

	if len(errors) > 0 {
		return total, fmt.Errorf("scrape errors (%d scores saved): %s", total, strings.Join(errors, "; "))
	}
	return total, nil
}

// FetchURL is a helper that fetches a URL and returns the body.
func FetchURL(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "WatchBot-BenchmarkTracker/1.0")
	req.Header.Set("Accept", "text/html,application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}
	return string(body), nil
}

// ---- Manual Data Entry Parser ----

// ManualParser allows injecting scores from code (e.g., from user's screenshot data).
// This is useful for bootstrapping the database or testing.
type ManualParser struct {
	scores []BenchmarkScore
}

func NewManualParser(scores []BenchmarkScore) *ManualParser {
	return &ManualParser{scores: scores}
}

func (p *ManualParser) Name() string { return "manual" }

func (p *ManualParser) Parse(ctx context.Context, client *http.Client) ([]BenchmarkScore, error) {
	return p.scores, nil
}

// SeedFromScreenshot returns scores from the Gemini 3.1 Pro evaluation table
// that the user shared (Google DeepMind methodology page).
func SeedFromScreenshot() []BenchmarkScore {
	type entry struct {
		bench, variant, model, provider string
		score                           float64
	}
	data := []entry{
		// HLE - No tools
		{"hle", "No tools", "Gemini 3.1 Pro", "google", 44.4},
		{"hle", "No tools", "Gemini 3 Pro", "google", 37.5},
		{"hle", "No tools", "Sonnet 4.6", "anthropic", 33.2},
		{"hle", "No tools", "Opus 4.6", "anthropic", 40.0},
		{"hle", "No tools", "GPT-5.2", "openai", 34.5},
		// HLE - Search+Code
		{"hle", "Search+Code", "Gemini 3.1 Pro", "google", 51.4},
		{"hle", "Search+Code", "Gemini 3 Pro", "google", 45.8},
		{"hle", "Search+Code", "Sonnet 4.6", "anthropic", 49.0},
		{"hle", "Search+Code", "Opus 4.6", "anthropic", 53.1},
		{"hle", "Search+Code", "GPT-5.2", "openai", 45.5},
		// ARC-AGI-2
		{"arc_agi_2", "", "Gemini 3.1 Pro", "google", 77.1},
		{"arc_agi_2", "", "Gemini 3 Pro", "google", 31.1},
		{"arc_agi_2", "", "Sonnet 4.6", "anthropic", 58.3},
		{"arc_agi_2", "", "Opus 4.6", "anthropic", 68.8},
		{"arc_agi_2", "", "GPT-5.2", "openai", 52.9},
		// GPQA Diamond
		{"gpqa_diamond", "", "Gemini 3.1 Pro", "google", 94.3},
		{"gpqa_diamond", "", "Gemini 3 Pro", "google", 91.9},
		{"gpqa_diamond", "", "Sonnet 4.6", "anthropic", 89.9},
		{"gpqa_diamond", "", "Opus 4.6", "anthropic", 91.3},
		{"gpqa_diamond", "", "GPT-5.2", "openai", 92.4},
		// Terminal-Bench
		{"terminal_bench", "Terminus-2", "Gemini 3.1 Pro", "google", 68.5},
		{"terminal_bench", "Terminus-2", "Gemini 3 Pro", "google", 56.9},
		{"terminal_bench", "Terminus-2", "Sonnet 4.6", "anthropic", 59.1},
		{"terminal_bench", "Terminus-2", "Opus 4.6", "anthropic", 65.4},
		{"terminal_bench", "Terminus-2", "GPT-5.2", "openai", 54.0},
		{"terminal_bench", "Terminus-2", "GPT-5.3-Codex", "openai", 64.7},
		{"terminal_bench", "Best self-reported", "GPT-5.2", "openai", 62.2},
		{"terminal_bench", "Best self-reported", "GPT-5.3-Codex", "openai", 77.3},
		// SWE-Bench Verified
		{"swe_bench_verified", "", "Gemini 3.1 Pro", "google", 80.6},
		{"swe_bench_verified", "", "Gemini 3 Pro", "google", 76.2},
		{"swe_bench_verified", "", "Sonnet 4.6", "anthropic", 79.6},
		{"swe_bench_verified", "", "Opus 4.6", "anthropic", 80.8},
		{"swe_bench_verified", "", "GPT-5.2", "openai", 80.0},
		// SWE-Bench Pro
		{"swe_bench_pro", "", "Gemini 3.1 Pro", "google", 54.2},
		{"swe_bench_pro", "", "Gemini 3 Pro", "google", 43.3},
		{"swe_bench_pro", "", "GPT-5.2", "openai", 55.6},
		{"swe_bench_pro", "", "GPT-5.3-Codex", "openai", 56.8},
		// LiveCodeBench
		{"livecodebench_pro", "", "Gemini 3.1 Pro", "google", 2887},
		{"livecodebench_pro", "", "Gemini 3 Pro", "google", 2439},
		{"livecodebench_pro", "", "GPT-5.2", "openai", 2393},
		// SciCode
		{"scicode", "", "Gemini 3.1 Pro", "google", 59},
		{"scicode", "", "Gemini 3 Pro", "google", 56},
		{"scicode", "", "Sonnet 4.6", "anthropic", 47},
		{"scicode", "", "Opus 4.6", "anthropic", 52},
		{"scicode", "", "GPT-5.2", "openai", 52},
		// APEX-Agents
		{"apex_agents", "", "Gemini 3.1 Pro", "google", 33.5},
		{"apex_agents", "", "Gemini 3 Pro", "google", 18.4},
		{"apex_agents", "", "Opus 4.6", "anthropic", 29.8},
		{"apex_agents", "", "GPT-5.2", "openai", 23.0},
		// GDPval-AA
		{"gdpval_aa", "", "Gemini 3.1 Pro", "google", 1317},
		{"gdpval_aa", "", "Gemini 3 Pro", "google", 1195},
		{"gdpval_aa", "", "Sonnet 4.6", "anthropic", 1633},
		{"gdpval_aa", "", "Opus 4.6", "anthropic", 1606},
		{"gdpval_aa", "", "GPT-5.2", "openai", 1462},
		// t2-bench
		{"t2_bench", "Retail", "Gemini 3.1 Pro", "google", 90.8},
		{"t2_bench", "Retail", "Gemini 3 Pro", "google", 85.3},
		{"t2_bench", "Retail", "Sonnet 4.6", "anthropic", 91.7},
		{"t2_bench", "Retail", "Opus 4.6", "anthropic", 91.9},
		{"t2_bench", "Retail", "GPT-5.2", "openai", 82.0},
		{"t2_bench", "Telecom", "Gemini 3.1 Pro", "google", 99.3},
		{"t2_bench", "Telecom", "Gemini 3 Pro", "google", 98.0},
		{"t2_bench", "Telecom", "Sonnet 4.6", "anthropic", 97.9},
		{"t2_bench", "Telecom", "Opus 4.6", "anthropic", 99.3},
		{"t2_bench", "Telecom", "GPT-5.2", "openai", 98.7},
		// MCP Atlas
		{"mcp_atlas", "", "Gemini 3.1 Pro", "google", 69.2},
		{"mcp_atlas", "", "Gemini 3 Pro", "google", 54.1},
		{"mcp_atlas", "", "Sonnet 4.6", "anthropic", 61.3},
		{"mcp_atlas", "", "Opus 4.6", "anthropic", 59.5},
		{"mcp_atlas", "", "GPT-5.2", "openai", 60.6},
		// BrowseComp
		{"browsecomp", "", "Gemini 3.1 Pro", "google", 85.9},
		{"browsecomp", "", "Gemini 3 Pro", "google", 59.2},
		{"browsecomp", "", "Sonnet 4.6", "anthropic", 74.7},
		{"browsecomp", "", "Opus 4.6", "anthropic", 84.0},
		{"browsecomp", "", "GPT-5.2", "openai", 65.8},
		// MMMU Pro
		{"mmmu_pro", "", "Gemini 3.1 Pro", "google", 80.5},
		{"mmmu_pro", "", "Gemini 3 Pro", "google", 81.0},
		{"mmmu_pro", "", "Sonnet 4.6", "anthropic", 74.5},
		{"mmmu_pro", "", "Opus 4.6", "anthropic", 73.9},
		{"mmmu_pro", "", "GPT-5.2", "openai", 79.5},
		// MMMLU
		{"mmmlu", "", "Gemini 3.1 Pro", "google", 92.6},
		{"mmmlu", "", "Gemini 3 Pro", "google", 91.8},
		{"mmmlu", "", "Sonnet 4.6", "anthropic", 89.3},
		{"mmmlu", "", "Opus 4.6", "anthropic", 91.1},
		{"mmmlu", "", "GPT-5.2", "openai", 89.6},
		// MRCR v2
		{"mrcr_v2", "128k (avg)", "Gemini 3.1 Pro", "google", 84.9},
		{"mrcr_v2", "128k (avg)", "Gemini 3 Pro", "google", 77.0},
		{"mrcr_v2", "128k (avg)", "Sonnet 4.6", "anthropic", 84.9},
		{"mrcr_v2", "128k (avg)", "Opus 4.6", "anthropic", 84.0},
		{"mrcr_v2", "128k (avg)", "GPT-5.2", "openai", 83.8},
		{"mrcr_v2", "1M (pointwise)", "Gemini 3.1 Pro", "google", 26.3},
		{"mrcr_v2", "1M (pointwise)", "Gemini 3 Pro", "google", 26.3},
	}

	scores := make([]BenchmarkScore, len(data))
	for i, d := range data {
		scores[i] = BenchmarkScore{
			BenchmarkID:   d.bench,
			ModelName:     d.model,
			ModelProvider: d.provider,
			Variant:       d.variant,
			Score:         d.score,
			SourceURL:     "https://deepmind.google/models/evals-methodology/gemini-3-1-pro",
		}
	}
	return scores
}
