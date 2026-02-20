package benchmarks

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/fogleman/gg"
)

// ImageRenderer renders a BenchmarkReport as a tech-style PNG image.
type ImageRenderer struct {
	Width     float64
	RowHeight float64
	HeaderH   float64
	GroupH    float64
	FooterH   float64
	PadLeft   float64
	PadRight  float64
	FontSize  float64
	TitleSize float64
	SmallSize float64
}

// NewImageRenderer creates a renderer with 2400px width.
func NewImageRenderer() *ImageRenderer {
	return &ImageRenderer{
		Width:     2400,
		RowHeight: 56,
		HeaderH:   100,
		GroupH:    52,
		FooterH:   80,
		PadLeft:   40,
		PadRight:  40,
		FontSize:  22,
		TitleSize: 32,
		SmallSize: 18,
	}
}

// RenderPNG generates a PNG image of the benchmark report.
func (r *ImageRenderer) RenderPNG(report *BenchmarkReport, outputPath string) error {
	// Calculate dimensions
	totalRows := r.countRows(report)
	height := r.HeaderH + float64(len(Categories))*r.GroupH + float64(totalRows)*r.RowHeight +
		r.RowHeight + r.FooterH + 60 // +60 for model header row + padding

	dc := gg.NewContext(int(r.Width), int(height))

	// Background
	r.drawBackground(dc, height)

	// Title
	y := r.drawTitle(dc, report)

	// Model header row
	y = r.drawModelHeaders(dc, report, y)

	// Benchmark rows by category
	for _, cat := range Categories {
		benchmarks := r.benchmarksForCategory(cat.ID)
		if len(benchmarks) == 0 {
			continue
		}
		y = r.drawCategoryHeader(dc, cat, y)
		for _, bench := range benchmarks {
			if len(bench.Variants) > 0 {
				for _, v := range bench.Variants {
					y = r.drawScoreRow(dc, report, bench, v, y)
				}
			} else {
				y = r.drawScoreRow(dc, report, bench, "", y)
			}
		}
	}

	// Footer
	r.drawFooter(dc, y, report)

	return dc.SavePNG(outputPath)
}

func (r *ImageRenderer) countRows(report *BenchmarkReport) int {
	count := 0
	for _, b := range AllBenchmarks {
		if len(b.Variants) > 0 {
			count += len(b.Variants)
		} else {
			count++
		}
	}
	return count
}

func (r *ImageRenderer) benchmarksForCategory(catID string) []BenchmarkDef {
	var result []BenchmarkDef
	for _, b := range AllBenchmarks {
		if b.Category == catID {
			result = append(result, b)
		}
	}
	return result
}

// ---- Drawing helpers ----

func (r *ImageRenderer) drawBackground(dc *gg.Context, height float64) {
	// Deep space gradient
	for y := 0; y < int(height); y++ {
		t := float64(y) / height
		cr := 10 + t*5
		cg := 10 + t*5
		cb := 26 + t*10
		dc.SetColor(color.RGBA{uint8(cr), uint8(cg), uint8(cb), 255})
		dc.DrawRectangle(0, float64(y), r.Width, 1)
		dc.Fill()
	}
}

func (r *ImageRenderer) drawTitle(dc *gg.Context, report *BenchmarkReport) float64 {
	// Title bar background
	dc.SetColor(hexColor("#1a1a3e"))
	dc.DrawRoundedRectangle(r.PadLeft, 20, r.Width-r.PadLeft-r.PadRight, r.HeaderH, 12)
	dc.Fill()

	// Accent line
	dc.SetColor(hexColor("#4a9eff"))
	dc.DrawRectangle(r.PadLeft, 20, 4, r.HeaderH)
	dc.Fill()

	// Title text
	r.loadFont(dc, r.TitleSize, true)
	dc.SetColor(color.White)
	title := fmt.Sprintf("AI Model Benchmark Report · %s", report.Date)
	dc.DrawStringAnchored(title, r.Width/2, 20+r.HeaderH/2-8, 0.5, 0.5)

	// Subtitle
	r.loadFont(dc, r.SmallSize, false)
	dc.SetColor(hexColor("#8888aa"))
	subtitle := fmt.Sprintf("%d benchmarks · %d models · Data from Artificial Analysis + official sources",
		len(AllBenchmarks), len(report.Models))
	dc.DrawStringAnchored(subtitle, r.Width/2, 20+r.HeaderH/2+20, 0.5, 0.5)

	return 20 + r.HeaderH + 16
}

func (r *ImageRenderer) drawModelHeaders(dc *gg.Context, report *BenchmarkReport, y float64) float64 {
	colWidth := (r.Width - r.PadLeft - r.PadRight - 280) / float64(len(report.Models))

	// Header background
	dc.SetColor(hexColor("#141428"))
	dc.DrawRectangle(r.PadLeft, y, r.Width-r.PadLeft-r.PadRight, r.RowHeight)
	dc.Fill()

	// "Benchmark" label
	r.loadFont(dc, r.SmallSize, true)
	dc.SetColor(hexColor("#6666aa"))
	dc.DrawString("Benchmark", r.PadLeft+16, y+r.RowHeight/2+6)

	// Model names
	r.loadFont(dc, r.SmallSize-2, true)
	x := r.PadLeft + 280
	for _, m := range report.Models {
		// Provider color dot
		dc.SetColor(hexColor(ProviderColor(m.Provider)))
		dotX := x + colWidth/2 - 4
		dc.DrawCircle(dotX, y+16, 4)
		dc.Fill()

		// Model name (truncate if too long)
		name := m.Name
		if len(name) > 14 {
			name = name[:14]
		}
		dc.SetColor(hexColor("#aaaacc"))
		tw, _ := dc.MeasureString(name)
		dc.DrawString(name, x+colWidth/2-tw/2, y+r.RowHeight/2+6)

		// Thinking mode label
		if m.Thinking != "" {
			r.loadFont(dc, 12, false)
			dc.SetColor(hexColor("#555577"))
			tw2, _ := dc.MeasureString(m.Thinking)
			dc.DrawString(m.Thinking, x+colWidth/2-tw2/2, y+r.RowHeight/2+22)
			r.loadFont(dc, r.SmallSize-2, true)
		}

		x += colWidth
	}

	// Separator line
	dc.SetColor(hexColor("#1a1a3e"))
	dc.SetLineWidth(1)
	dc.DrawLine(r.PadLeft, y+r.RowHeight, r.Width-r.PadRight, y+r.RowHeight)
	dc.Stroke()

	return y + r.RowHeight
}

func (r *ImageRenderer) drawCategoryHeader(dc *gg.Context, cat CategoryMeta, y float64) float64 {
	// Category background
	dc.SetColor(hexColor("#0d0d1f"))
	dc.DrawRectangle(r.PadLeft, y, r.Width-r.PadLeft-r.PadRight, r.GroupH)
	dc.Fill()

	// Colored left bar
	dc.SetColor(hexColor(cat.Color))
	dc.DrawRectangle(r.PadLeft, y, 4, r.GroupH)
	dc.Fill()

	// Emoji + Label
	r.loadFont(dc, r.FontSize, true)
	dc.SetColor(hexColor(cat.Color))
	label := fmt.Sprintf("%s %s", cat.Emoji, cat.Label)
	dc.DrawString(label, r.PadLeft+16, y+r.GroupH/2+7)

	return y + r.GroupH
}

func (r *ImageRenderer) drawScoreRow(dc *gg.Context, report *BenchmarkReport, bench BenchmarkDef, variant string, y float64) float64 {
	colWidth := (r.Width - r.PadLeft - r.PadRight - 280) / float64(len(report.Models))
	isOdd := int(y/r.RowHeight)%2 == 1

	// Row background
	if isOdd {
		dc.SetColor(hexColor("#12122a"))
	} else {
		dc.SetColor(hexColor("#0f0f20"))
	}
	dc.DrawRectangle(r.PadLeft, y, r.Width-r.PadLeft-r.PadRight, r.RowHeight)
	dc.Fill()

	// Benchmark name
	r.loadFont(dc, r.SmallSize, false)
	dc.SetColor(hexColor("#c0c0d0"))
	name := bench.Name
	if variant != "" {
		name = fmt.Sprintf("  %s", variant)
	}
	dc.DrawString(name, r.PadLeft+20, y+r.RowHeight/2+6)

	// Unit hint
	if variant == "" || variant == bench.Variants[0] {
		r.loadFont(dc, 13, false)
		dc.SetColor(hexColor("#444460"))
		dc.DrawString(bench.Unit, r.PadLeft+240, y+r.RowHeight/2+6)
	}

	// Scores
	r.loadFont(dc, r.FontSize, false)
	x := r.PadLeft + 280
	for _, m := range report.Models {
		score, exists := report.GetScore(bench.ID, variant, m.Name)
		isTop := report.IsHighest(bench.ID, variant, m.Name)

		cellCenter := x + colWidth/2

		if !exists {
			// Missing data
			dc.SetColor(hexColor("#404050"))
			tw, _ := dc.MeasureString("—")
			dc.DrawString("—", cellCenter-tw/2, y+r.RowHeight/2+7)
		} else {
			// Format score
			scoreStr := formatScore(score, bench.Unit)

			if isTop {
				// Highlight: red glow background + red text
				tw, _ := dc.MeasureString(scoreStr)
				padX := 8.0
				padY := 4.0
				bgX := cellCenter - tw/2 - padX
				bgY := y + r.RowHeight/2 - 12 - padY
				bgW := tw + 2*padX
				bgH := 24.0 + 2*padY

				dc.SetColor(color.RGBA{255, 45, 85, 40}) // #ff2d5528
				dc.DrawRoundedRectangle(bgX, bgY, bgW, bgH, 4)
				dc.Fill()

				r.loadFont(dc, r.FontSize, true)
				dc.SetColor(hexColor("#ff4757"))
				dc.DrawString(scoreStr, cellCenter-tw/2, y+r.RowHeight/2+7)
				r.loadFont(dc, r.FontSize, false)
			} else {
				dc.SetColor(hexColor("#e0e0e0"))
				tw, _ := dc.MeasureString(scoreStr)
				dc.DrawString(scoreStr, cellCenter-tw/2, y+r.RowHeight/2+7)
			}
		}

		x += colWidth
	}

	// Bottom separator
	dc.SetColor(hexColor("#1a1a3e30"))
	dc.SetLineWidth(0.5)
	dc.DrawLine(r.PadLeft+280, y+r.RowHeight, r.Width-r.PadRight, y+r.RowHeight)
	dc.Stroke()

	return y + r.RowHeight
}

func (r *ImageRenderer) drawFooter(dc *gg.Context, y float64, report *BenchmarkReport) {
	y += 16

	// Footer bar
	dc.SetColor(hexColor("#0a0a16"))
	dc.DrawRoundedRectangle(r.PadLeft, y, r.Width-r.PadLeft-r.PadRight, r.FooterH, 8)
	dc.Fill()

	r.loadFont(dc, 16, false)
	dc.SetColor(hexColor("#444460"))
	footer := fmt.Sprintf("WatchBot Benchmark Tracker · Data scraped %s · Red = highest score per benchmark",
		report.Date)
	dc.DrawStringAnchored(footer, r.Width/2, y+r.FooterH/2+4, 0.5, 0.5)
}

// ---- Helpers ----

func (r *ImageRenderer) loadFont(dc *gg.Context, size float64, bold bool) {
	// gg uses system fonts; we use the default if custom font is not found
	// In production, embed Noto Sans CJK for Chinese support
	if err := dc.LoadFontFace("/System/Library/Fonts/Helvetica.ttc", size); err != nil {
		// Fallback — gg will use a basic built-in font
		_ = dc.LoadFontFace("/System/Library/Fonts/SFNSMono.ttf", size)
	}
}

func formatScore(score float64, unit string) string {
	if unit == "Elo" {
		return fmt.Sprintf("%d", int(math.Round(score)))
	}
	// Percentage
	if score == math.Trunc(score) {
		return fmt.Sprintf("%.0f%%", score)
	}
	return fmt.Sprintf("%.1f%%", score)
}

func hexColor(hex string) color.Color {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 8 {
		// RGBA
		var r, g, b, a uint8
		fmt.Sscanf(hex, "%02x%02x%02x%02x", &r, &g, &b, &a)
		return color.RGBA{r, g, b, a}
	}
	var cr, cg, cb uint8
	fmt.Sscanf(hex, "%02x%02x%02x", &cr, &cg, &cb)
	return color.RGBA{cr, cg, cb, 255}
}
