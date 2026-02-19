# 产品 4：竞品监控 Bot（WatchBot）详细设计

## 1. 产品定义

### 1.1 产品愿景
>
> "你的竞品改了价格、砍了功能、换了策略——我比你先知道。"

### 1.2 目标用户

| 画像 | 描述 | 付费意愿 | 典型场景 |
|------|------|---------|---------|
| **SaaS 创始人** | 需要时刻关注竞品动态 | 🟢 高 | 竞品降价，需快速响应 |
| **产品经理** | 跟踪竞品功能更新 | 🟢 高 | 竞品上新功能，需评估跟进 |
| **市场/增长负责人** | 监控竞品营销策略变化 | 🟡 中 | 竞品换了 CTA/定价结构 |
| **投资人/分析师** | 跟踪行业动态 | 🟡 中 | 判断赛道竞争态势 |

### 1.3 核心功能

| 功能 | 优先级 | MVP | V2 |
|------|--------|-----|-----|
| 添加竞品域名 | P0 | ✅ | ✅ |
| 自动发现关键页面 | P0 | ✅ (/pricing, /features) | + blog, changelog |
| 定时抓取 + 存储快照 | P0 | ✅ 每天 1 次 | 每天 4 次 |
| HTML diff 引擎 | P0 | ✅ 文本 diff | + 视觉截图 diff |
| LLM 智能分析 | P0 | ✅ | ✅ + 更精准 |
| 邮件通知 | P0 | ✅ | ✅ |
| Web 仪表盘 | P1 | ✅ 基础版 | ✅ 完整版 |
| Slack/Webhook 通知 | P1 | ❌ | ✅ |
| 变更历史时间线 | P1 | ❌ | ✅ |
| 竞品周报 PDF | P2 | ❌ | ✅ |
| API 访问 | P2 | ❌ | ✅ |
| 截图对比 | P2 | ❌ | ✅ (Playwright) |

---

## 2. 软件架构

### 2.1 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│              Web Frontend (Next.js / HTML+JS)                    │
│                                                                  │
│  ┌─────────┐ ┌───────────┐ ┌──────────┐ ┌───────────────────┐  │
│  │ 登录注册 │ │ 竞品管理   │ │ 报告查看 │ │ 账户/订阅管理     │  │
│  └─────────┘ └───────────┘ └──────────┘ └───────────────────┘  │
└──────────────────────────┬──────────────────────────────────────┘
                           │ REST API (JSON)
┌──────────────────────────▼──────────────────────────────────────┐
│                    Go Backend API Server                         │
│                                                                  │
│  ┌─────────────────┐  ┌───────────────┐  ┌──────────────────┐  │
│  │ User API         │  │ Competitor API │  │ Report API        │  │
│  │ POST /register   │  │ POST /comp    │  │ GET /reports      │  │
│  │ POST /login      │  │ GET  /comp    │  │ GET /reports/:id  │  │
│  │ GET  /profile    │  │ DELETE /comp  │  │ GET /timeline     │  │
│  └─────────────────┘  └───────────────┘  └──────────────────┘  │
│                                                                  │
│  ┌─────────────────┐  ┌───────────────────────────────────────┐ │
│  │ Stripe Webhook   │  │ Auth Middleware (JWT)                 │ │
│  │ POST /webhook    │  │                                       │ │
│  └─────────────────┘  └───────────────────────────────────────┘ │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                 Background Worker (Cron)                          │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Pipeline Runner                         │   │
│  │                                                            │   │
│  │  1. Fetch    →  2. Diff     →  3. Analyze   →  4. Notify  │   │
│  │  抓取页面       计算差异        LLM 分析         推送通知   │   │
│  └──────────────────────────────────────────────────────────┘   │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                     PostgreSQL / SQLite                           │
│                                                                  │
│  users │ competitors │ pages │ snapshots │ analyses │ plans      │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 数据库设计

```sql
-- 用户表
CREATE TABLE users (
    id            SERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    plan          TEXT DEFAULT 'free',        -- free / growth / pro
    stripe_id     TEXT,                       -- Stripe Customer ID
    created_at    TIMESTAMP DEFAULT NOW()
);

-- 竞品表
CREATE TABLE competitors (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER REFERENCES users(id),
    name        TEXT NOT NULL,               -- "Visualping"
    domain      TEXT NOT NULL,               -- "visualping.io"
    status      TEXT DEFAULT 'active',       -- active / paused
    created_at  TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, domain)
);

-- 监控页面表
CREATE TABLE pages (
    id            SERIAL PRIMARY KEY,
    competitor_id INTEGER REFERENCES competitors(id),
    url           TEXT NOT NULL,
    page_type     TEXT NOT NULL,              -- pricing / features / blog / changelog
    check_interval INTEGER DEFAULT 86400,     -- 检查间隔（秒）
    last_checked  TIMESTAMP,
    status        TEXT DEFAULT 'active',
    UNIQUE(competitor_id, url)
);

-- 快照表
CREATE TABLE snapshots (
    id          SERIAL PRIMARY KEY,
    page_id     INTEGER REFERENCES pages(id),
    content     TEXT NOT NULL,                -- 清洗后的文本内容
    raw_html    TEXT,                         -- 原始 HTML（压缩存储）
    checksum    TEXT NOT NULL,                -- SHA256，用于快速判断是否变更
    captured_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_snapshots_page_time ON snapshots(page_id, captured_at DESC);

-- 分析报告表
CREATE TABLE analyses (
    id              SERIAL PRIMARY KEY,
    page_id         INTEGER REFERENCES pages(id),
    old_snapshot_id INTEGER REFERENCES snapshots(id),
    new_snapshot_id INTEGER REFERENCES snapshots(id),
    change_type     TEXT,                     -- pricing / feature / content / brand
    severity        TEXT,                     -- high / medium / low
    summary         TEXT,                     -- 变更摘要
    strategic_insight TEXT,                   -- 竞争含义
    action_items    TEXT,                     -- JSON: 行动建议数组
    raw_diff        TEXT,                     -- 原始 diff
    llm_response    TEXT,                     -- LLM 完整响应（调试用）
    created_at      TIMESTAMP DEFAULT NOW()
);

-- 通知记录表
CREATE TABLE notifications (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER REFERENCES users(id),
    analysis_id INTEGER REFERENCES analyses(id),
    channel     TEXT,                         -- email / slack / webhook
    status      TEXT,                         -- sent / failed
    sent_at     TIMESTAMP DEFAULT NOW()
);
```

### 2.3 核心流程 — 监控 Pipeline

```go
// internal/watchbot/pipeline.go
type Pipeline struct {
    fetcher   *Fetcher
    differ    *Differ
    analyzer  *Analyzer
    notifier  *Notifier
    store     *Store
}

func (p *Pipeline) Run(ctx context.Context) error {
    // 1. 获取所有待检查的页面
    pages, err := p.store.GetPagesForCheck()

    for _, page := range pages {
        // 2. 抓取当前版本
        content, err := p.fetcher.Fetch(ctx, page.URL)

        // 3. 与上一版本对比
        lastSnapshot, _ := p.store.GetLatestSnapshot(page.ID)
        newChecksum := sha256(content)

        if lastSnapshot != nil && lastSnapshot.Checksum == newChecksum {
            continue // 无变更，跳过
        }

        // 4. 保存新快照
        newSnapshot := p.store.SaveSnapshot(page.ID, content, newChecksum)

        if lastSnapshot == nil {
            continue // 首次抓取，无需分析
        }

        // 5. 生成 diff
        diff := p.differ.Diff(lastSnapshot.Content, content)

        // 6. LLM 分析
        analysis, err := p.analyzer.Analyze(ctx, AnalysisInput{
            Competitor: page.CompetitorName,
            PageType:   page.PageType,
            OldDate:    lastSnapshot.CapturedAt,
            NewDate:    time.Now(),
            Diff:       diff,
        })

        // 7. 保存分析结果
        p.store.SaveAnalysis(page.ID, lastSnapshot.ID, newSnapshot.ID, analysis)

        // 8. 通知用户（仅 medium/high severity）
        if analysis.Severity != "low" {
            p.notifier.Notify(page.UserID, analysis)
        }
    }
    return nil
}
```

### 2.4 页面爬虫 — 智能内容提取

```go
// internal/watchbot/fetcher.go
type Fetcher struct {
    client   *http.Client
    // 未来可扩展：Playwright 支持 JS 渲染
}

func (f *Fetcher) Fetch(ctx context.Context, url string) (string, error) {
    resp, err := f.client.Get(url)
    // ...
    body, _ := io.ReadAll(resp.Body)

    // 使用 goquery 提取核心内容（去掉导航、页脚、广告）
    doc, _ := goquery.NewDocumentFromReader(bytes.NewReader(body))

    // 移除无关元素
    doc.Find("nav, footer, header, script, style, .cookie-banner").Remove()

    // 提取文本（保留结构）
    content := extractStructuredText(doc)
    return content, nil
}

// extractStructuredText 保留标题层级和列表结构
func extractStructuredText(doc *goquery.Document) string {
    var buf strings.Builder
    doc.Find("main, article, .content, #content, body").First().Each(func(i int, s *goquery.Selection) {
        s.Find("h1, h2, h3, h4, p, li, td, th, span.price").Each(func(j int, el *goquery.Selection) {
            tag := goquery.NodeName(el)
            text := strings.TrimSpace(el.Text())
            if text == "" { return }
            switch tag {
            case "h1": buf.WriteString("# " + text + "\n")
            case "h2": buf.WriteString("## " + text + "\n")
            case "h3": buf.WriteString("### " + text + "\n")
            case "li": buf.WriteString("- " + text + "\n")
            default:   buf.WriteString(text + "\n")
            }
        })
    })
    return buf.String()
}
```

### 2.5 Diff 引擎

```go
// pkg/differ/differ.go
type DiffResult struct {
    HasChanges bool     `json:"has_changes"`
    Added      []string `json:"added"`       // 新增的行
    Removed    []string `json:"removed"`     // 删除的行
    Modified   []string `json:"modified"`    // 修改的行
    Unified    string   `json:"unified"`     // unified diff 格式
    Summary    string   `json:"summary"`     // "3 additions, 2 deletions"
}

func Diff(oldContent, newContent string) DiffResult {
    // 使用 go-diff 库或自实现
    // 输出 unified diff 格式供 LLM 分析
}
```

---

## 3. API 设计

### 3.1 REST API

```
认证: Bearer JWT Token

用户
  POST   /api/v1/auth/register        注册
  POST   /api/v1/auth/login            登录
  GET    /api/v1/auth/profile          获取用户信息

竞品管理
  POST   /api/v1/competitors           添加竞品
  GET    /api/v1/competitors           列出竞品
  GET    /api/v1/competitors/:id       竞品详情
  DELETE /api/v1/competitors/:id       删除竞品
  GET    /api/v1/competitors/:id/pages 竞品的监控页面

报告
  GET    /api/v1/reports                所有分析报告
  GET    /api/v1/reports/:id            报告详情
  GET    /api/v1/reports/timeline       变更时间线

订阅
  POST   /api/v1/billing/checkout      创建 Stripe Checkout
  POST   /api/v1/billing/portal        跳转 Stripe Portal
  POST   /api/v1/webhook/stripe        Stripe Webhook
```

### 3.2 数据模型

```go
// 添加竞品
// POST /api/v1/competitors
type AddCompetitorRequest struct {
    Name   string `json:"name" validate:"required"`
    Domain string `json:"domain" validate:"required,url"`
}

type AddCompetitorResponse struct {
    ID        int           `json:"id"`
    Name      string        `json:"name"`
    Domain    string        `json:"domain"`
    Pages     []PageInfo    `json:"pages"`      // 自动发现的页面
    CreatedAt time.Time     `json:"created_at"`
}

// 分析报告
type AnalysisReport struct {
    ID               int       `json:"id"`
    Competitor       string    `json:"competitor"`
    PageType         string    `json:"page_type"`
    ChangeType       string    `json:"change_type"`
    Severity         string    `json:"severity"`
    Summary          string    `json:"summary"`
    StrategicInsight string    `json:"strategic_insight"`
    ActionItems      []string  `json:"action_items"`
    DiffPreview      string    `json:"diff_preview"`
    DetectedAt       time.Time `json:"detected_at"`
}
```

---

## 4. 部署方案

### 4.1 MVP 部署

```yaml
# docker-compose.yml
version: '3.8'
services:
  watchbot-api:
    build:
      context: .
      dockerfile: deploy/docker/Dockerfile.watchbot
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://user:pass@db:5432/watchbot
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
      - STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET}
      - JWT_SECRET=${JWT_SECRET}
      - SMTP_HOST=${SMTP_HOST}
    depends_on:
      - db

  watchbot-worker:
    build:
      context: .
      dockerfile: deploy/docker/Dockerfile.watchbot
    command: ["watchbot", "worker"]    # 运行后台 worker
    environment:
      - DATABASE_URL=postgres://user:pass@db:5432/watchbot
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    depends_on:
      - db

  db:
    image: postgres:16-alpine
    volumes:
      - pgdata:/var/lib/postgresql/data
    environment:
      - POSTGRES_DB=watchbot
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass

  web:
    build:
      context: ./web/watchbot-dashboard
    ports:
      - "3000:3000"

volumes:
  pgdata:
```

### 4.2 成本估算

| 项目 | MVP 阶段 | 100 用户 | 1000 用户 |
|------|---------|---------|----------|
| VPS | $10/月 | $20/月 | $50/月 |
| PostgreSQL | 内含 | $15/月 (Supabase) | $50/月 |
| LLM API | $5/月 | $30/月 | $200/月 |
| Stripe 手续费 | 2.9%+30¢ | 2.9%+30¢ | 2.9%+30¢ |
| 域名+CDN | $2/月 | $2/月 | $10/月 |
| **总计** | **~$17/月** | **~$67/月** | **~$310/月** |

---

## 5. 商业化

### 5.1 定价

| 功能 | Free | Growth $19/月 | Pro $49/月 |
|------|------|:-------------:|:----------:|
| 竞品数量 | 1 | 5 | 20 |
| 监控页面类型 | 仅 pricing | pricing+features+blog | 全部 |
| 检查频率 | 每周 1 次 | 每天 1 次 | 每天 2 次 |
| 通知渠道 | 邮件 | 邮件+Slack | 邮件+Slack+Webhook |
| 历史记录 | 无 | 30 天 | 1 年 |
| LLM 智能分析 | ❌ | ✅ | ✅ |
| API 访问 | ❌ | ❌ | ✅ |
| 竞品周报 PDF | ❌ | ❌ | ✅ |

### 5.2 用户获取策略

| 阶段 | 渠道 | 策略 |
|------|------|------|
| 冷启动 | Twitter/X | 发布"我监控了 XX 竞品 30 天后发现了什么" thread |
| 冷启动 | Reddit | r/SaaS, r/Entrepreneur 发帖 |
| 增长 | SEO | "competitor monitoring tools"、"track competitor pricing" |
| 增长 | 内容 | 每月发布"SaaS 定价趋势报告"（用产品数据） |
| 留存 | 产品内 | 每周自动发送竞品周报邮件 |

### 5.3 关键指标

| 指标 | 1 个月目标 | 3 个月目标 | 6 个月目标 |
|------|-----------|-----------|-----------|
| 注册用户 | 50 | 300 | 1000 |
| 付费用户 | 5 | 30 | 80 |
| MRR | $95 | $750 | $2,800 |
| Churn Rate | - | <10% | <8% |
