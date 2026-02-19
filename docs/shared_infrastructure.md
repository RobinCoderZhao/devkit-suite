# 共享基础设施 + 部署 + 商业化总览

## 1. 共享代码库设计

### 1.1 LLM 统一封装 — `pkg/llm`

```go
// pkg/llm/client.go
// 支持 OpenAI / Gemini / Claude / Ollama / MiniMax 的统一接口

type Provider string
const (
    ProviderOpenAI   Provider = "openai"
    ProviderGemini   Provider = "gemini"
    ProviderClaude   Provider = "claude"
    ProviderOllama   Provider = "ollama"
    ProviderMiniMax  Provider = "minimax"  // OpenAI-兼容 API
)

type Config struct {
    Provider   Provider `yaml:"provider"`
    Model      string   `yaml:"model"`
    APIKey     string   `yaml:"api_key"`
    BaseURL    string   `yaml:"base_url"`     // Ollama 或代理
    MaxRetries int      `yaml:"max_retries"`
    Timeout    time.Duration `yaml:"timeout"`
}

type Client interface {
    Generate(ctx context.Context, req Request) (*Response, error)
    GenerateJSON(ctx context.Context, req Request, out any) error  // 结构化输出
    StreamGenerate(ctx context.Context, req Request) (<-chan Chunk, error)
}

type Request struct {
    System   string            `json:"system,omitempty"`
    Messages []Message         `json:"messages"`
    MaxTokens int              `json:"max_tokens,omitempty"`
    Temperature float64        `json:"temperature,omitempty"`
    JSONMode    bool           `json:"json_mode,omitempty"`    // 强制 JSON 输出
}

type Response struct {
    Content    string `json:"content"`
    TokensUsed int    `json:"tokens_used"`
    Cost       float64 `json:"cost"`       // 估算费用
    Model      string  `json:"model"`
    LatencyMs  int64   `json:"latency_ms"`
}

// 工厂方法
func NewClient(cfg Config) (Client, error) {
    switch cfg.Provider {
    case ProviderOpenAI:
        return newOpenAIClient(cfg)
    case ProviderGemini:
        return newGeminiClient(cfg)
    case ProviderClaude:
        return newClaudeClient(cfg)
    case ProviderOllama:
        return newOllamaClient(cfg)
    case ProviderMiniMax:
        if cfg.BaseURL == "" {
            cfg.BaseURL = "https://api.minimax.io/v1"
        }
        return newOpenAIClient(cfg)  // MiniMax 复用 OpenAI 客户端
    default:
        return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
    }
}
```

### 1.2 爬虫引擎 — `pkg/scraper`

```go
// pkg/scraper/fetcher.go
type FetchOptions struct {
    UserAgent    string
    Timeout      time.Duration
    RetryCount   int
    ProxyURL     string            // 代理（反爬用）
    Headers      map[string]string
    WaitForJS    bool              // 是否需要 JS 渲染（需 Playwright）
}

type FetchResult struct {
    URL        string
    StatusCode int
    RawHTML    string
    CleanText  string              // 去除标签后的结构化文本
    FetchedAt  time.Time
    Duration   time.Duration
}

type Fetcher interface {
    Fetch(ctx context.Context, url string, opts FetchOptions) (*FetchResult, error)
}

// 两种实现：
// 1. HTTPFetcher  — 简单 HTTP GET（大部分场景够用）
// 2. BrowserFetcher — Playwright（JS 渲染页面）
```

### 1.3 Diff 引擎 — `pkg/differ`

```go
// pkg/differ/differ.go
type DiffResult struct {
    HasChanges  bool     `json:"has_changes"`
    AddedLines  []string `json:"added"`
    RemovedLines []string `json:"removed"`
    UnifiedDiff string   `json:"unified_diff"`
    Stats       DiffStats `json:"stats"`
}

type DiffStats struct {
    Additions int `json:"additions"`
    Deletions int `json:"deletions"`
    Changes   int `json:"changes"`
}

func TextDiff(oldText, newText string) DiffResult  // 文本 diff
func HTMLDiff(oldHTML, newHTML string) DiffResult   // HTML 结构 diff
```

### 1.4 通知层 — `pkg/notify`

```go
// pkg/notify/notify.go
type Channel string
const (
    ChannelTelegram Channel = "telegram"
    ChannelEmail    Channel = "email"
    ChannelSlack    Channel = "slack"
    ChannelWebhook  Channel = "webhook"
)

type Message struct {
    Title    string
    Body     string
    HTMLBody string   // 富文本 HTML（邮件使用）
    Format   string   // "markdown" / "html" / "plain"
    URL      string   // 可选：附带链接
}

type Notifier interface {
    Send(ctx context.Context, msg Message) error
}

// 统一发送器（根据用户配置选择渠道）
type Dispatcher struct {
    channels map[Channel]Notifier
}

func (d *Dispatcher) Dispatch(ctx context.Context, channels []Channel, msg Message) error {
    for _, ch := range channels {
        if notifier, ok := d.channels[ch]; ok {
            if err := notifier.Send(ctx, msg); err != nil {
                log.Error("notify failed", "channel", ch, "err", err)
            }
        }
    }
    return nil
}

// 邮件通知器支持：
// - Gmail SMTP (STARTTLS port 587)
// - RFC 2047 base64 编码（支持中文/emoji 标题）
// - Pre-rendered HTML body（来自 publisher）
// - 按订阅者语言偏好发送对应版本
```

---

## 2. 统一部署架构

### 2.1 MVP 阶段（单机 VPS）

```
Hetzner / Contabo VPS (€4.5-10/月, 2C4G)
┌─────────────────────────────────────────┐
│  Docker Compose                         │
│                                         │
│  ┌─────────┐  ┌──────────┐             │
│  │ NewsBot │  │ WatchBot │             │
│  │ (cron)  │  │ API+Worker│             │
│  └─────────┘  └──────────┘             │
│                                         │
│  ┌──────────┐  ┌─────────────────┐     │
│  │ PostgreSQL│  │ Nginx (反代)    │     │
│  └──────────┘  └─────────────────┘     │
│                                         │
│  Caddy/Nginx: SSL + 反向代理             │
│  *.your-domain.com                      │
│  - api.your-domain.com → WatchBot:8080  │
│  - app.your-domain.com → Web:3000       │
└─────────────────────────────────────────┘
```

### 2.2 CI/CD（GitHub Actions）

```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  build-cli:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - uses: goreleaser/goreleaser-action@v5
        with: { args: release }
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  build-docker:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKER_USER }}
          password: ${{ secrets.DOCKER_PASS }}
      - uses: docker/build-push-action@v5
        with:
          push: true
          tags: yourname/watchbot:${{ github.ref_name }}
```

---

## 3. 统一支付体系

### 3.1 支付渠道选择

| 产品 | 支付方式 | 平台 | 理由 |
|------|---------|------|------|
| NewsBot Pro | 订阅 $9/月 | Stripe | 全球用户，经常性收入 |
| DevKit Pro | License Key $12/月 | Paddle | 处理全球税务 |
| MCP 模板 | 一次性 $29-299 | Gumroad | 最快上架 |
| WatchBot | 订阅 $19-49/月 | Stripe | SaaS 标准 |

### 3.2 Stripe 集成简要

```go
// internal/billing/stripe.go
func CreateCheckoutSession(userID int, plan string) (string, error) {
    priceID := getPriceID(plan) // 从配置映射
    params := &stripe.CheckoutSessionParams{
        Mode: stripe.String("subscription"),
        LineItems: []*stripe.CheckoutSessionLineItemParams{
            { Price: stripe.String(priceID), Quantity: stripe.Int64(1) },
        },
        SuccessURL: stripe.String("https://app.your.com/success"),
        CancelURL:  stripe.String("https://app.your.com/pricing"),
        ClientReferenceID: stripe.String(fmt.Sprintf("%d", userID)),
    }
    session, err := session.New(params)
    return session.URL, err
}

// Webhook 处理
func HandleWebhook(w http.ResponseWriter, r *http.Request) {
    event := stripe.ConstructEvent(body, sig, webhookSecret)
    switch event.Type {
    case "checkout.session.completed":
        // 激活用户订阅
    case "customer.subscription.deleted":
        // 降级到免费版
    case "invoice.payment_failed":
        // 发送催付邮件
    }
}
```

---

## 4. 推广总体策略

### 4.1 品牌建设

| 资产 | 说明 | 优先级 |
|------|------|--------|
| GitHub 组织 | `github.com/devkit-suite` | 🟢 高 |
| 域名 | `devkit.tools` 或类似 | 🟢 高 |
| Logo | 用 AI 生成，简洁 | 🟡 中 |
| Landing Page | 产品矩阵展示页 | 🟢 高 |
| Twitter/X 账号 | 每日发内容 | 🟢 高 |

### 4.2 内容策略

```
每周内容节奏：
┌─────────┬─────────────────────────────────────┐
│ 周一     │ AI 行业周回顾（从 NewsBot 数据生成）   │
│ 周三     │ 技术教程（CLI 工具使用 / MCP 开发）    │
│ 周五     │ 竞品洞察（从 WatchBot 数据生成）       │
│ 周日     │ 构建日志 (Build in Public)            │
└─────────┴─────────────────────────────────────┘
```

### 4.3 社区运营

| 社区 | 人群 | 打法 |
|------|------|------|
| **r/golang** | Go 开发者 | 分享 DevKit + MCP 技术博客 |
| **r/SaaS** | SaaS 创始人 | 分享 WatchBot + 竞品洞察 |
| **r/OpenAI, r/ClaudeAI** | AI 开发者 | 分享 NewsBot 日报 |
| **Twitter AI 圈** | 全部 | Build in Public 日志 |
| **IndieHackers** | 独立开发者 | 收入进度分享 |
| **Product Hunt** | 产品爱好者 | 每个产品一次 Launch |
| **掘金/知乎/即刻** | 中文用户 | 中文技术文章 + 产品推广 |

---

## 5. 风险管理

| 风险 | 概率 | 影响 | 应对 |
|------|------|------|------|
| LLM API 成本飙升 | 中 | 高 | 支持 Ollama 本地模型，设置用量上限 |
| 爬虫被封 | 中 | 中 | 轮换 User-Agent，支持代理池 |
| 竞品出现 | 高 | 低 | 速度 > 完美，先发优势 |
| 无人付费 | 中 | 高 | 先验证需求（NewsBot 免费），再投入开发 |
| 技术债务 | 中 | 中 | 共享库设计，统一接口，写测试 |
| LLM 分析不准确 | 高 | 高 | 人工审核机制 + 用户反馈循环优化 prompt |

---

## 6. 第一步行动清单

### 今天（30 分钟准备）

- [ ] 注册 `github.com/devkit-suite` 组织
- [ ] 创建 Telegram Bot（@BotFather）
- [ ] 创建 Gumroad 账号

### 本周（MVP 启动）

- [ ] 初始化 Go monorepo 项目 + go.mod
- [ ] 实现 `pkg/llm` 统一封装（先支持 OpenAI）
- [ ] 实现 NewsBot 核心（爬虫 + 摘要 + Telegram 推送）
- [ ] 部署到 VPS，设置 cron
- [ ] 在 Reddit + Twitter 发布首期日报

### 第 2-4 周（品牌建设）

- [ ] 开发 `devkit commit` + `devkit review`
- [ ] GoReleaser 自动构建 + Homebrew Tap
- [ ] GitHub 开源 + Show HN / Product Hunt

### 第 5-8 周（付费产品）

- [ ] 从 robotIM 提取 MCP 框架 → 模板包
- [ ] 开发 WatchBot 核心 Pipeline
- [ ] Web 仪表盘 + Stripe 接入
- [ ] Landing Page 上线
