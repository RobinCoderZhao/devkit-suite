// WatchBot — 竞品监控 Bot
//
// Usage:
//
//	watchbot check      # 运行一次全量检查
//	watchbot serve      # 以定时任务模式运行
//	watchbot targets    # 列出监控目标
//	watchbot version    # 显示版本
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/watchbot"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/llm"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/notify"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/scraper"
)

var version = "dev"

// Config holds WatchBot configuration.
type Config struct {
	Targets  []watchbot.Target
	LLM      llm.Config
	Telegram notify.TelegramConfig
	Interval time.Duration
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: watchbot <check|serve|targets|version>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "check":
		if err := runCheck(); err != nil {
			slog.Error("check failed", "error", err)
			os.Exit(1)
		}
	case "serve":
		serve()
	case "targets":
		listTargets()
	case "version":
		fmt.Printf("watchbot %s\n", version)
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func loadConfig() Config {
	return Config{
		Targets: defaultTargets(),
		LLM: llm.Config{
			Provider:    llm.Provider(getEnv("LLM_PROVIDER", "openai")),
			Model:       getEnv("LLM_MODEL", "gpt-4o-mini"),
			APIKey:      os.Getenv("LLM_API_KEY"),
			MaxRetries:  3,
			Timeout:     60 * time.Second,
			Temperature: 0.3,
		},
		Telegram: notify.TelegramConfig{
			BotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
			ChannelID: os.Getenv("TELEGRAM_CHANNEL_ID"),
		},
		Interval: 6 * time.Hour,
	}
}

func defaultTargets() []watchbot.Target {
	return []watchbot.Target{
		{ID: "openai-api", Name: "OpenAI API Docs", URL: "https://platform.openai.com/docs/api-reference", Category: "api_docs", Interval: "6h"},
		{ID: "openai-changelog", Name: "OpenAI Changelog", URL: "https://platform.openai.com/docs/changelog", Category: "changelog", Interval: "6h"},
		{ID: "anthropic-api", Name: "Anthropic API Docs", URL: "https://docs.anthropic.com/en/api", Category: "api_docs", Interval: "6h"},
		{ID: "gemini-api", Name: "Gemini API Docs", URL: "https://ai.google.dev/gemini-api/docs", Category: "api_docs", Interval: "6h"},
		{ID: "huggingface-blog", Name: "HuggingFace Blog", URL: "https://huggingface.co/blog", Category: "blog", Interval: "24h"},
	}
}

// snapshotCache stores the latest content for each target (in-memory for MVP)
var snapshotCache = make(map[string]string)

func runCheck() error {
	ctx := context.Background()
	cfg := loadConfig()

	slog.Info("starting WatchBot check", "targets", len(cfg.Targets))

	fetcher := scraper.NewHTTPFetcher()

	// Set up LLM client (optional)
	var llmClient llm.Client
	if cfg.LLM.APIKey != "" {
		var err error
		llmClient, err = llm.NewClient(cfg.LLM)
		if err != nil {
			slog.Warn("LLM client creation failed, running without analysis", "error", err)
		} else {
			defer llmClient.Close()
		}
	}

	// Set up notification
	dispatcher := notify.NewDispatcher()
	var channels []notify.Channel
	if cfg.Telegram.BotToken != "" {
		dispatcher.Register(notify.NewTelegramNotifier(cfg.Telegram))
		channels = append(channels, notify.ChannelTelegram)
	}

	pipeline := watchbot.NewPipeline(fetcher, llmClient, dispatcher, channels)

	alertCount := 0
	for _, target := range cfg.Targets {
		prev := snapshotCache[target.ID]
		alert, newContent, err := pipeline.Check(ctx, target, prev)
		if err != nil {
			slog.Error("check failed", "target", target.Name, "error", err)
			continue
		}
		snapshotCache[target.ID] = newContent
		if alert != nil {
			alertCount++
			slog.Info("alert generated", "target", target.Name, "severity", alert.Severity)
			if len(channels) == 0 {
				fmt.Printf("\n%s [%s] %s\n%s\n\n", severityEmoji(alert.Severity), alert.Severity, alert.TargetName, alert.Analysis)
			}
		}
	}

	slog.Info("check complete", "targets", len(cfg.Targets), "alerts", alertCount)
	return nil
}

func serve() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := loadConfig()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutdown signal received")
		cancel()
	}()

	slog.Info("WatchBot serving", "interval", cfg.Interval, "targets", len(cfg.Targets))

	// Run once immediately
	runCheck()

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCheck()
		}
	}
}

func listTargets() {
	targets := defaultTargets()
	fmt.Printf("监控目标 (%d):\n\n", len(targets))
	for i, t := range targets {
		fmt.Printf("  %d. [%s] %s\n     URL: %s\n     间隔: %s\n\n", i+1, t.Category, t.Name, t.URL, t.Interval)
	}
}

func severityEmoji(s string) string {
	switch s {
	case "critical":
		return "🔴"
	case "important":
		return "🟡"
	case "minor":
		return "🟢"
	default:
		return "⚪"
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
