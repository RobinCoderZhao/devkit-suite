package parsers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/benchmarks"
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

// cleanModelName removes common suffixes/prefixes from model names.
func cleanModelName(name string) string {
	// Remove parenthetical suffixes: "Model (thinking)", "Model (high)"
	re := regexp.MustCompile(`\s*\([^)]*\)\s*`)
	name = re.ReplaceAllString(name, "")

	// Remove common prefixes
	name = strings.TrimPrefix(name, "🥇 ")
	name = strings.TrimPrefix(name, "🥈 ")
	name = strings.TrimPrefix(name, "🥉 ")

	// Remove markdown formatting
	name = strings.ReplaceAll(name, "**", "")
	name = strings.ReplaceAll(name, "*", "")

	return strings.TrimSpace(name)
}

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
func ParseScore(cell string) (float64, bool) {
	cell = strings.TrimSpace(cell)
	cell = strings.TrimSuffix(cell, "%")
	cell = strings.ReplaceAll(cell, ",", "")

	matches := scoreRegex.FindStringSubmatch(cell)
	if len(matches) < 2 {
		return 0, false
	}

	var score float64
	_, err := fmt.Sscanf(matches[1], "%f", &score)
	return score, err == nil
}
