// NewsBot — AI 热点日报 Bot
//
// Usage:
//
//	newsbot run          # 运行一次：抓取 → 分析 → 发布
//	newsbot serve        # 以定时任务模式运行（每天 8:00 执行）
//	newsbot version      # 显示版本
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/analyzer"
	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/publisher"
	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/scheduler"
	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/sources"
	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/store"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/llm"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/notify"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: newsbot <run|serve|version>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		if err := runOnce(); err != nil {
			slog.Error("run failed", "error", err)
			os.Exit(1)
		}
	case "serve":
		serve()
	case "version":
		fmt.Printf("newsbot %s\n", version)
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

// NewsBotConfig holds all configuration for NewsBot.
type NewsBotConfig struct {
	LLM      llm.Config
	Telegram notify.TelegramConfig
	Email    notify.EmailConfig
	DBPath   string
}

func loadConfig() NewsBotConfig {
	return NewsBotConfig{
		LLM: llm.Config{
			Provider:    llm.Provider(getEnv("LLM_PROVIDER", "openai")),
			Model:       getEnv("LLM_MODEL", "gpt-4o-mini"),
			APIKey:      os.Getenv("LLM_API_KEY"),
			MaxRetries:  3,
			Timeout:     120 * time.Second,
			MaxTokens:   4096,
			Temperature: 0.3,
		},
		Telegram: notify.TelegramConfig{
			BotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
			ChannelID: os.Getenv("TELEGRAM_CHANNEL_ID"),
		},
		Email: notify.EmailConfig{
			SMTPHost: getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort: getEnv("SMTP_PORT", "465"),
			From:     getEnv("SMTP_FROM", "robin254817@gmail.com"),
			Password: os.Getenv("SMTP_PASSWORD"),
			To:       os.Getenv("SMTP_TO"),
		},
		DBPath: getEnv("NEWSBOT_DB", "newsbot.db"),
	}
}

func runOnce() error {
	ctx := context.Background()
	cfg := loadConfig()

	slog.Info("starting NewsBot run")

	// 1. Initialize data sources (8 diverse AI news feeds)
	registry := sources.NewRegistry()
	registry.Register(sources.NewHackerNewsSource(30))
	registry.Register(sources.NewRSSSource("TechCrunch AI", "https://techcrunch.com/category/artificial-intelligence/feed/"))
	registry.Register(sources.NewRSSSource("MIT Tech Review", "https://www.technologyreview.com/topic/artificial-intelligence/feed"))
	registry.Register(sources.NewRSSSource("The Verge AI", "https://www.theverge.com/rss/ai-artificial-intelligence/index.xml"))
	registry.Register(sources.NewRSSSource("Ars Technica AI", "https://feeds.arstechnica.com/arstechnica/technology-lab"))
	registry.Register(sources.NewRSSSource("VentureBeat AI", "https://venturebeat.com/category/ai/feed/"))
	registry.Register(sources.NewRSSSource("OpenAI Blog", "https://openai.com/blog/rss.xml"))
	registry.Register(sources.NewRSSSource("Google AI Blog", "https://blog.google/technology/ai/rss/"))

	// 2. Fetch articles
	slog.Info("fetching articles from all sources")
	articles, err := registry.FetchAll(ctx)
	if err != nil {
		return fmt.Errorf("fetch articles: %w", err)
	}
	slog.Info("fetched articles", "count", len(articles))

	// 3. Store articles
	db, err := store.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	saved, err := db.SaveArticles(ctx, articles)
	if err != nil {
		slog.Warn("failed to save some articles", "error", err)
	}
	slog.Info("saved articles", "new", saved, "total", len(articles))

	// 4. Analyze with LLM
	if cfg.LLM.APIKey == "" {
		slog.Warn("LLM API key not set, skipping analysis")
		return nil
	}

	llmClient, err := llm.NewClient(cfg.LLM)
	if err != nil {
		return fmt.Errorf("create LLM client: %w", err)
	}
	defer llmClient.Close()

	a := analyzer.NewAnalyzer(llmClient)
	digest, err := a.Analyze(ctx, articles)
	if err != nil {
		return fmt.Errorf("analyze articles: %w", err)
	}
	slog.Info("analysis complete", "headlines", len(digest.Headlines), "tokens", digest.TokensUsed, "cost", digest.Cost)

	// 5. Save digest
	if err := db.SaveDigest(ctx, digest); err != nil {
		slog.Warn("failed to save digest", "error", err)
	}

	// 6. Publish
	dispatcher := notify.NewDispatcher()
	var channels []notify.Channel

	if cfg.Telegram.BotToken != "" {
		dispatcher.Register(notify.NewTelegramNotifier(cfg.Telegram))
		channels = append(channels, notify.ChannelTelegram)
	}

	if cfg.Email.From != "" && cfg.Email.Password != "" && cfg.Email.To != "" {
		dispatcher.Register(notify.NewEmailNotifier(cfg.Email))
		channels = append(channels, notify.ChannelEmail)
	}

	if len(channels) > 0 {
		pub := publisher.NewPublisher(dispatcher, channels)
		if err := pub.Publish(ctx, digest); err != nil {
			return fmt.Errorf("publish digest: %w", err)
		}
		slog.Info("digest published", "channels", len(channels))
	} else {
		// Print to stdout if no channels configured
		fmt.Println(publisher.FormatDigest(digest))
	}

	return nil
}

func serve() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sched := scheduler.NewScheduler()
	sched.Add(scheduler.Job{
		Name:     "daily-digest",
		Schedule: "0 8 * * *",
		Fn: func(ctx context.Context) error {
			return runOnce()
		},
	})

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutdown signal received")
		sched.Stop()
		cancel()
	}()

	slog.Info("NewsBot serving, will run digest every 24 hours")
	sched.Start(ctx, 24*time.Hour)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
