// WatchBot — 竞品监控 Bot (V2 Multi-User)
//
// Usage:
//
//	watchbot add <url-or-text>       # 添加监控目标（URL 或自然语言）
//	watchbot remove <name>           # 删除竞品
//	watchbot list                    # 列出所有竞品及页面
//	watchbot subscribe               # 添加订阅者
//	watchbot unsubscribe             # 取消订阅
//	watchbot subscribers             # 列出订阅者
//	watchbot check                   # 运行一次全量检查
//	watchbot serve                   # 守护进程模式
//	watchbot version                 # 显示版本
package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/watchbot"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/llm"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/notify"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/scraper"
)

var version = "2.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "add":
		cmdAdd()
	case "remove":
		cmdRemove()
	case "list":
		cmdList()
	case "subscribe":
		cmdSubscribe()
	case "unsubscribe":
		cmdUnsubscribe()
	case "subscribers":
		cmdSubscribers()
	case "check":
		cmdCheck()
	case "serve":
		cmdServe()
	case "version":
		fmt.Printf("watchbot %s\n", version)
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`WatchBot — 竞品监控 (V2)

Usage:
  watchbot add <url-or-text>                     添加监控目标
  watchbot remove --name=<name>                  删除竞品
  watchbot list                                  列出所有竞品
  watchbot subscribe --email=<e> --competitors=<names>  订阅
  watchbot unsubscribe --email=<e>               取消订阅
  watchbot subscribers                           列出订阅者
  watchbot check                                 运行一次全量检查
  watchbot serve                                 守护进程模式
  watchbot version                               版本`)
}

// --- Database ---

func openDB() (*sql.DB, *watchbot.Store) {
	dbPath := getEnv("WATCHBOT_DB", "data/watchbot.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		slog.Error("open database failed", "path", dbPath, "error", err)
		os.Exit(1)
	}
	store := watchbot.NewStore(db)
	ctx := context.Background()
	if err := store.InitDB(ctx); err != nil {
		slog.Error("init database failed", "error", err)
		os.Exit(1)
	}
	return db, store
}

// --- LLM ---

func newLLMClient() llm.Client {
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		return nil
	}
	cfg := llm.Config{
		Provider:    llm.Provider(getEnv("LLM_PROVIDER", "openai")),
		Model:       getEnv("LLM_MODEL", "gpt-4o-mini"),
		APIKey:      apiKey,
		MaxRetries:  3,
		Timeout:     60 * time.Second,
		Temperature: 0.3,
	}
	if cfg.Provider == "minimax" {
		cfg.BaseURL = "https://api.minimax.io/v1"
	}
	client, err := llm.NewClient(cfg)
	if err != nil {
		slog.Warn("LLM client creation failed", "error", err)
		return nil
	}
	return client
}

// --- Commands ---

func cmdAdd() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: watchbot add <url-or-natural-language>")
		fmt.Println("Examples:")
		fmt.Println("  watchbot add https://stripe.com/pricing")
		fmt.Println(`  watchbot add "监控 Gemini API 文档变化"`)
		os.Exit(1)
	}

	input := strings.Join(os.Args[2:], " ")
	ctx := context.Background()
	db, store := openDB()
	defer db.Close()

	if watchbot.IsURL(input) {
		// Direct URL mode
		fmt.Printf("🔍 验证 URL: %s\n", input)
		vr := watchbot.ValidateURL(ctx, input)
		if !vr.Valid {
			fmt.Printf("❌ URL 无效: %s\n", vr.Error)
			if vr.URL != "" {
				fmt.Printf("   标准化后: %s\n", vr.URL)
			}
			os.Exit(1)
		}
		domain := watchbot.ExtractDomain(vr.URL)
		pageType := watchbot.GuessPageType(vr.URL)
		name := promptInput(fmt.Sprintf("竞品名称 (默认: %s): ", domain))
		if name == "" {
			name = domain
		}
		compID, _ := store.AddCompetitor(ctx, name, domain)
		_, _ = store.AddPage(ctx, compID, vr.URL, pageType)
		fmt.Printf("✅ 已添加: %s [%s] %s\n", name, pageType, vr.URL)
	} else {
		// Natural language mode
		llmClient := newLLMClient()
		if llmClient == nil {
			fmt.Println("❌ 自然语言模式需要配置 LLM_API_KEY")
			fmt.Println("   或者直接使用 URL: watchbot add https://...")
			os.Exit(1)
		}
		defer llmClient.Close()

		resolver := watchbot.NewResolver(llmClient, watchbot.ResolverConfig{
			GoogleAPIKey: os.Getenv("GOOGLE_API_KEY"),
			GoogleCX:     os.Getenv("GOOGLE_CX"),
			BingAPIKey:   os.Getenv("BING_API_KEY"),
		})

		fmt.Printf("🤖 分析: \"%s\"\n", input)
		result, err := resolver.Resolve(ctx, input)
		if err != nil {
			fmt.Printf("❌ 解析失败: %v\n", err)
			os.Exit(1)
		}

		if result.Error != "" {
			fmt.Printf("❌ %s\n", result.Error)
			fmt.Println("   请提供具体信息，例如：")
			fmt.Println(`   watchbot add "监控 OpenAI API 文档变化"`)
			fmt.Println("   watchbot add https://openai.com/pricing")
			os.Exit(1)
		}

		if len(result.URLs) == 0 {
			fmt.Printf("🤔 识别到产品: %s，但无法确定 URL\n", result.Name)
			fmt.Println("   请手动输入 URL：watchbot add <url>")
			os.Exit(1)
		}

		// Show candidate and ask for confirmation
		fmt.Printf("\n🤖 建议监控 (来源: %s)：\n", result.Source)
		fmt.Printf("  [%s] %s\n", result.PageType, result.Name)
		for _, u := range result.URLs {
			fmt.Printf("  %s\n", u)
		}
		confirm := promptInput("\n确认添加？[Y/n]: ")
		if confirm != "" && strings.ToLower(confirm) != "y" {
			fmt.Println("已取消")
			return
		}

		domain := watchbot.ExtractDomain(result.URLs[0])
		compID, _ := store.AddCompetitor(ctx, result.Name, domain)
		for _, u := range result.URLs {
			pageType := watchbot.GuessPageType(u)
			_, _ = store.AddPage(ctx, compID, u, pageType)
		}
		fmt.Printf("✅ 已添加: %s (%d 个页面)\n", result.Name, len(result.URLs))
	}
}

func cmdRemove() {
	name := getFlag("--name")
	if name == "" {
		fmt.Println("Usage: watchbot remove --name=<competitor-name>")
		os.Exit(1)
	}
	ctx := context.Background()
	db, store := openDB()
	defer db.Close()

	if err := store.RemoveCompetitor(ctx, name); err != nil {
		fmt.Printf("❌ 删除失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 已删除: %s\n", name)
}

func cmdList() {
	ctx := context.Background()
	db, store := openDB()
	defer db.Close()

	competitors, err := store.ListCompetitors(ctx)
	if err != nil {
		slog.Error("list failed", "error", err)
		os.Exit(1)
	}

	if len(competitors) == 0 {
		fmt.Println("暂无监控目标。使用 watchbot add <url> 添加。")
		return
	}

	fmt.Printf("监控目标 (%d):\n\n", len(competitors))
	for i, c := range competitors {
		fmt.Printf("  %d. %s (%s)\n", i+1, c.Name, c.Domain)
		pages, _ := store.GetPagesByCompetitor(ctx, c.ID)
		for _, p := range pages {
			status := "✅"
			if p.Status != "active" {
				status = "⏸️"
			}
			checked := "未检查"
			if p.LastChecked != nil {
				checked = p.LastChecked.Format("2006-01-02 15:04")
			}
			fmt.Printf("     %s [%s] %s (最后检查: %s)\n", status, p.PageType, p.URL, checked)
		}
		fmt.Println()
	}
}

func cmdSubscribe() {
	email := getFlag("--email")
	competitors := getFlag("--competitors")
	if email == "" || competitors == "" {
		fmt.Println("Usage: watchbot subscribe --email=<email> --competitors=<name1,name2,...>")
		os.Exit(1)
	}

	ctx := context.Background()
	db, store := openDB()
	defer db.Close()

	subID, err := store.AddSubscriber(ctx, email)
	if err != nil {
		fmt.Printf("❌ 添加订阅者失败: %v\n", err)
		os.Exit(1)
	}

	names := strings.Split(competitors, ",")
	for _, name := range names {
		name = strings.TrimSpace(name)
		comp, err := store.GetCompetitor(ctx, name)
		if err != nil || comp == nil {
			fmt.Printf("⚠️ 竞品 \"%s\" 不存在，跳过\n", name)
			continue
		}
		if err := store.Subscribe(ctx, subID, comp.ID); err != nil {
			fmt.Printf("⚠️ 订阅 \"%s\" 失败: %v\n", name, err)
			continue
		}
		fmt.Printf("  ✅ %s\n", name)
	}
	fmt.Printf("\n📧 已订阅: %s\n", email)
}

func cmdUnsubscribe() {
	email := getFlag("--email")
	if email == "" {
		fmt.Println("Usage: watchbot unsubscribe --email=<email>")
		os.Exit(1)
	}

	ctx := context.Background()
	db, store := openDB()
	defer db.Close()

	if err := store.RemoveSubscriber(ctx, email); err != nil {
		fmt.Printf("❌ 取消订阅失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 已取消订阅: %s\n", email)
}

func cmdSubscribers() {
	ctx := context.Background()
	db, store := openDB()
	defer db.Close()

	subs, err := store.ListSubscribers(ctx)
	if err != nil {
		slog.Error("list subscribers failed", "error", err)
		os.Exit(1)
	}

	if len(subs) == 0 {
		fmt.Println("暂无订阅者。使用 watchbot subscribe 添加。")
		return
	}

	fmt.Printf("订阅者 (%d):\n\n", len(subs))
	for _, s := range subs {
		fmt.Printf("  📧 %s → %s\n", s.Email, strings.Join(s.CompetitorNames, ", "))
	}
}

func cmdCheck() {
	ctx := context.Background()
	db, store := openDB()
	defer db.Close()

	llmClient := newLLMClient()
	if llmClient != nil {
		defer llmClient.Close()
	}

	fetcher := scraper.NewHTTPFetcher()
	dispatcher := notify.NewDispatcher()

	// Setup email
	var channels []notify.Channel
	emailCfg := loadEmailConfig()
	if emailCfg.SMTPHost != "" {
		dispatcher.SetEmailConfig(emailCfg)
	}

	// Setup Telegram
	tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if tgToken != "" {
		dispatcher.Register(notify.NewTelegramNotifier(notify.TelegramConfig{
			BotToken:  tgToken,
			ChannelID: os.Getenv("TELEGRAM_CHANNEL_ID"),
		}))
		channels = append(channels, notify.ChannelTelegram)
	}

	pipeline := watchbot.NewGlobalPipeline(store, fetcher, llmClient, dispatcher, channels)
	if err := pipeline.RunCheck(ctx); err != nil {
		slog.Error("check failed", "error", err)
		os.Exit(1)
	}
}

func cmdServe() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutdown signal received")
		cancel()
	}()

	interval := 6 * time.Hour
	slog.Info("WatchBot serving", "interval", interval)

	// Run immediately
	cmdCheck()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cmdCheck()
		}
	}
}

// --- Helpers ---

func loadEmailConfig() notify.EmailConfig {
	return notify.EmailConfig{
		SMTPHost: getEnv("SMTP_HOST", ""),
		SMTPPort: getEnv("SMTP_PORT", "587"),
		From:     os.Getenv("SMTP_FROM"),
		Password: os.Getenv("SMTP_PASSWORD"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getFlag(name string) string {
	prefix := name + "="
	for _, arg := range os.Args[2:] {
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}
	return ""
}

func promptInput(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}
