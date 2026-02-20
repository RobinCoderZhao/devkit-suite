# 开发计划 — Phase 0：项目初始化 + 共享基础设施

> **本文档是开发执行手册，后续开发时直接按步骤执行。**
>
> Go 版本：**1.25** | 项目根目录：`devkit-suite/`

---

## Step 0.1：初始化 Monorepo

### 操作

```bash
cd devkit-suite
go mod init github.com/RobinCoderZhao/devkit-suite
```

### 创建目录结构

```
devkit-suite/
├── cmd/
│   ├── newsbot/                  # 产品 1 入口
│   ├── devkit/                   # 产品 2 入口
│   └── watchbot/                 # 产品 4 入口
├── pkg/                          # 共享公共库
│   ├── llm/                      # LLM 多模型封装
│   ├── scraper/                  # 爬虫引擎
│   ├── differ/                   # Diff 引擎
│   ├── notify/                   # 通知层
│   └── storage/                  # 存储抽象
├── internal/                     # 各产品内部逻辑
│   ├── newsbot/
│   ├── devkit/
│   └── watchbot/
├── templates/                    # 产品 3：MCP 模板包
│   └── mcp-server/
├── web/                          # 产品 4 前端
│   └── watchbot-dashboard/
├── deploy/
│   ├── docker/
│   └── scripts/
├── configs/
│   └── config.example.yaml
├── docs/
├── .github/
│   └── workflows/
├── Makefile
├── go.mod
└── README.md
```

### Makefile

```makefile
.PHONY: all build-newsbot build-devkit build-watchbot test lint clean

GO=go
GOFLAGS=-trimpath -ldflags="-s -w"

all: build-newsbot build-devkit build-watchbot

build-newsbot:
 $(GO) build $(GOFLAGS) -o bin/newsbot ./cmd/newsbot

build-devkit:
 $(GO) build $(GOFLAGS) -o bin/devkit ./cmd/devkit

build-watchbot:
 $(GO) build $(GOFLAGS) -o bin/watchbot ./cmd/watchbot

test:
 $(GO) test ./... -v -count=1

lint:
 golangci-lint run ./...

clean:
 rm -rf bin/
```

---

## Step 0.2：实现 `pkg/llm` — LLM 统一封装

### 文件清单

| 文件 | 职责 |
|------|------|
| `pkg/llm/client.go` | 接口定义 + 工厂方法 |
| `pkg/llm/openai.go` | OpenAI 实现 |
| `pkg/llm/gemini.go` | Gemini 实现 |
| `pkg/llm/claude.go` | Claude 实现（预留） |
| `pkg/llm/ollama.go` | Ollama 本地模型实现（预留） |
| `pkg/llm/retry.go` | 重试 + 限流 + 错误处理 |
| `pkg/llm/cost.go` | Token 用量 + 费用估算 |
| `pkg/llm/llm_test.go` | 单元测试 |

### 核心接口

```go
// pkg/llm/client.go
package llm

type Provider string
const (
    OpenAI  Provider = "openai"
    Gemini  Provider = "gemini"
    Claude  Provider = "claude"
    Ollama  Provider = "ollama"
)

type Config struct {
    Provider    Provider      `yaml:"provider"`
    Model       string        `yaml:"model"`
    APIKey      string        `yaml:"api_key"`
    BaseURL     string        `yaml:"base_url"`
    MaxRetries  int           `yaml:"max_retries"`
    Timeout     time.Duration `yaml:"timeout"`
    MaxTokens   int           `yaml:"max_tokens"`
    Temperature float64       `yaml:"temperature"`
}

type Client interface {
    Generate(ctx context.Context, req *Request) (*Response, error)
    GenerateJSON(ctx context.Context, req *Request, out any) error
}

type Request struct {
    System      string    `json:"system,omitempty"`
    Messages    []Message `json:"messages"`
    MaxTokens   int       `json:"max_tokens,omitempty"`
    Temperature float64   `json:"temperature,omitempty"`
    JSONMode    bool      `json:"json_mode,omitempty"`
}

type Message struct {
    Role    string `json:"role"`    // system / user / assistant
    Content string `json:"content"`
}

type Response struct {
    Content    string  `json:"content"`
    TokensIn   int     `json:"tokens_in"`
    TokensOut  int     `json:"tokens_out"`
    Cost       float64 `json:"cost"`
    Model      string  `json:"model"`
    LatencyMs  int64   `json:"latency_ms"`
}

func NewClient(cfg Config) (Client, error) {
    switch cfg.Provider {
    case OpenAI:  return newOpenAIClient(cfg)
    case Gemini:  return newGeminiClient(cfg)
    default:      return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
    }
}
```

### 依赖

```
go get github.com/sashabaranov/go-openai
go get github.com/google/generative-ai-go
go get google.golang.org/api
```

### 验证标准

- [ ] `go test ./pkg/llm/...` 通过（Mock 测试）
- [ ] OpenAI 真实调用测试通过
- [ ] Gemini 真实调用测试通过
- [ ] 重试逻辑在 429 错误时正确重试

---

## Step 0.3：实现 `pkg/scraper` — 爬虫引擎

### 文件清单

| 文件 | 职责 |
|------|------|
| `pkg/scraper/scraper.go` | 接口定义 |
| `pkg/scraper/http_fetcher.go` | HTTP 抓取实现 |
| `pkg/scraper/parser.go` | HTML → 结构化文本提取 |
| `pkg/scraper/ratelimit.go` | 请求限速 |
| `pkg/scraper/scraper_test.go` | 测试 |

### 核心接口

```go
// pkg/scraper/scraper.go
package scraper

type FetchOptions struct {
    UserAgent  string            `yaml:"user_agent"`
    Timeout    time.Duration     `yaml:"timeout"`
    RetryCount int               `yaml:"retry_count"`
    Headers    map[string]string `yaml:"headers"`
}

type FetchResult struct {
    URL        string
    StatusCode int
    RawHTML    string
    CleanText  string        // 提取后的结构化文本
    Title      string
    FetchedAt  time.Time
    Duration   time.Duration
}

type Fetcher interface {
    Fetch(ctx context.Context, url string, opts *FetchOptions) (*FetchResult, error)
}
```

### 依赖

```
go get github.com/PuerkitoBio/goquery
go get golang.org/x/time/rate
```

---

## Step 0.4：实现 `pkg/differ` — Diff 引擎

### 文件清单

| 文件 | 职责 |
|------|------|
| `pkg/differ/differ.go` | 接口 + 文本 diff |
| `pkg/differ/html_differ.go` | HTML 结构化 diff |
| `pkg/differ/differ_test.go` | 测试 |

### 依赖

```
go get github.com/sergi/go-diff/diffmatchpatch
```

---

## Step 0.5：实现 `pkg/notify` — 通知层

### 文件清单

| 文件 | 职责 |
|------|------|
| `pkg/notify/notify.go` | 接口定义 + Dispatcher |
| `pkg/notify/telegram.go` | Telegram Bot 推送 |
| `pkg/notify/email.go` | SMTP 邮件推送 |
| `pkg/notify/slack.go` | Slack Webhook（预留） |
| `pkg/notify/webhook.go` | 通用 Webhook |
| `pkg/notify/notify_test.go` | 测试 |

### 依赖

```
go get gopkg.in/telebot.v4
go get github.com/jordan-wright/email
```

---

## Step 0.6：实现 `pkg/storage` — 存储抽象

### 文件清单

| 文件 | 职责 |
|------|------|
| `pkg/storage/storage.go` | 接口定义 |
| `pkg/storage/sqlite.go` | SQLite 实现 |
| `pkg/storage/postgres.go` | PostgreSQL 实现（预留） |
| `pkg/storage/migrate.go` | 数据库迁移管理 |

### 依赖

```
go get github.com/mattn/go-sqlite3
go get github.com/jmoiron/sqlx
```

---

## Step 0.7：配置管理

### 配置文件

```yaml
# configs/config.example.yaml
app:
  name: "devkit-suite"
  env: "development"          # development / production
  log_level: "info"           # debug / info / warn / error

llm:
  provider: "openai"
  model: "gpt-4o-mini"
  api_key: "${OPENAI_API_KEY}"     # 支持环境变量引用
  max_retries: 3
  timeout: "30s"

scraper:
  user_agent: "DevkitSuite/1.0"
  timeout: "15s"
  retry_count: 2

notify:
  telegram:
    bot_token: "${TELEGRAM_BOT_TOKEN}"
    channel_id: "${TELEGRAM_CHANNEL_ID}"
  email:
    smtp_host: ""
    smtp_port: 587
    username: ""
    password: ""

storage:
  driver: "sqlite"              # sqlite / postgres
  sqlite:
    path: "./data/devkit.db"
  postgres:
    dsn: "${DATABASE_URL}"
```

### 配置加载

```go
// pkg/config/config.go
package config

type Config struct {
    App     AppConfig     `yaml:"app"`
    LLM     llm.Config    `yaml:"llm"`
    Scraper ScraperConfig `yaml:"scraper"`
    Notify  NotifyConfig  `yaml:"notify"`
    Storage StorageConfig `yaml:"storage"`
}

func Load(path string) (*Config, error) {
    // 1. 读取 YAML 文件
    // 2. 展开环境变量 ${VAR}
    // 3. 验证必填字段
}
```

### 依赖

```
go get gopkg.in/yaml.v3
```

---

## Phase 0 完成检查清单

- [ ] `go mod init` 完成
- [ ] 目录结构创建完毕
- [ ] Makefile 可用
- [ ] `pkg/llm` 编译通过 + 测试通过
- [ ] `pkg/scraper` 编译通过 + 测试通过
- [ ] `pkg/differ` 编译通过 + 测试通过
- [ ] `pkg/notify` 编译通过 + 测试通过
- [ ] `pkg/storage` 编译通过 + 测试通过
- [ ] `configs/config.example.yaml` 创建完毕
- [ ] `go build ./...` 全部通过
- [ ] `go test ./pkg/...` 全部通过

---

## Phase 0 预计耗时

| 步骤 | 时间 |
|------|------|
| 0.1 初始化项目 | 30 分钟 |
| 0.2 LLM 封装 | 3-4 小时 |
| 0.3 爬虫引擎 | 2-3 小时 |
| 0.4 Diff 引擎 | 1-2 小时 |
| 0.5 通知层 | 2-3 小时 |
| 0.6 存储层 | 2-3 小时 |
| 0.7 配置管理 | 1 小时 |
| **总计** | **约 12-16 小时（2-3 天）** |
