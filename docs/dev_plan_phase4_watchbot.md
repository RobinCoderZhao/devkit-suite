# 开发计划 — Phase 4：竞品监控 Bot（WatchBot）

> 前置依赖：Phase 0（`pkg/llm`, `pkg/scraper`, `pkg/differ`, `pkg/notify` 全部完成）
>
> 项目路径：`API-Change-Sentinel/cmd/watchbot/` + `API-Change-Sentinel/internal/watchbot/` + `API-Change-Sentinel/web/watchbot-dashboard/`

---

## Step 4.1：目录结构

```
cmd/watchbot/
└── main.go                         # 入口：支持 api / worker 子命令

internal/watchbot/
├── app.go                          # 应用初始化
├── api/                            # HTTP API 层
│   ├── server.go                   # 路由注册 + HTTP Server
│   ├── middleware.go               # JWT 认证 + CORS + 日志
│   ├── auth_handler.go             # POST /auth/register, /auth/login
│   ├── competitor_handler.go       # CRUD /competitors
│   ├── report_handler.go           # GET /reports, /reports/:id, /timeline
│   ├── billing_handler.go          # Stripe checkout / webhook
│   └── response.go                 # 统一 JSON 响应格式
├── model/                          # 数据模型
│   ├── user.go                     # User struct + 密码 hash
│   ├── competitor.go               # Competitor struct
│   ├── page.go                     # Page struct
│   ├── snapshot.go                 # Snapshot struct
│   └── analysis.go                 # Analysis struct
├── store/                          # 数据层
│   ├── store.go                    # Store 接口
│   ├── postgres.go                 # PostgreSQL 实现
│   ├── sqlite.go                   # SQLite 实现 (MVP)
│   ├── schema.sql                  # 建表 SQL
│   └── store_test.go
├── pipeline/                       # 后台监控流水线
│   ├── pipeline.go                 # Pipeline Runner 编排
│   ├── fetcher.go                  # 页面抓取 (调用 pkg/scraper)
│   ├── differ.go                   # 变更对比 (调用 pkg/differ)
│   ├── analyzer.go                 # LLM 智能分析
│   ├── prompts.go                  # 分析 Prompt 模板
│   ├── notifier.go                 # 通知分发
│   ├── discoverer.go               # 自动发现竞品的关键页面
│   └── pipeline_test.go
├── worker/                         # 后台 Worker
│   ├── worker.go                   # Cron 调度 + Pipeline 执行
│   └── worker_test.go
└── billing/                        # 支付
    ├── stripe.go                   # Stripe Checkout + Webhook
    └── plans.go                    # 套餐定义

web/watchbot-dashboard/             # 前端 (Next.js 或纯 HTML+JS)
├── index.html                      # Landing Page
├── app.html                        # 仪表盘主页
├── css/
├── js/
│   ├── api.js                      # 后端 API 调用封装
│   ├── auth.js                     # 登录注册逻辑
│   ├── competitors.js              # 竞品管理界面
│   └── reports.js                  # 报告查看界面
└── assets/
```

## Step 4.2：入口程序

```go
// cmd/watchbot/main.go
package main

// 子命令：
//   watchbot api       — 启动 HTTP API 服务（端口 8080）
//   watchbot worker    — 启动后台 Worker（Cron 调度抓取 + 分析）
//   watchbot migrate   — 执行数据库迁移
//   watchbot seed      — 插入测试数据

func main() {
    rootCmd := &cobra.Command{Use: "watchbot"}
    rootCmd.AddCommand(apiCmd, workerCmd, migrateCmd, seedCmd)
    rootCmd.Execute()
}
```

## Step 4.3：数据库 Schema

```sql
-- internal/watchbot/store/schema.sql

CREATE TABLE users (
    id            SERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    plan          TEXT DEFAULT 'free',
    stripe_customer_id TEXT,
    max_competitors INT DEFAULT 1,
    check_interval  INT DEFAULT 604800,    -- 免费: 7天, Growth: 1天, Pro: 12h
    created_at    TIMESTAMP DEFAULT NOW(),
    updated_at    TIMESTAMP DEFAULT NOW()
);

CREATE TABLE competitors (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    domain      TEXT NOT NULL,
    logo_url    TEXT,
    status      TEXT DEFAULT 'active',
    created_at  TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, domain)
);

CREATE TABLE pages (
    id              SERIAL PRIMARY KEY,
    competitor_id   INTEGER NOT NULL REFERENCES competitors(id) ON DELETE CASCADE,
    url             TEXT NOT NULL,
    page_type       TEXT NOT NULL,          -- pricing / features / blog / changelog / about
    check_interval  INTEGER DEFAULT 86400,
    last_checked_at TIMESTAMP,
    last_changed_at TIMESTAMP,
    status          TEXT DEFAULT 'active',
    created_at      TIMESTAMP DEFAULT NOW(),
    UNIQUE(competitor_id, url)
);

CREATE TABLE snapshots (
    id          SERIAL PRIMARY KEY,
    page_id     INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    checksum    TEXT NOT NULL,
    raw_html    TEXT,
    captured_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_snapshots_page_time ON snapshots(page_id, captured_at DESC);

CREATE TABLE analyses (
    id              SERIAL PRIMARY KEY,
    page_id         INTEGER NOT NULL REFERENCES pages(id),
    old_snapshot_id INTEGER REFERENCES snapshots(id),
    new_snapshot_id INTEGER REFERENCES snapshots(id),
    change_type     TEXT,
    severity        TEXT DEFAULT 'low',
    summary         TEXT,
    strategic_insight TEXT,
    action_items    JSONB,
    raw_diff        TEXT,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE TABLE notifications (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    analysis_id INTEGER REFERENCES analyses(id),
    channel     TEXT NOT NULL,
    status      TEXT DEFAULT 'pending',
    sent_at     TIMESTAMP
);
```

## Step 4.4：API 路由

```go
// internal/watchbot/api/server.go
func (s *Server) setupRoutes() {
    r := http.NewServeMux()

    // 公开接口
    r.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
    r.HandleFunc("POST /api/v1/auth/login", s.handleLogin)

    // 需要认证的接口
    auth := s.authMiddleware

    r.HandleFunc("GET /api/v1/auth/profile", auth(s.handleProfile))

    r.HandleFunc("POST /api/v1/competitors", auth(s.handleAddCompetitor))
    r.HandleFunc("GET /api/v1/competitors", auth(s.handleListCompetitors))
    r.HandleFunc("GET /api/v1/competitors/{id}", auth(s.handleGetCompetitor))
    r.HandleFunc("DELETE /api/v1/competitors/{id}", auth(s.handleDeleteCompetitor))
    r.HandleFunc("GET /api/v1/competitors/{id}/pages", auth(s.handleListPages))

    r.HandleFunc("GET /api/v1/reports", auth(s.handleListReports))
    r.HandleFunc("GET /api/v1/reports/{id}", auth(s.handleGetReport))
    r.HandleFunc("GET /api/v1/timeline", auth(s.handleTimeline))

    r.HandleFunc("POST /api/v1/billing/checkout", auth(s.handleCheckout))
    r.HandleFunc("POST /api/v1/billing/portal", auth(s.handlePortal))

    // Stripe Webhook（公开，用签名验证）
    r.HandleFunc("POST /api/v1/webhook/stripe", s.handleStripeWebhook)

    // 健康检查
    r.HandleFunc("GET /health", s.handleHealth)

    s.handler = r
}
```

## Step 4.5：页面自动发现

```go
// internal/watchbot/pipeline/discoverer.go
// 添加竞品时，自动发现其关键页面

var commonPaths = map[string]string{
    "/pricing":    "pricing",
    "/price":      "pricing",
    "/plans":      "pricing",
    "/features":   "features",
    "/product":    "features",
    "/changelog":  "changelog",
    "/updates":    "changelog",
    "/blog":       "blog",
    "/about":      "about",
}

func (d *Discoverer) Discover(ctx context.Context, domain string) ([]PageInfo, error) {
    var pages []PageInfo
    for path, pageType := range commonPaths {
        url := "https://" + domain + path
        resp, err := http.Head(url)
        if err == nil && resp.StatusCode == 200 {
            pages = append(pages, PageInfo{URL: url, Type: pageType})
        }
    }
    return pages, nil
}
```

## Step 4.6：LLM 分析 Prompt

```go
// internal/watchbot/pipeline/prompts.go
const AnalysisPrompt = `你是一位竞争情报分析师。分析以下竞品页面的变化。

竞品名称: {{.Competitor}}
页面类型: {{.PageType}}
旧版本日期: {{.OldDate}}
新版本日期: {{.NewDate}}

以下是页面内容的 diff（- 表示删除，+ 表示新增）:
` + "```diff" + `
{{.Diff}}
` + "```" + `

请分析并输出 JSON：
{
  "change_type": "pricing|feature_add|feature_remove|content|brand",
  "severity": "high|medium|low",
  "summary": "1-2 句话描述变更内容",
  "strategic_insight": "这个变更在竞争策略上意味着什么？",
  "action_items": ["建议采取的行动1", "建议2"]
}

判断标准：
- HIGH: 定价变更、功能大幅调整、核心产品方向变化
- MEDIUM: 新功能上线、文案调整暗示策略变化
- LOW: 纯文案/样式微调`
```

## Step 4.7：JWT 认证

```go
// internal/watchbot/api/middleware.go
// 使用 github.com/golang-jwt/jwt/v5

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := extractBearerToken(r)
        claims, err := validateJWT(token, s.jwtSecret)
        if err != nil {
            writeJSON(w, 401, Error{Message: "unauthorized"})
            return
        }
        ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
        next(w, r.WithContext(ctx))
    }
}
```

### 依赖

```
go get github.com/golang-jwt/jwt/v5
go get github.com/stripe/stripe-go/v82
go get golang.org/x/crypto/bcrypt
```

## Step 4.8：部署

```dockerfile
# deploy/docker/Dockerfile.watchbot
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /watchbot ./cmd/watchbot

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /watchbot /usr/local/bin/watchbot
EXPOSE 8080
ENTRYPOINT ["watchbot"]
CMD ["api"]
```

```yaml
# deploy/docker/docker-compose.watchbot.yml
services:
  watchbot-api:
    build:
      context: ../..
      dockerfile: deploy/docker/Dockerfile.watchbot
    ports: ["8080:8080"]
    command: ["watchbot", "api"]
    environment:
      - DATABASE_URL=postgres://watchbot:pass@db:5432/watchbot
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
      - STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
      - STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET}
    depends_on: [db]

  watchbot-worker:
    build:
      context: ../..
      dockerfile: deploy/docker/Dockerfile.watchbot
    command: ["watchbot", "worker"]
    environment:
      - DATABASE_URL=postgres://watchbot:pass@db:5432/watchbot
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    depends_on: [db]

  db:
    image: postgres:17-alpine
    volumes: [pgdata:/var/lib/postgresql/data]
    environment:
      POSTGRES_DB: watchbot
      POSTGRES_USER: watchbot
      POSTGRES_PASSWORD: pass

  web:
    image: nginx:alpine
    ports: ["3000:80"]
    volumes:
      - ../../web/watchbot-dashboard:/usr/share/nginx/html:ro

volumes:
  pgdata:
```

## Step 4.9：前端仪表盘（MVP — 纯 HTML+JS）

```
使用纯 HTML + Vanilla JS + CSS 实现 MVP 版仪表盘。

页面清单：
1. login.html      — 登录/注册
2. dashboard.html  — 主面板（竞品列表 + 最新变更）
3. competitor.html  — 单个竞品详情（页面列表 + 变更历史）
4. report.html      — 单个分析报告详情
5. settings.html    — 账号设置 + 订阅管理

设计要求：
- 深色主题，现代 SaaS 风格
- 响应式布局（桌面 + 移动端）
- 使用 CSS 变量管理主题色
- 所有 API 调用封装在 api.js 中
- JWT Token 存储在 localStorage
```

## Step 4.10：开发顺序 & 验证

| 序号 | 任务 | 验证标准 | 预计时间 |
|------|------|---------|---------|
| 1 | 数据模型 + Store 接口 + SQLite 实现 | CRUD 测试通过 | 3h |
| 2 | JWT 认证 + 用户注册/登录 API | curl 测试通过 | 2h |
| 3 | 竞品 CRUD API | curl 增删改查正常 | 2h |
| 4 | 页面自动发现 | 添加竞品后自动发现 ≥2 个页面 | 1h |
| 5 | Pipeline: 抓取 + diff | 检测到页面变更 | 3h |
| 6 | Pipeline: LLM 分析 | 输出结构化分析 JSON | 2h |
| 7 | Pipeline: 通知分发 | 邮件发送成功 | 1h |
| 8 | Worker: Cron 调度 | 定时自动执行 pipeline | 1h |
| 9 | 报告 API + 时间线 | 返回历史分析列表 | 1h |
| 10 | 前端: 登录 + 仪表盘 | 浏览器可登录查看 | 4h |
| 11 | Stripe 集成 | Checkout 跳转成功 | 2h |
| 12 | Docker 构建 + 部署 | `docker-compose up` 跑通 | 2h |
| **总计** | | | **约 24h（4-5 天）** |

---

## 所有 Phase 总预计时间

| Phase | 内容 | 时间 |
|-------|------|------|
| Phase 0 | 项目初始化 + 共享基础设施 | 12-16h（2-3天） |
| Phase 1 | NewsBot | 14h（2-3天） |
| Phase 2 | DevKit CLI | 14h（2-3天） |
| Phase 3 | MCP Template | 16h（2-3天） |
| Phase 4 | WatchBot | 24h（4-5天） |
| **总计** | | **80-84h（约 14-17天）** |

> [!TIP]
> 建议按 Phase 0 → 1 → 2 → 3 → 4 顺序开发。
> Phase 0 是基础，必须先完成。
> Phase 1 和 2 可并行（NewsBot 和 DevKit 依赖不同的共享包）。
> Phase 3 相对独立。
> Phase 4 依赖最重，放最后。
