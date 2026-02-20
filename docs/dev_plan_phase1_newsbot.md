# 开发计划 — Phase 1：AI 热点日报 Bot（NewsBot）

> 前置依赖：Phase 0 全部完成
>
> 项目路径：`devkit-suite/cmd/newsbot/` + `devkit-suite/internal/newsbot/`

---

## Step 1.1：NewsBot 目录结构

```
cmd/newsbot/
└── main.go                          # 入口：解析命令行参数，加载配置，执行任务

internal/newsbot/
├── app.go                           # 应用主逻辑：编排采集→分析→分发流程
├── sources/                         # 各数据源爬虫实现
│   ├── source.go                    # Source 接口定义
│   ├── github_trending.go           # GitHub Trending 爬虫
│   ├── openai_blog.go               # OpenAI 博客爬虫
│   ├── anthropic_news.go            # Anthropic 新闻爬虫
│   ├── google_ai_blog.go            # Google AI 博客爬虫
│   ├── hackernews.go                # Hacker News AI 分类爬虫
│   └── sources_test.go
├── analyzer/                        # LLM 分析层
│   ├── analyzer.go                  # 分析器：去重、评分、摘要
│   ├── prompts.go                   # Prompt 模板常量
│   ├── models.go                    # DailyDigest / DigestArticle 数据模型
│   └── analyzer_test.go
├── publisher/                       # 分发层（调用 pkg/notify）
│   ├── publisher.go                 # 分发编排器
│   ├── formatter.go                 # 各渠道格式化（Telegram Markdown / Email HTML）
│   └── publisher_test.go
├── store/                           # 数据层（调用 pkg/storage）
│   ├── store.go                     # NewsBot 专用存储接口
│   ├── schema.sql                   # 建表 SQL（articles / digests / subscribers / push_logs）
│   └── store_test.go
└── scheduler/                       # 定时任务
    └── scheduler.go                 # Go 内置 ticker 或 cron 表达式
```

## Step 1.2：入口程序

```go
// cmd/newsbot/main.go
package main

// 支持的子命令：
//   newsbot run          — 执行一次：采集→分析→推送
//   newsbot serve        — 启动常驻服务（内置 scheduler）
//   newsbot sources      — 列出所有数据源及其状态
//   newsbot history      — 查看历史日报

func main() {
    rootCmd := &cobra.Command{Use: "newsbot"}
    rootCmd.AddCommand(runCmd, serveCmd, sourcesCmd, historyCmd)
    rootCmd.Execute()
}
```

## Step 1.3：数据源接口

```go
// internal/newsbot/sources/source.go
type Article struct {
    ID        string    `json:"id" db:"id"`          // SHA256(URL)
    Title     string    `json:"title" db:"title"`
    URL       string    `json:"url" db:"url"`
    Content   string    `json:"content" db:"content"` // 正文片段（前 500 字）
    Source    string    `json:"source" db:"source"`   // "github" / "openai" / ...
    Category string    `json:"category" db:"category"`
    PubDate  time.Time `json:"pub_date" db:"pub_date"`
}

type Source interface {
    Name() string
    Fetch(ctx context.Context) ([]Article, error)
}
```

### 各数据源实现要点

| 数据源 | URL | 解析方式 | 频率 |
|--------|-----|---------|------|
| GitHub Trending | `https://github.com/trending?since=daily` | goquery 解析 repo 列表 | 每天 1 次 |
| OpenAI Blog | `https://openai.com/blog` | goquery 解析文章列表 | 每天 2 次 |
| Anthropic News | `https://www.anthropic.com/news` | goquery 解析 | 每天 2 次 |
| Google AI Blog | `https://blog.google/technology/ai/` | goquery 解析 | 每天 1 次 |
| Hacker News | `https://hacker-news.firebaseio.com/v0/topstories.json` | JSON API | 每天 2 次 |

## Step 1.4：LLM 分析 Prompt

```go
// internal/newsbot/analyzer/prompts.go
const DailyDigestPrompt = `你是一位资深 AI 行业分析师。请从以下 {{.Count}} 条信息中，筛选出最重要的 5-8 条，为开发者编写每日简报。

筛选优先级：API 变更/弃用 > 新模型发布 > 重大融资/收购 > 热门开源项目 > 论文

{{range .Articles}}
---
标题: {{.Title}}
来源: {{.Source}}
内容: {{.Content}}
时间: {{.PubDate}}
{{end}}

输出 JSON，结构如下：
{
  "headline": "一句话今日总结",
  "articles": [
    {
      "title": "标题",
      "summary": "2-3句摘要，重点说明对开发者的影响",
      "severity": "high|medium|low",
      "tags": ["tag1", "tag2"],
      "source_url": "原文链接"
    }
  ]
}`
```

## Step 1.5：数据库 Schema

```sql
-- internal/newsbot/store/schema.sql
CREATE TABLE IF NOT EXISTS articles (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    url         TEXT NOT NULL UNIQUE,
    content     TEXT,
    summary     TEXT,
    source      TEXT NOT NULL,
    category    TEXT,
    severity    TEXT DEFAULT 'low',
    tags        TEXT,              -- JSON array
    pub_date    DATETIME,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS digests (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    date        TEXT NOT NULL UNIQUE,
    headline    TEXT,
    content     TEXT,              -- 完整 JSON
    article_ids TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS subscribers (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    platform    TEXT NOT NULL,
    identifier  TEXT NOT NULL,
    plan        TEXT DEFAULT 'free',
    keywords    TEXT,
    active      BOOLEAN DEFAULT TRUE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(platform, identifier)
);

CREATE TABLE IF NOT EXISTS push_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    digest_id   INTEGER REFERENCES digests(id),
    subscriber_id INTEGER REFERENCES subscribers(id),
    status      TEXT,
    sent_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Step 1.6：部署

```dockerfile
# deploy/docker/Dockerfile.newsbot
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /newsbot ./cmd/newsbot

FROM alpine:3.21
RUN apk --no-cache add ca-certificates sqlite-libs
COPY --from=builder /newsbot /usr/local/bin/newsbot
COPY configs/config.example.yaml /etc/newsbot/config.yaml
VOLUME ["/data"]
ENTRYPOINT ["newsbot"]
CMD ["serve"]
```

```yaml
# deploy/docker/docker-compose.newsbot.yml
services:
  newsbot:
    build:
      context: ../..
      dockerfile: deploy/docker/Dockerfile.newsbot
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TELEGRAM_CHANNEL_ID=${TELEGRAM_CHANNEL_ID}
    volumes:
      - newsbot-data:/data
    restart: unless-stopped
volumes:
  newsbot-data:
```

## Step 1.7：开发顺序 & 验证

| 序号 | 任务 | 验证标准 | 预计时间 |
|------|------|---------|---------|
| 1 | 实现 `Source` 接口 + GitHub Trending 爬虫 | 抓到 ≥10 条数据 | 2h |
| 2 | 实现 OpenAI + HackerNews 爬虫 | 各抓到数据 | 2h |
| 3 | 实现 store 层 + schema 建表 | 数据可写入可查询 | 2h |
| 4 | 实现 analyzer + LLM prompt | 输出结构化 JSON | 3h |
| 5 | 实现 formatter + Telegram 推送 | Telegram 收到消息 | 2h |
| 6 | 实现 `newsbot run` 命令串联全流程 | 一条命令完成采集→分析→推送 | 1h |
| 7 | 实现 `newsbot serve` + scheduler | 定时自动运行 | 1h |
| 8 | Docker 构建 + 部署测试 | `docker-compose up` 可用 | 1h |
| **总计** | | | **约 14h（2-3 天）** |
