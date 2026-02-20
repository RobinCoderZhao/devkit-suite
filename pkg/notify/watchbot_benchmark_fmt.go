// Package notify — watchbot_benchmark_fmt.go provides Benchmark report formatters.
// Used by WatchBot to send benchmark comparison updates via email/Telegram.
package notify

import (
	"fmt"
	"strings"

	"github.com/RobinCoderZhao/devkit-suite/pkg/benchmarks"
)

// BenchmarkDigestData holds benchmark report data for notification formatting.
type BenchmarkDigestData struct {
	Report     *benchmarks.BenchmarkReport
	PNGPath    string // path to rendered PNG file
	HTMLTable  string // pre-rendered HTML table
	ScoreCount int    // total number of scores
	NewScores  int    // newly scraped scores
	Date       string
}

// BenchmarkEmailFormatter produces HTML email with benchmark comparison.
type BenchmarkEmailFormatter struct{}

func NewBenchmarkEmailFormatter() *BenchmarkEmailFormatter {
	return &BenchmarkEmailFormatter{}
}

func (f *BenchmarkEmailFormatter) Format(data BenchmarkDigestData) Message {
	var sb strings.Builder

	sb.WriteString(EmailWrapperOpen())
	sb.WriteString(EmailHeader(
		"📊 AI Benchmark Report",
		fmt.Sprintf("%d benchmarks · %d models · %s", len(benchmarks.AllBenchmarks), len(data.Report.Models), data.Date),
		"#4a9eff", "#6c5ce7",
	))

	// HTML table body
	if data.HTMLTable != "" {
		sb.WriteString(fmt.Sprintf(`
<tr><td style="padding:16px 24px;background-color:#0f0f23;">
  %s
</td></tr>`, data.HTMLTable))
	}

	// Summary
	sb.WriteString(fmt.Sprintf(`
<tr><td style="padding:16px 40px;background-color:#1a1a2e;">
  <p style="margin:0;font-size:13px;color:#808090;">
    📊 Data: %d scores · %d new this run
    <br>🔗 Source: <a href="https://llm-stats.com" style="color:#4a9eff;">llm-stats.com</a> + vendor evals
  </p>
</td></tr>`, data.ScoreCount, data.NewScores))

	sb.WriteString(EmailFooter("WatchBot Benchmark Tracker", "AI Model Comparison System", "#4a9eff"))
	sb.WriteString(EmailWrapperClose())

	return Message{
		Title:    fmt.Sprintf("📊 AI Benchmark Report — %s", data.Date),
		Body:     f.formatPlainText(data),
		HTMLBody: sb.String(),
		Format:   "html",
	}
}

func (f *BenchmarkEmailFormatter) formatPlainText(data BenchmarkDigestData) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 AI Benchmark Report — %s\n\n", data.Date))
	sb.WriteString(fmt.Sprintf("%d benchmarks, %d models\n", len(benchmarks.AllBenchmarks), len(data.Report.Models)))
	sb.WriteString(fmt.Sprintf("Scores: %d total, %d new\n\n", data.ScoreCount, data.NewScores))

	// Top performers by category
	for _, cat := range benchmarks.Categories {
		sb.WriteString(fmt.Sprintf("%s %s\n", cat.Emoji, cat.Label))
		for _, bench := range benchmarks.AllBenchmarks {
			if bench.Category != cat.ID {
				continue
			}
			variants := bench.Variants
			if len(variants) == 0 {
				variants = []string{""}
			}
			for _, v := range variants {
				key := benchmarks.ScoreKey(bench.ID, v)
				leader, ok := data.Report.HighestOf[key]
				if !ok {
					continue
				}
				score, _ := data.Report.GetScore(bench.ID, v, leader)
				label := bench.Name
				if v != "" {
					label = fmt.Sprintf("  %s", v)
				}
				sb.WriteString(fmt.Sprintf("  %s: 🔴 %s (%.1f%s)\n", label, leader, score, bench.Unit))
			}
		}
	}
	return sb.String()
}

// BenchmarkTelegramFormatter produces Telegram messages with benchmark summary.
type BenchmarkTelegramFormatter struct{}

func NewBenchmarkTelegramFormatter() *BenchmarkTelegramFormatter {
	return &BenchmarkTelegramFormatter{}
}

func (f *BenchmarkTelegramFormatter) Format(data BenchmarkDigestData) Message {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 *AI Benchmark Report* — %s\n", data.Date))
	sb.WriteString(fmt.Sprintf("%d benchmarks · %d models\n\n", len(benchmarks.AllBenchmarks), len(data.Report.Models)))

	// Highlight top 3 leaders across all benchmarks
	leaders := make(map[string]int)
	for _, modelName := range data.Report.HighestOf {
		leaders[modelName]++
	}

	sb.WriteString("🏆 *Top Leaders:*\n")
	type ldr struct {
		name  string
		count int
	}
	var sorted []ldr
	for name, count := range leaders {
		sorted = append(sorted, ldr{name, count})
	}
	// Simple sort by count desc
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].count > sorted[i].count {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	for i, l := range sorted {
		if i >= 5 {
			break
		}
		sb.WriteString(fmt.Sprintf("  🔴 %s — %d benchmarks\n", l.name, l.count))
	}

	sb.WriteString(fmt.Sprintf("\n_%d scores · %d new_", data.ScoreCount, data.NewScores))

	return Message{
		Title:  fmt.Sprintf("📊 Benchmark Report — %s", data.Date),
		Body:   sb.String(),
		Format: "markdown",
	}
}
