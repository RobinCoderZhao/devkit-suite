// NewsBot is an AI-powered daily news aggregator that fetches, analyzes,
// and distributes AI news digests via email in multiple languages.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/analyzer"
	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/i18n"
	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/publisher"
	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/sources"
	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/store"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/llm"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/notify"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "run":
		err = runOnce()
	case "subscribe":
		err = cmdSubscribe()
	case "unsubscribe":
		err = cmdUnsubscribe()
	case "subscribers":
		err = cmdListSubscribers()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		slog.Error("run failed", "error", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`NewsBot - AI News Daily Digest

Usage: newsbot <command> [options]

Commands:
  run                     Fetch, analyze, and send daily digest
  subscribe               Add email subscriber
    --email=<addr>        Email address (required)
    --lang=<codes>        Language codes, comma-separated (default: zh)
                          Supported: zh, en, ja, ko, de, es
  unsubscribe             Remove email subscriber
    --email=<addr>        Email address (required)
  subscribers             List all active subscribers
  help                    Show this help

Environment Variables:
  LLM_PROVIDER     LLM provider: openai, minimax, gemini, claude (default: openai)
  LLM_API_KEY      API key for the LLM provider
  LLM_MODEL        Model name (default: gpt-4o-mini)
  NEWSBOT_DB       SQLite database path (default: newsbot.db)
  SMTP_HOST        SMTP server host (default: smtp.gmail.com)
  SMTP_PORT        SMTP port: 465 or 587 (default: 587)
  SMTP_FROM        Sender email (default: robin254817@gmail.com)
  SMTP_PASSWORD    SMTP app password
  SMTP_TO          Legacy: default recipient (use 'subscribe' command instead)`)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// NewsBotConfig holds all configuration for NewsBot.
type NewsBotConfig struct {
	LLM    llm.Config
	Email  notify.EmailConfig
	DBPath string
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
		Email: notify.EmailConfig{
			SMTPHost: getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort: getEnv("SMTP_PORT", "587"),
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

	// 5. Get subscribers and determine needed languages
	subscribers, _ := db.GetActiveSubscribers(ctx)

	// Add legacy SMTP_TO as a zh subscriber if no DB subscribers exist
	if len(subscribers) == 0 && cfg.Email.To != "" {
		subscribers = append(subscribers, store.Subscriber{
			Email:     cfg.Email.To,
			Languages: "zh",
			Active:    true,
		})
	}

	// Collect unique languages needed
	langSet := map[i18n.Language]bool{i18n.LangEN: true} // default English
	for _, sub := range subscribers {
		for _, l := range sub.LanguageList() {
			langSet[i18n.Language(l)] = true
		}
	}
	var neededLangs []i18n.Language
	for l := range langSet {
		neededLangs = append(neededLangs, l)
	}
	slog.Info("languages needed", "langs", neededLangs, "subscribers", len(subscribers))

	// 6. Translate to all needed languages
	translator := i18n.NewTranslator(llmClient)
	digests := translator.TranslateAll(ctx, digest, neededLangs)
	slog.Info("translation complete", "languages", len(digests))

	// 7. Save all language versions
	for lang, d := range digests {
		if err := db.SaveDigest(ctx, d, string(lang)); err != nil {
			slog.Warn("failed to save digest", "lang", lang, "error", err)
		}
	}

	// 8. Publish to subscribers
	if len(subscribers) > 0 && cfg.Email.Password != "" {
		dispatcher := notify.NewDispatcher()
		dispatcher.SetEmailConfig(cfg.Email)
		pub := publisher.NewPublisher(dispatcher)

		sent := 0
		for _, sub := range subscribers {
			for _, langStr := range sub.LanguageList() {
				lang := i18n.Language(langStr)
				d, ok := digests[lang]
				if !ok {
					d = digest // Fallback to Chinese
				}
				if err := pub.PublishToEmail(ctx, d, lang, sub.Email); err != nil {
					slog.Error("email send failed", "email", sub.Email, "lang", lang, "error", err)
				} else {
					slog.Info("email sent", "email", sub.Email, "lang", lang)
					sent++
				}
			}
		}
		slog.Info("digest published", "emails_sent", sent)
	} else {
		// Print to stdout if no subscribers/email configured
		fmt.Println(publisher.FormatDigest(digest, i18n.LangEN))
	}

	return nil
}

// --- CLI Commands ---

func cmdSubscribe() error {
	email, lang := "", "zh"
	for _, arg := range os.Args[2:] {
		if strings.HasPrefix(arg, "--email=") {
			email = strings.TrimPrefix(arg, "--email=")
		} else if strings.HasPrefix(arg, "--lang=") {
			lang = strings.TrimPrefix(arg, "--lang=")
		}
	}
	if email == "" {
		return fmt.Errorf("--email is required. Usage: newsbot subscribe --email=user@example.com --lang=zh,en")
	}

	// Validate languages
	langs := i18n.ParseLanguages(lang)
	langStrs := make([]string, len(langs))
	for i, l := range langs {
		langStrs[i] = string(l)
	}
	langCSV := strings.Join(langStrs, ",")

	cfg := loadConfig()
	db, err := store.New(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.AddSubscriber(context.Background(), email, langCSV); err != nil {
		return fmt.Errorf("add subscriber: %w", err)
	}

	fmt.Printf("✅ Subscribed: %s (languages: %s)\n", email, langCSV)
	for _, l := range langs {
		fmt.Printf("   • %s — %s\n", l, i18n.LanguageName(l))
	}
	return nil
}

func cmdUnsubscribe() error {
	email := ""
	for _, arg := range os.Args[2:] {
		if strings.HasPrefix(arg, "--email=") {
			email = strings.TrimPrefix(arg, "--email=")
		}
	}
	if email == "" {
		return fmt.Errorf("--email is required. Usage: newsbot unsubscribe --email=user@example.com")
	}

	cfg := loadConfig()
	db, err := store.New(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.RemoveSubscriber(context.Background(), email); err != nil {
		return fmt.Errorf("remove subscriber: %w", err)
	}

	fmt.Printf("✅ Unsubscribed: %s\n", email)
	return nil
}

func cmdListSubscribers() error {
	cfg := loadConfig()
	db, err := store.New(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	subs, err := db.GetActiveSubscribers(context.Background())
	if err != nil {
		return err
	}

	if len(subs) == 0 {
		fmt.Println("No active subscribers.")
		return nil
	}

	fmt.Printf("Active subscribers (%d):\n", len(subs))
	for _, s := range subs {
		langs := s.LanguageList()
		var langNames []string
		for _, l := range langs {
			langNames = append(langNames, fmt.Sprintf("%s(%s)", l, i18n.LanguageName(i18n.Language(l))))
		}
		fmt.Printf("  📧 %s — %s\n", s.Email, strings.Join(langNames, ", "))
	}
	return nil
}
