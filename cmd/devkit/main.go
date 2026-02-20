// DevKit — AI-powered Developer CLI Toolkit
//
// Usage:
//
//	devkit commit     # AI 生成 commit message
//	devkit review     # AI 代码审查
//	devkit version    # 显示版本
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	devkitcfg "github.com/RobinCoderZhao/devkit-suite/internal/devkit/config"
	"github.com/RobinCoderZhao/devkit-suite/internal/devkit/git"
	"github.com/RobinCoderZhao/devkit-suite/internal/devkit/prompt"
	"github.com/RobinCoderZhao/devkit-suite/pkg/llm"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "devkit",
		Short: "AI-powered Developer CLI Toolkit",
		Long:  "DevKit 是一个 AI 驱动的开发者命令行工具套件，帮助你编写 commit message、审查代码等。",
	}

	rootCmd.AddCommand(commitCmd())
	rootCmd.AddCommand(reviewCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func commitCmd() *cobra.Command {
	var autoStage bool
	var direct bool

	cmd := &cobra.Command{
		Use:   "commit",
		Short: "AI 生成 conventional commit message",
		Long:  "分析 staged git diff，使用 LLM 生成符合 Conventional Commits 规范的 commit message。",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommit(autoStage, direct)
		},
	}

	cmd.Flags().BoolVarP(&autoStage, "all", "a", false, "自动 stage 所有变更")
	cmd.Flags().BoolVarP(&direct, "yes", "y", false, "不确认直接 commit")
	return cmd
}

func reviewCmd() *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "review",
		Short: "AI 代码审查",
		Long:  "分析 staged/unstaged 变更，使用 LLM 进行代码审查，输出评分和建议。",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReview(outputJSON)
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "输出 JSON 格式")
	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("devkit %s\n", version)
		},
	}
}

func runCommit(autoStage, direct bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg, err := devkitcfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	repo, err := git.OpenCurrent()
	if err != nil {
		return fmt.Errorf("❌ %w", err)
	}

	if autoStage {
		fmt.Println("📦 Staging all changes...")
		if err := repo.AddAll(ctx); err != nil {
			return fmt.Errorf("stage changes: %w", err)
		}
	}

	hasStagedChanges, err := repo.HasStagedChanges(ctx)
	if err != nil {
		return err
	}
	if !hasStagedChanges {
		fmt.Println("⚠️  没有 staged 的变更。使用 `git add` 或 `devkit commit -a` 来 stage 变更。")
		return nil
	}

	diff, err := repo.StagedDiff(ctx)
	if err != nil {
		return fmt.Errorf("get diff: %w", err)
	}

	if len(diff) > 15000 {
		diff = diff[:15000] + "\n... (truncated)"
	}

	files, _ := repo.StagedFiles(ctx)
	fmt.Printf("📝 Staged files (%d):\n", len(files))
	for _, f := range files {
		fmt.Printf("   %s\n", f)
	}

	fmt.Println("\n🤖 Generating commit message...")

	if cfg.LLM.APIKey == "" {
		return fmt.Errorf("❌ LLM API Key未设置。设置环境变量 LLM_API_KEY 或 OPENAI_API_KEY，或在 .devkit.yaml 中配置")
	}

	client, err := llm.NewClient(cfg.LLM)
	if err != nil {
		return fmt.Errorf("create LLM client: %w", err)
	}
	defer client.Close()

	resp, err := client.Generate(ctx, &llm.Request{
		Messages: []llm.Message{
			{Role: "user", Content: fmt.Sprintf(prompt.CommitPrompt, diff)},
		},
		Temperature: 0.3,
	})
	if err != nil {
		return fmt.Errorf("LLM generation failed: %w", err)
	}

	commitMsg := strings.TrimSpace(resp.Content)
	fmt.Printf("\n✨ Generated commit message:\n\n%s\n\n", commitMsg)
	fmt.Printf("📊 Tokens: %d in / %d out | Cost: $%.4f\n\n", resp.TokensIn, resp.TokensOut, resp.Cost)

	if direct {
		return repo.Commit(ctx, commitMsg)
	}

	fmt.Print("🚀 Use this commit message? [Y/n/e(dit)] ")
	var answer string
	fmt.Scanln(&answer)

	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "", "y", "yes":
		if err := repo.Commit(ctx, commitMsg); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
		fmt.Println("✅ Committed!")
	case "n", "no":
		fmt.Println("❌ Cancelled.")
	case "e", "edit":
		fmt.Println("📝 Launching editor (TODO: open $EDITOR)")
		// TODO: open editor with commitMsg pre-filled
	default:
		fmt.Println("❌ Cancelled.")
	}

	return nil
}

func runReview(outputJSON bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cfg, err := devkitcfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	repo, err := git.OpenCurrent()
	if err != nil {
		return fmt.Errorf("❌ %w", err)
	}

	// Try staged diff first, then working tree
	diff, err := repo.StagedDiff(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(diff) == "" {
		diff, err = repo.WorkingDiff(ctx)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(diff) == "" {
		fmt.Println("⚠️  没有检测到变更。")
		return nil
	}

	if len(diff) > 20000 {
		diff = diff[:20000] + "\n... (truncated)"
	}

	fmt.Println("🔍 AI Code Review in progress...")

	if cfg.LLM.APIKey == "" {
		return fmt.Errorf("❌ LLM API Key未设置。设置环境变量 LLM_API_KEY 或 OPENAI_API_KEY")
	}

	client, err := llm.NewClient(cfg.LLM)
	if err != nil {
		return fmt.Errorf("create LLM client: %w", err)
	}
	defer client.Close()

	resp, err := client.Generate(ctx, &llm.Request{
		Messages: []llm.Message{
			{Role: "user", Content: fmt.Sprintf(prompt.ReviewPrompt, diff)},
		},
		JSONMode: true,
	})
	if err != nil {
		return fmt.Errorf("LLM review failed: %w", err)
	}

	if outputJSON {
		fmt.Println(resp.Content)
		return nil
	}

	// Parse and display formatted review
	var review ReviewResult
	if err := json.Unmarshal([]byte(resp.Content), &review); err != nil {
		// Fallback: just print the raw response
		fmt.Println(resp.Content)
		return nil
	}

	printReview(review)
	fmt.Printf("\n📊 Tokens: %d in / %d out | Cost: $%.4f\n", resp.TokensIn, resp.TokensOut, resp.Cost)
	return nil
}

// ReviewResult holds the structured code review result.
type ReviewResult struct {
	Score      int      `json:"score"`
	Summary    string   `json:"summary"`
	Issues     []Issue  `json:"issues"`
	Highlights []string `json:"highlights"`
}

// Issue represents a code review issue.
type Issue struct {
	Severity    string `json:"severity"`
	File        string `json:"file"`
	Line        string `json:"line"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

func printReview(r ReviewResult) {
	scoreEmoji := "⚪"
	switch {
	case r.Score >= 9:
		scoreEmoji = "🟢"
	case r.Score >= 7:
		scoreEmoji = "🟡"
	case r.Score >= 5:
		scoreEmoji = "🟠"
	default:
		scoreEmoji = "🔴"
	}

	fmt.Printf("\n%s Score: %d/10 — %s\n\n", scoreEmoji, r.Score, r.Summary)

	if len(r.Issues) > 0 {
		fmt.Println("⚠️  Issues:")
		for i, issue := range r.Issues {
			sev := "🟢"
			switch issue.Severity {
			case "high":
				sev = "🔴"
			case "medium":
				sev = "🟡"
			}
			fmt.Printf("  %d. %s [%s] %s:%s\n", i+1, sev, issue.Severity, issue.File, issue.Line)
			fmt.Printf("     %s\n", issue.Description)
			if issue.Suggestion != "" {
				fmt.Printf("     💡 %s\n", issue.Suggestion)
			}
			fmt.Println()
		}
	}

	if len(r.Highlights) > 0 {
		fmt.Println("✅ Highlights:")
		for _, h := range r.Highlights {
			fmt.Printf("   • %s\n", h)
		}
	}
}

// slog is used for debug logging when needed
var _ = slog.Debug
