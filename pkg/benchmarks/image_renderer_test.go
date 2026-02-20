package benchmarks

import (
	"os"
	"testing"
)

func TestRenderImage(t *testing.T) {
	report := NewReport(DefaultModels, "2026-02-20")

	// Sample data from the user's screenshot
	// 🧠 Reasoning
	report.SetScore("hle", "No tools", "Gemini 3.1 Pro", 44.4)
	report.SetScore("hle", "No tools", "Gemini 3 Pro", 37.5)
	report.SetScore("hle", "No tools", "Opus 4.6", 40.0)
	report.SetScore("hle", "No tools", "Sonnet 4.6", 33.2)
	report.SetScore("hle", "No tools", "GPT-5.2", 34.5)
	report.SetScore("hle", "Search+Code", "Gemini 3.1 Pro", 51.4)
	report.SetScore("hle", "Search+Code", "Gemini 3 Pro", 45.8)
	report.SetScore("hle", "Search+Code", "Opus 4.6", 53.1)
	report.SetScore("hle", "Search+Code", "Sonnet 4.6", 49.0)
	report.SetScore("hle", "Search+Code", "GPT-5.2", 45.5)

	report.SetScore("arc_agi_2", "", "Gemini 3.1 Pro", 77.1)
	report.SetScore("arc_agi_2", "", "Gemini 3 Pro", 31.1)
	report.SetScore("arc_agi_2", "", "Opus 4.6", 68.8)
	report.SetScore("arc_agi_2", "", "Sonnet 4.6", 58.3)
	report.SetScore("arc_agi_2", "", "GPT-5.2", 52.9)

	report.SetScore("gpqa_diamond", "", "Gemini 3.1 Pro", 94.3)
	report.SetScore("gpqa_diamond", "", "Gemini 3 Pro", 91.9)
	report.SetScore("gpqa_diamond", "", "Opus 4.6", 91.3)
	report.SetScore("gpqa_diamond", "", "Sonnet 4.6", 89.9)
	report.SetScore("gpqa_diamond", "", "GPT-5.2", 92.4)

	// 💻 Coding
	report.SetScore("terminal_bench", "Terminus-2", "Gemini 3.1 Pro", 68.5)
	report.SetScore("terminal_bench", "Terminus-2", "Gemini 3 Pro", 56.9)
	report.SetScore("terminal_bench", "Terminus-2", "Opus 4.6", 65.4)
	report.SetScore("terminal_bench", "Terminus-2", "Sonnet 4.6", 59.1)
	report.SetScore("terminal_bench", "Terminus-2", "GPT-5.2", 54.0)
	report.SetScore("terminal_bench", "Terminus-2", "GPT-5.3-Codex", 64.7)
	report.SetScore("terminal_bench", "Best self-reported", "GPT-5.2", 62.2)
	report.SetScore("terminal_bench", "Best self-reported", "GPT-5.3-Codex", 77.3)

	report.SetScore("swe_bench_verified", "", "Gemini 3.1 Pro", 80.6)
	report.SetScore("swe_bench_verified", "", "Gemini 3 Pro", 76.2)
	report.SetScore("swe_bench_verified", "", "Opus 4.6", 80.8)
	report.SetScore("swe_bench_verified", "", "Sonnet 4.6", 79.6)
	report.SetScore("swe_bench_verified", "", "GPT-5.2", 80.0)

	report.SetScore("swe_bench_pro", "", "Gemini 3.1 Pro", 54.2)
	report.SetScore("swe_bench_pro", "", "Gemini 3 Pro", 43.3)
	report.SetScore("swe_bench_pro", "", "GPT-5.3-Codex", 56.8)
	report.SetScore("swe_bench_pro", "", "GPT-5.2", 55.6)

	report.SetScore("livecodebench_pro", "", "Gemini 3.1 Pro", 2887)
	report.SetScore("livecodebench_pro", "", "Gemini 3 Pro", 2439)
	report.SetScore("livecodebench_pro", "", "GPT-5.2", 2393)

	report.SetScore("scicode", "", "Gemini 3.1 Pro", 59)
	report.SetScore("scicode", "", "Gemini 3 Pro", 56)
	report.SetScore("scicode", "", "Opus 4.6", 52)
	report.SetScore("scicode", "", "Sonnet 4.6", 47)
	report.SetScore("scicode", "", "GPT-5.2", 52)

	// 🤖 Agent
	report.SetScore("apex_agents", "", "Gemini 3.1 Pro", 33.5)
	report.SetScore("apex_agents", "", "Gemini 3 Pro", 18.4)
	report.SetScore("apex_agents", "", "Opus 4.6", 29.8)
	report.SetScore("apex_agents", "", "GPT-5.2", 23.0)

	report.SetScore("gdpval_aa", "", "Gemini 3.1 Pro", 1317)
	report.SetScore("gdpval_aa", "", "Gemini 3 Pro", 1195)
	report.SetScore("gdpval_aa", "", "Sonnet 4.6", 1633)
	report.SetScore("gdpval_aa", "", "Opus 4.6", 1606)
	report.SetScore("gdpval_aa", "", "GPT-5.2", 1462)

	report.SetScore("t2_bench", "Retail", "Gemini 3.1 Pro", 90.8)
	report.SetScore("t2_bench", "Retail", "Gemini 3 Pro", 85.3)
	report.SetScore("t2_bench", "Retail", "Opus 4.6", 91.9)
	report.SetScore("t2_bench", "Retail", "Sonnet 4.6", 91.7)
	report.SetScore("t2_bench", "Retail", "GPT-5.2", 82.0)
	report.SetScore("t2_bench", "Telecom", "Gemini 3.1 Pro", 99.3)
	report.SetScore("t2_bench", "Telecom", "Gemini 3 Pro", 98.0)
	report.SetScore("t2_bench", "Telecom", "Opus 4.6", 99.3)
	report.SetScore("t2_bench", "Telecom", "Sonnet 4.6", 97.9)
	report.SetScore("t2_bench", "Telecom", "GPT-5.2", 98.7)

	report.SetScore("mcp_atlas", "", "Gemini 3.1 Pro", 69.2)
	report.SetScore("mcp_atlas", "", "Gemini 3 Pro", 54.1)
	report.SetScore("mcp_atlas", "", "Opus 4.6", 59.5)
	report.SetScore("mcp_atlas", "", "Sonnet 4.6", 61.3)
	report.SetScore("mcp_atlas", "", "GPT-5.2", 60.6)

	// 🔍 Search
	report.SetScore("browsecomp", "", "Gemini 3.1 Pro", 85.9)
	report.SetScore("browsecomp", "", "Gemini 3 Pro", 59.2)
	report.SetScore("browsecomp", "", "Opus 4.6", 84.0)
	report.SetScore("browsecomp", "", "Sonnet 4.6", 74.7)
	report.SetScore("browsecomp", "", "GPT-5.2", 65.8)

	// 🖼 Multimodal
	report.SetScore("mmmu_pro", "", "Gemini 3.1 Pro", 80.5)
	report.SetScore("mmmu_pro", "", "Gemini 3 Pro", 81.0)
	report.SetScore("mmmu_pro", "", "Opus 4.6", 73.9)
	report.SetScore("mmmu_pro", "", "Sonnet 4.6", 74.5)
	report.SetScore("mmmu_pro", "", "GPT-5.2", 79.5)

	// 📚 Knowledge
	report.SetScore("mmmlu", "", "Gemini 3.1 Pro", 92.6)
	report.SetScore("mmmlu", "", "Gemini 3 Pro", 91.8)
	report.SetScore("mmmlu", "", "Opus 4.6", 91.1)
	report.SetScore("mmmlu", "", "Sonnet 4.6", 89.3)
	report.SetScore("mmmlu", "", "GPT-5.2", 89.6)

	// 🧾 Long Context
	report.SetScore("mrcr_v2", "128k (avg)", "Gemini 3.1 Pro", 84.9)
	report.SetScore("mrcr_v2", "128k (avg)", "Gemini 3 Pro", 77.0)
	report.SetScore("mrcr_v2", "128k (avg)", "Sonnet 4.6", 84.9)
	report.SetScore("mrcr_v2", "128k (avg)", "Opus 4.6", 84.0)
	report.SetScore("mrcr_v2", "128k (avg)", "GPT-5.2", 83.8)
	report.SetScore("mrcr_v2", "1M (pointwise)", "Gemini 3.1 Pro", 26.3)
	report.SetScore("mrcr_v2", "1M (pointwise)", "Gemini 3 Pro", 26.3)

	// Render
	outPath := "test_benchmark_report.png"
	err := NewImageRenderer().RenderPNG(report, outPath)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	t.Logf("Generated %s (%.1f KB)", outPath, float64(info.Size())/1024)

	// Cleanup
	// os.Remove(outPath) // Comment out to inspect the image
}
