package benchmarks

import (
	"fmt"
	"html"
	"math"
	"strings"
)

// HTMLRenderer produces an HTML table for email or web display.
type HTMLRenderer struct{}

func NewHTMLRenderer() *HTMLRenderer { return &HTMLRenderer{} }

// RenderHTML generates a complete HTML table of the benchmark report.
func (r *HTMLRenderer) RenderHTML(report *BenchmarkReport) string {
	var sb strings.Builder

	// Table open
	sb.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border-collapse:collapse;background-color:#0f0f23;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;font-size:13px;">`)

	// Header row
	sb.WriteString(`<tr style="background:#141428;">`)
	sb.WriteString(`<td style="padding:10px 12px;color:#6666aa;font-weight:700;border-bottom:2px solid #1a1a3e;">Benchmark</td>`)
	for _, m := range report.Models {
		provColor := ProviderColor(m.Provider)
		sb.WriteString(fmt.Sprintf(`<td style="padding:10px 8px;text-align:center;color:#aaaacc;font-weight:600;border-bottom:2px solid #1a1a3e;"><span style="display:inline-block;width:8px;height:8px;border-radius:50%%;background:%s;margin-right:4px;"></span>%s</td>`,
			provColor, html.EscapeString(m.Name)))
	}
	sb.WriteString(`</tr>`)

	// Rows by category
	for _, cat := range Categories {
		benchmarks := benchmarksForCategory(cat.ID)
		if len(benchmarks) == 0 {
			continue
		}

		// Category header
		sb.WriteString(fmt.Sprintf(`<tr><td colspan="%d" style="padding:10px 12px;background:#0d0d1f;color:%s;font-weight:700;font-size:14px;border-left:4px solid %s;">%s %s</td></tr>`,
			len(report.Models)+1, cat.Color, cat.Color, cat.Emoji, cat.Label))

		for _, bench := range benchmarks {
			if len(bench.Variants) > 0 {
				for _, v := range bench.Variants {
					r.writeScoreRow(&sb, report, bench, v)
				}
			} else {
				r.writeScoreRow(&sb, report, bench, "")
			}
		}
	}

	sb.WriteString(`</table>`)
	return sb.String()
}

func (r *HTMLRenderer) writeScoreRow(sb *strings.Builder, report *BenchmarkReport, bench BenchmarkDef, variant string) {
	sb.WriteString(`<tr>`)

	// Benchmark name
	name := bench.Name
	if variant != "" {
		name = "  " + variant
	}
	sb.WriteString(fmt.Sprintf(`<td style="padding:8px 12px;color:#c0c0d0;border-bottom:1px solid rgba(255,255,255,0.04);">%s <span style="color:#444460;font-size:11px;">%s</span></td>`,
		html.EscapeString(name), bench.Unit))

	// Scores
	for _, m := range report.Models {
		score, exists := report.GetScore(bench.ID, variant, m.Name)
		isTop := report.IsHighest(bench.ID, variant, m.Name)

		if !exists {
			sb.WriteString(`<td style="padding:8px;text-align:center;color:#404050;border-bottom:1px solid rgba(255,255,255,0.04);">—</td>`)
		} else {
			scoreStr := htmlFormatScore(score, bench.Unit)
			if isTop {
				sb.WriteString(fmt.Sprintf(`<td style="padding:8px;text-align:center;border-bottom:1px solid rgba(255,255,255,0.04);"><span style="background:rgba(255,45,85,0.15);color:#ff4757;font-weight:700;padding:2px 8px;border-radius:4px;">%s</span></td>`, scoreStr))
			} else {
				sb.WriteString(fmt.Sprintf(`<td style="padding:8px;text-align:center;color:#e0e0e0;border-bottom:1px solid rgba(255,255,255,0.04);">%s</td>`, scoreStr))
			}
		}
	}
	sb.WriteString(`</tr>`)
}

func benchmarksForCategory(catID string) []BenchmarkDef {
	var result []BenchmarkDef
	for _, b := range AllBenchmarks {
		if b.Category == catID {
			result = append(result, b)
		}
	}
	return result
}

func htmlFormatScore(score float64, unit string) string {
	if unit == "Elo" {
		return fmt.Sprintf("%d", int(math.Round(score)))
	}
	if score == math.Trunc(score) {
		return fmt.Sprintf("%.0f%%", score)
	}
	return fmt.Sprintf("%.1f%%", score)
}
