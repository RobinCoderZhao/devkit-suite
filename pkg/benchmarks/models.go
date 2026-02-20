// Package benchmarks provides shared benchmark tracking for AI model evaluation.
//
// It defines 16 standardized benchmarks across 7 capability categories,
// supports structured data storage, multi-source scraping, and dual rendering
// (HTML table + PNG image).
package benchmarks

import "time"

// ---- Capability Categories ----

const (
	CatReasoning   = "reasoning"
	CatCoding      = "coding"
	CatAgent       = "agent"
	CatSearch      = "search"
	CatMultimodal  = "multimodal"
	CatKnowledge   = "knowledge"
	CatLongContext = "long_context"
)

// CategoryMeta holds display info for each category.
type CategoryMeta struct {
	ID    string
	Label string
	Emoji string
	Color string // hex color for rendering
}

// Categories defines the display order and styling of capability groups.
var Categories = []CategoryMeta{
	{CatReasoning, "Reasoning", "🧠", "#4a9eff"},
	{CatCoding, "Coding", "💻", "#4caf50"},
	{CatAgent, "Agent", "🤖", "#ff9800"},
	{CatSearch, "Search", "🔍", "#ab47bc"},
	{CatMultimodal, "Multimodal", "🖼", "#ef5350"},
	{CatKnowledge, "Knowledge", "📚", "#26c6da"},
	{CatLongContext, "Long Context", "🧾", "#78909c"},
}

// ---- Benchmark Definitions ----

// BenchmarkDef defines a benchmark standard test.
type BenchmarkDef struct {
	ID       string   // unique key, e.g. "swe_bench_verified"
	Name     string   // display name, e.g. "SWE-Bench Verified"
	Category string   // one of Cat* constants
	Unit     string   // "%" or "Elo"
	Variants []string // sub-tests, e.g. ["No tools", "Search+Code"]
}

// AllBenchmarks defines all 16 tracked benchmarks in display order.
var AllBenchmarks = []BenchmarkDef{
	// 🧠 Reasoning
	{
		ID: "hle", Name: "Humanity's Last Exam",
		Category: CatReasoning, Unit: "%",
		Variants: []string{"No tools", "Search+Code"},
	},
	{
		ID: "arc_agi_2", Name: "ARC-AGI-2",
		Category: CatReasoning, Unit: "%",
	},
	{
		ID: "gpqa_diamond", Name: "GPQA Diamond",
		Category: CatReasoning, Unit: "%",
	},

	// 💻 Coding
	{
		ID: "terminal_bench", Name: "Terminal-Bench 2.0",
		Category: CatCoding, Unit: "%",
		Variants: []string{"Terminus-2", "Best self-reported"},
	},
	{
		ID: "swe_bench_verified", Name: "SWE-Bench Verified",
		Category: CatCoding, Unit: "%",
	},
	{
		ID: "swe_bench_pro", Name: "SWE-Bench Pro",
		Category: CatCoding, Unit: "%",
	},
	{
		ID: "livecodebench_pro", Name: "LiveCodeBench Pro",
		Category: CatCoding, Unit: "Elo",
	},
	{
		ID: "scicode", Name: "SciCode",
		Category: CatCoding, Unit: "%",
	},

	// 🤖 Agent
	{
		ID: "apex_agents", Name: "APEX-Agents",
		Category: CatAgent, Unit: "%",
	},
	{
		ID: "gdpval_aa", Name: "GDPval-AA",
		Category: CatAgent, Unit: "Elo",
	},
	{
		ID: "t2_bench", Name: "t2-bench",
		Category: CatAgent, Unit: "%",
		Variants: []string{"Retail", "Telecom"},
	},
	{
		ID: "mcp_atlas", Name: "MCP Atlas",
		Category: CatAgent, Unit: "%",
	},

	// 🔍 Search
	{
		ID: "browsecomp", Name: "BrowseComp",
		Category: CatSearch, Unit: "%",
	},

	// 🖼 Multimodal
	{
		ID: "mmmu_pro", Name: "MMMU Pro",
		Category: CatMultimodal, Unit: "%",
	},

	// 📚 Knowledge
	{
		ID: "mmmlu", Name: "MMMLU",
		Category: CatKnowledge, Unit: "%",
	},

	// 🧾 Long Context
	{
		ID: "mrcr_v2", Name: "MRCR v2 (8-needle)",
		Category: CatLongContext, Unit: "%",
		Variants: []string{"128k (avg)", "1M (pointwise)"},
	},
}

// BenchmarksByCategory returns benchmarks grouped by category in display order.
func BenchmarksByCategory() map[string][]BenchmarkDef {
	m := make(map[string][]BenchmarkDef)
	for _, b := range AllBenchmarks {
		m[b.Category] = append(m[b.Category], b)
	}
	return m
}

// FindBenchmark returns a benchmark definition by ID.
func FindBenchmark(id string) *BenchmarkDef {
	for i := range AllBenchmarks {
		if AllBenchmarks[i].ID == id {
			return &AllBenchmarks[i]
		}
	}
	return nil
}

// ---- Score Data ----

// BenchmarkScore holds one benchmark result for a specific model.
type BenchmarkScore struct {
	ID            int       `json:"id"`
	BenchmarkID   string    `json:"benchmark_id"`
	ModelName     string    `json:"model_name"`
	ModelProvider string    `json:"model_provider"`
	Variant       string    `json:"variant"`
	Score         float64   `json:"score"`
	SourceURL     string    `json:"source_url"`
	ScrapedAt     time.Time `json:"scraped_at"`
}

// ---- Model Configuration ----

// ModelConfig defines a model to track in the benchmark comparison.
type ModelConfig struct {
	Name         string `json:"name" yaml:"name"`                   // "Gemini 3.1 Pro"
	Provider     string `json:"provider" yaml:"provider"`           // "google"
	Thinking     string `json:"thinking,omitempty" yaml:"thinking"` // "High" / "Max"
	Gen          string `json:"gen,omitempty" yaml:"gen"`           // "latest" / "previous"
	DisplayOrder int    `json:"display_order" yaml:"display_order"`
}

// DefaultModels returns the default 12 model columns.
var DefaultModels = []ModelConfig{
	{Name: "Gemini 3.1 Pro", Provider: "google", Thinking: "High", Gen: "latest", DisplayOrder: 1},
	{Name: "Gemini 3 Pro", Provider: "google", Thinking: "High", Gen: "previous", DisplayOrder: 2},
	{Name: "Opus 4.6", Provider: "anthropic", Thinking: "Max", Gen: "latest", DisplayOrder: 3},
	{Name: "Sonnet 4.6", Provider: "anthropic", Thinking: "Max", Gen: "previous", DisplayOrder: 4},
	{Name: "GPT-5.2", Provider: "openai", Thinking: "xhigh", Gen: "latest", DisplayOrder: 5},
	{Name: "GPT-5.3-Codex", Provider: "openai", Thinking: "xhigh", Gen: "previous", DisplayOrder: 6},
	{Name: "Qwen3-235B", Provider: "alibaba", Gen: "latest", DisplayOrder: 7},
	{Name: "Qwen2.5-Max", Provider: "alibaba", Gen: "previous", DisplayOrder: 8},
	{Name: "DeepSeek-R2", Provider: "deepseek", Gen: "latest", DisplayOrder: 9},
	{Name: "DeepSeek-V3", Provider: "deepseek", Gen: "previous", DisplayOrder: 10},
	{Name: "MiniMax-M2.5", Provider: "minimax", Gen: "latest", DisplayOrder: 11},
	{Name: "MiniMax-M1", Provider: "minimax", Gen: "previous", DisplayOrder: 12},
}

// ProviderColor returns the brand color for rendering.
func ProviderColor(provider string) string {
	switch provider {
	case "google":
		return "#4285f4"
	case "anthropic":
		return "#d4a574"
	case "openai":
		return "#10a37f"
	case "alibaba":
		return "#ff6a00"
	case "deepseek":
		return "#4a6cf7"
	case "minimax":
		return "#7c3aed"
	default:
		return "#808080"
	}
}

// ---- Report Data ----

// BenchmarkReport holds all data needed to render a comparison table/image.
type BenchmarkReport struct {
	Models     []ModelConfig
	Benchmarks []BenchmarkDef
	Scores     map[string]map[string]float64 // [benchmarkID+variant][modelName] → score
	HighestOf  map[string]string             // [benchmarkID+variant] → modelName (highest scorer)
	Date       string
}

// ScoreKey builds a lookup key for the Scores map.
func ScoreKey(benchmarkID, variant string) string {
	if variant == "" {
		return benchmarkID
	}
	return benchmarkID + ":" + variant
}

// NewReport creates an empty report with the given models.
func NewReport(models []ModelConfig, date string) *BenchmarkReport {
	return &BenchmarkReport{
		Models:     models,
		Benchmarks: AllBenchmarks,
		Scores:     make(map[string]map[string]float64),
		HighestOf:  make(map[string]string),
		Date:       date,
	}
}

// SetScore records a score and updates the highest tracker.
func (r *BenchmarkReport) SetScore(benchmarkID, variant, modelName string, score float64) {
	key := ScoreKey(benchmarkID, variant)
	if r.Scores[key] == nil {
		r.Scores[key] = make(map[string]float64)
	}
	r.Scores[key][modelName] = score

	// Track highest
	if current, exists := r.HighestOf[key]; !exists || score > r.Scores[key][current] {
		r.HighestOf[key] = modelName
	}
}

// GetScore returns the score for a model on a benchmark, and whether it exists.
func (r *BenchmarkReport) GetScore(benchmarkID, variant, modelName string) (float64, bool) {
	key := ScoreKey(benchmarkID, variant)
	if row, ok := r.Scores[key]; ok {
		score, exists := row[modelName]
		return score, exists
	}
	return 0, false
}

// IsHighest checks if a model has the highest score for a benchmark.
func (r *BenchmarkReport) IsHighest(benchmarkID, variant, modelName string) bool {
	key := ScoreKey(benchmarkID, variant)
	return r.HighestOf[key] == modelName
}

// ModelScoreCount returns the number of benchmark scores a model has.
func (r *BenchmarkReport) ModelScoreCount(modelName string) int {
	count := 0
	for _, scores := range r.Scores {
		if _, ok := scores[modelName]; ok {
			count++
		}
	}
	return count
}

// FilterEmptyModels removes models with fewer than minScores data points.
// Ensures the final model count is >= minModels by adding fallback models.
func (r *BenchmarkReport) FilterEmptyModels(minScores, minModels int) {
	if minScores <= 0 {
		minScores = 1
	}
	if minModels <= 0 {
		minModels = 10
	}

	// Filter models that have enough scores
	var kept []ModelConfig
	for _, m := range r.Models {
		if r.ModelScoreCount(m.Name) >= minScores {
			kept = append(kept, m)
		}
	}

	// If we don't have enough models, add fallbacks
	if len(kept) < minModels {
		existing := make(map[string]bool)
		for _, m := range kept {
			existing[m.Name] = true
		}
		for _, fb := range FallbackModels {
			if len(kept) >= minModels {
				break
			}
			if !existing[fb.Name] && r.ModelScoreCount(fb.Name) >= minScores {
				kept = append(kept, fb)
				existing[fb.Name] = true
			}
		}
	}

	// Re-assign display order
	for i := range kept {
		kept[i].DisplayOrder = i + 1
	}

	r.Models = kept
}

// FallbackModels are older-generation models from the big 3 providers,
// used to fill the comparison table when newer models lack data.
var FallbackModels = []ModelConfig{
	// Google older gens
	{Name: "Gemini 2.5 Pro", Provider: "google", Thinking: "High", Gen: "older"},
	{Name: "Gemini 2.0 Flash", Provider: "google", Gen: "older"},
	// Anthropic older gens
	{Name: "Claude 3.5 Sonnet", Provider: "anthropic", Gen: "older"},
	{Name: "Claude 3 Opus", Provider: "anthropic", Gen: "older"},
	// OpenAI older gens
	{Name: "GPT-4o", Provider: "openai", Gen: "older"},
	{Name: "o1", Provider: "openai", Thinking: "High", Gen: "older"},
	{Name: "o3-mini", Provider: "openai", Thinking: "High", Gen: "older"},
}
