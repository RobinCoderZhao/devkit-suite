package parsers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/RobinCoderZhao/devkit-suite/pkg/benchmarks"
)

// ExtractMarkdownTable parses a Markdown table from Jina Reader output.
// Returns rows as [][]string (header row first, then data rows).
func ExtractMarkdownTable(markdown string) [][]string {
	lines := strings.Split(markdown, "\n")
	var rows [][]string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "|") {
			continue
		}
		// Skip separator rows like |---|---|---|
		if isSeparatorRow(line) {
			continue
		}
		cells := splitTableRow(line)
		if len(cells) >= 2 {
			rows = append(rows, cells)
		}
	}
	return rows
}

// isSeparatorRow detects markdown table separator rows.
func isSeparatorRow(line string) bool {
	cleaned := strings.ReplaceAll(line, "|", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	cleaned = strings.TrimSpace(cleaned)
	return cleaned == ""
}

// splitTableRow splits a markdown table row by pipe characters.
func splitTableRow(line string) []string {
	// Remove leading/trailing pipes
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")

	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// MatchModelName fuzzy-matches a raw model name from a leaderboard to known models.
// Returns the matched ModelConfig and true if found.
func MatchModelName(rawName string, knownModels []benchmarks.ModelConfig) (*benchmarks.ModelConfig, bool) {
	rawLower := strings.ToLower(cleanModelName(rawName))

	// Check alias table first
	if aliasedName, ok := modelAliases[rawLower]; ok {
		rawLower = strings.ToLower(aliasedName)
	}

	for i := range knownModels {
		modelLower := strings.ToLower(knownModels[i].Name)

		// Exact match
		if rawLower == modelLower {
			return &knownModels[i], true
		}

		// Contains match (e.g., "Gemini 3.1 Pro (thinking)" → "Gemini 3.1 Pro")
		if strings.Contains(rawLower, modelLower) {
			return &knownModels[i], true
		}

		// Reverse contains (e.g., "GPT-5.2" found in "OpenAI GPT-5.2")
		if strings.Contains(rawLower, strings.ToLower(knownModels[i].Name)) {
			return &knownModels[i], true
		}
	}

	// Try normalized matching
	rawNorm := normalizeModelName(rawLower)
	for i := range knownModels {
		modelNorm := normalizeModelName(strings.ToLower(knownModels[i].Name))
		if rawNorm == modelNorm || strings.Contains(rawNorm, modelNorm) {
			return &knownModels[i], true
		}
	}

	return nil, false
}

// modelAliases maps real-world model names (from LLM leaderboards)
// to the tracked model names in our DefaultModels/FallbackModels.
var modelAliases = map[string]string{
	// Google models
	"gemini 2.5 pro":   "Gemini 2.5 Pro",
	"gemini 2.5 flash": "Gemini 2.0 Flash",
	"gemini-2.5-pro":   "Gemini 2.5 Pro",
	"gemini-2.5-flash": "Gemini 2.0 Flash",
	// OpenAI models
	"o3":                "o3-mini",
	"o3-mini":           "o3-mini",
	"gpt-5 mini":        "GPT-4o",
	"gpt-4o":            "GPT-4o",
	"chatgpt-4o latest": "GPT-4o",
	"o1":                "o1",
	// Anthropic models
	"claude sonnet 4.5": "Claude 3.5 Sonnet",
	"claude 3.7 sonnet": "Claude 3.5 Sonnet",
	"claude opus 4.6":   "Claude 3 Opus",
	"claude 3 opus":     "Claude 3 Opus",
	"claude 3.5 sonnet": "Claude 3.5 Sonnet",
	// DeepSeek
	"deepseek-v3.2": "DeepSeek-V3",
	"deepseek-v3":   "DeepSeek-V3",
	"deepseek-r2":   "DeepSeek-R2",
	// Alibaba
	"qwen3-235b":  "Qwen3-235B",
	"qwen-3-235b": "Qwen3-235B",
	"qwen2.5-max": "Qwen2.5-Max",
	// MiniMax
	"minimax-m2.5": "MiniMax-M2.5",
	"minimax-m1":   "MiniMax-M1",
}

// cleanModelName removes common suffixes/prefixes from model names.
func cleanModelName(name string) string {
	// IMPORTANT: Remove image markdown FIRST: ![alt](url)
	// Otherwise linkRegex matches [alt text] inside ![]() giving wrong result
	name = imgRegex.ReplaceAllString(name, "")

	// Extract name from markdown link: [Model Name](url) Provider
	if linkMatch := linkRegex.FindStringSubmatch(name); len(linkMatch) >= 2 {
		name = linkMatch[1]
	}

	// Remove parenthetical suffixes: "Model (thinking)", "Model (high)"
	parenRegex := regexp.MustCompile(`\s*\([^)]*\)\s*`)
	name = parenRegex.ReplaceAllString(name, "")

	// Remove common prefixes
	name = strings.TrimPrefix(name, "🥇 ")
	name = strings.TrimPrefix(name, "🥈 ")
	name = strings.TrimPrefix(name, "🥉 ")

	// Remove markdown formatting
	name = strings.ReplaceAll(name, "**", "")
	name = strings.ReplaceAll(name, "*", "")

	return strings.TrimSpace(name)
}

var (
	// Matches [text](url) — captures text
	linkRegex = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	// Matches ![alt](url)
	imgRegex = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
)

// normalizeModelName creates a simplified form for matching.
func normalizeModelName(name string) string {
	// Remove version punctuation variations
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, ".", "")
	return name
}

var scoreRegex = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)\s*%?$`)

// ParseScore extracts a numeric score from a cell value.
// Handles both percentage (85.0) and decimal (0.850) formats.
func ParseScore(cell string) (float64, bool) {
	cell = strings.TrimSpace(cell)

	// Skip non-score cells
	if cell == "" || cell == "—" || cell == "-" || cell == "N/A" {
		return 0, false
	}

	// Remove trailing asterisks, daggers, etc.
	cell = strings.TrimRight(cell, "*†‡")

	hasPercentSign := strings.HasSuffix(cell, "%")
	cell = strings.TrimSuffix(cell, "%")
	cell = strings.ReplaceAll(cell, ",", "")

	matches := scoreRegex.FindStringSubmatch(cell)
	if len(matches) < 2 {
		return 0, false
	}

	var score float64
	_, err := fmt.Sscanf(matches[1], "%f", &score)
	if err != nil {
		return 0, false
	}

	// Auto-detect decimal format: if score is 0.0-1.0 and no % sign,
	// it's likely a decimal — convert to percentage.
	// Exception: values like 0 or 1 exactly (could be count)
	if !hasPercentSign && score > 0 && score < 1.0 {
		score = score * 100
	}

	return score, true
}
