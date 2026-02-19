# 产品 1：AI 热点日报 Bot（NewsBot）详细设计

## 1. 产品定义

### 1.1 产品愿景
>
> "每天 3 分钟，掌握 AI 行业全部重要动态。"

### 1.2 目标用户画像

| 画像 | 描述 | 痛点 |
|------|------|------|
| **AI 开发者** | 使用 LLM API 构建产品的工程师 | 不想错过 API 变更导致线上事故 |
| **技术 Lead/CTO** | 小团队技术决策者 | 需要每天快速了解行业动态辅助决策 |
| **独立开发者** | IndieHacker、Solopreneur | 时间有限，需要高效的信息获取方式 |
| **AI 投资人/分析师** | 关注 AI 赛道的投资人 | 需要第一时间知道重大技术突破 |

### 1.3 核心功能

| 功能 | 优先级 | MVP | V2 | 状态 |
|------|--------|-----|-----|------|
| 多源数据采集 | P0 | ✅ 5 个源 | 20+ 个源 | ✅ 已实现 8 个源 |
| LLM 智能摘要 | P0 | ✅ 每日 1 篇 | 多篇 + 分类 | ✅ MiniMax M2.5 |
| Telegram 推送 | P0 | ✅ | ✅ | 🔜 待接入 |
| 邮件推送 | P1 | ❌ | ✅ | ✅ Gmail SMTP |
| **多语言日报** | P1 | ❌ | ✅ | ✅ 6 种语言 |
| **订阅者管理** | P1 | ❌ | ✅ | ✅ CLI 管理 |
| 历史搜索 | P1 | ❌ | ✅ | 🔜 |
| 自定义关键词订阅 | P2 | ❌ | ✅ | 🔜 |
| 每周深度分析报告 | P2 | ❌ | ✅（付费） | 🔜 |
| Slack/Discord 集成 | P2 | ❌ | ✅（付费） | 🔜 |

### 1.4 已实现数据源

| # | 来源 | 类型 | 说明 |
|---|------|------|------|
| 1 | Hacker News | API | Top 30 stories |
| 2 | TechCrunch AI | RSS | AI 频道 |
| 3 | MIT Tech Review | RSS | AI 频道 |
| 4 | The Verge AI | RSS/Atom | AI 频道 |
| 5 | Ars Technica | RSS | Tech Lab |
| 6 | VentureBeat AI | RSS | AI 频道 |
| 7 | OpenAI Blog | RSS | 官方博客 |
| 8 | Google AI Blog | RSS | 官方博客 |

### 1.5 多语言支持

| 语言代码 | 语言 | 邮件标题 |
|----------|------|----------|
| zh | 中文 | AI 热点日报 |
| en | English | AI Daily Digest |
| ja | 日本語 | AI デイリーダイジェスト |
| ko | 한국어 | AI 데일리 다이제스트 |
| de | Deutsch | AI Täglicher Überblick |
| es | Español | AI Resumen Diario |

---

## 2. 软件架构

### 2.1 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        调度层 (Scheduler)                     │
│                    Cron: 每天 07:00 / 19:00 UTC+8            │
└────────────┬──────────────────────────────────┬──────────────┘
             │                                  │
    ┌────────▼────────┐              ┌──────────▼──────────┐
    │   采集层 (Scraper) │              │   分析层 (Analyzer)  │
    │                   │              │                     │
    │ ┌───────────────┐ │    结果       │ ┌─────────────────┐ │
    │ │ GitHub        │ │────────────▶ │ │ 去重 + 过滤     │ │
    │ │ OpenAI Blog   │ │              │ │ 重要性评分       │ │
    │ │ Anthropic     │ │              │ │ LLM 摘要生成    │ │
    │ │ Google AI     │ │              │ │ 分类标签        │ │
    │ │ Hacker News   │ │              │ └─────────────────┘ │
    │ └───────────────┘ │              └──────────┬──────────┘
    └───────────────────┘                         │
                                       ┌──────────▼──────────┐
                                       │   分发层 (Publisher)  │
                                       │                     │
                                       │ Telegram / Email    │
                                       │ Twitter / RSS       │
                                       │ Webhook             │
                                       └──────────┬──────────┘
                                                  │
                                       ┌──────────▼──────────┐
                                       │   存储层 (Storage)    │
                                       │                     │
                                       │ SQLite：文章表       │
                                       │ 推送记录表           │
                                       │ 用户订阅表           │
                                       └─────────────────────┘
```

### 2.2 核心模块设计

#### 采集层 — Scraper Interface

```go
// pkg/scraper/scraper.go
type Article struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    URL       string    `json:"url"`
    Content   string    `json:"content"`
    Source    string    `json:"source"`    // "github" / "openai" / "hackernews"
    Category string    `json:"category"`  // "model_release" / "api_change" / "paper"
    PubDate  time.Time `json:"pub_date"`
    RawHTML  string    `json:"-"`
}

type Scraper interface {
    Name() string
    Fetch(ctx context.Context) ([]Article, error)
}

// 已实现的数据源
type HackerNewsSource struct { ... }  // API
type RSSSource struct { ... }         // 通用 RSS/Atom
```

#### 分析层 — LLM Summarizer

```go
// internal/newsbot/analyzer/analyzer.go
type DailyDigest struct {
    Date       string          `json:"date"`
    Summary    string          `json:"summary"`     // 今日概览（每条新闻独立成句）
    Articles   []DigestArticle `json:"articles"`
    TotalCount int             `json:"total_count"`
}

type DigestArticle struct {
    Title      string   `json:"title"`
    Summary    string   `json:"summary"`     // 2-3 句话摘要
    Impact     string   `json:"impact"`      // 对开发者的影响
    Severity   string   `json:"severity"`    // high / medium / low
    Tags       []string `json:"tags"`
    SourceURL  string   `json:"source_url"`
}
```

**LLM Prompt 模板**：

```
你是一位资深 AI 行业分析师，负责为开发者编写每日简报。

今天采集到以下 {{count}} 条信息：
{{#each articles}}
---
标题: {{title}}
来源: {{source}}
内容片段: {{content_snippet}}
发布时间: {{pub_date}}
{{/each}}

请完成以下任务：
1. 从中筛选出最重要的 5-8 条（优先级：API 变更 > 模型发布 > 重大融资 > 开源项目 > 论文）
2. 为每条生成 2-3 句摘要，重点说明"对开发者意味着什么"
3. 标注影响等级：🔴 高 / 🟡 中 / 🟢 低
4. 生成一句话今日总结

输出 JSON 格式（DailyDigest 结构）
```

#### 分发层 — Publisher + i18n

```go
// internal/newsbot/publisher/publisher.go
pub := publisher.NewPublisher(dispatcher)
pub.PublishToEmail(ctx, digest, lang, email)

// internal/newsbot/i18n/translator.go
translator := i18n.NewTranslator(llmClient)
digests := translator.TranslateAll(ctx, digest, []Language{LangZH, LangEN, LangJA})
// 并行翻译：分析一次(zh) + 翻译 N 次，节省 80% LLM 成本
```

**邮件模板特性**：

- 暗色主题 Newsletter 设计（渐变头部、交替背景）
- 重要性徽章（重要/关注/了解）
- 彩色标签 + "阅读原文 →" 链接
- 概览按句分行（支持中英文句号拆分）
- 所有 UI 文本完整 i18n（6 种语言）

### 2.3 数据库设计

```sql
-- 文章表
CREATE TABLE articles (
    id          TEXT PRIMARY KEY,           -- SHA256(url)
    title       TEXT NOT NULL,
    url         TEXT NOT NULL UNIQUE,
    content     TEXT,
    summary     TEXT,                       -- LLM 生成的摘要
    source      TEXT NOT NULL,              -- github / openai / hackernews
    category    TEXT,                       -- model_release / api_change
    severity    TEXT DEFAULT 'low',         -- high / medium / low
    tags        TEXT,                       -- JSON array
    pub_date    DATETIME,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 日报表
CREATE TABLE digests (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    date        TEXT NOT NULL UNIQUE,       -- 2026-02-24
    headline    TEXT,
    content     TEXT,                       -- 完整日报 JSON
    article_ids TEXT,                       -- 关联文章 ID 列表
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 订阅者表
CREATE TABLE subscribers (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    platform    TEXT NOT NULL,              -- telegram / email
    identifier  TEXT NOT NULL,              -- chat_id / email
    plan        TEXT DEFAULT 'free',        -- free / pro
    keywords    TEXT,                       -- 自定义关键词 JSON
    active      BOOLEAN DEFAULT TRUE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(platform, identifier)
);

-- 推送记录表
CREATE TABLE push_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    digest_id   INTEGER REFERENCES digests(id),
    subscriber_id INTEGER REFERENCES subscribers(id),
    status      TEXT,                       -- sent / failed
    sent_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 3. 部署方案

### 3.1 MVP 部署（$5/月）

```yaml
# docker-compose.yml
version: '3.8'
services:
  newsbot:
    build: .
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TELEGRAM_CHANNEL_ID=${TELEGRAM_CHANNEL_ID}
    volumes:
      - ./data:/app/data              # SQLite 数据持久化
    restart: unless-stopped

# Cron 通过系统 crontab 或 Go 内置 ticker 实现
# 0 7,19 * * * docker exec newsbot /app/newsbot run
```

**部署步骤**：

1. 租一台 Hetzner/Contabo VPS（€4.5/月，2C4G）
2. 安装 Docker + docker-compose
3. `git clone` + 配置环境变量
4. `docker-compose up -d`
5. 系统 crontab 定时执行

### 3.2 生产部署（扩展时）

| 组件 | 方案 | 成本 |
|------|------|------|
| 计算 | Fly.io / Railway | $5-10/月 |
| 数据库 | Turso (SQLite 云端) 或 Supabase | 免费 tier |
| 定时任务 | GitHub Actions scheduled | 免费 |
| 域名 | Cloudflare | $10/年 |

---

## 4. 商业化设计

### 4.1 免费 → 付费转化漏斗

```
Telegram 公开频道              免费订阅者
  ↓ 每天推送 3-5 条摘要         (获客)
  ↓
  ↓ 底部 CTA："完整版 8 条 + 深度分析 → 升级 Pro"
  ↓
Pro 邮件列表                   付费用户 $9/月
  ↓ 每天完整日报 + 每周深度报告
  ↓
  ↓ 邮件签名推荐 DevKit / WatchBot
  ↓
其他产品转化                    交叉销售
```

### 4.2 定价

| 功能 | Free | Pro $9/月 |
|------|------|-----------|
| 每日摘要 | 3-5 条 | 全部 8-12 条 |
| 数据源 | 5 个 | 20+ 个 |
| 分发渠道 | Telegram | Telegram + Email + Slack + Webhook |
| 历史搜索 | ❌ | ✅ 90 天 |
| 每周深度报告 | ❌ | ✅ PDF |
| 自定义关键词 | ❌ | ✅ 3 个 |
| API 访问 | ❌ | ✅ |

### 4.3 运营成本

| 项目 | 月成本 |
|------|--------|
| VPS | $5 |
| LLM API (MiniMax M2.5) | ~$1（每天分析 ~7K tokens + 翻译 ~5K×5 tokens） |
| 域名 | ~$1 |
| **总计** | **~$7/月** |

> MiniMax M2.5 相比 GPT-4o-mini 成本降低约 60%。

> 1 个付费用户即可覆盖成本。

---

## 5. 推广策略

### 5.1 冷启动（第 1 周）

| 动作 | 渠道 | 预期效果 |
|------|------|---------|
| 发帖 "I built an AI news bot" | r/ChatGPT, r/OpenAI | 50-100 订阅 |
| Twitter thread 展示日报样例 | Twitter/X | 20-50 订阅 |
| IndieHackers 分享构建过程 | IndieHackers | 10-30 订阅 |
| 知乎/即刻发帖 | 中文社区 | 50-100 订阅 |

### 5.2 持续增长

- **内容飞轮**：日报本身就是内容，订阅者会转发 → 自然增长
- **SEO**：将每日日报发布为网页版，积累搜索流量
- **合作**：与 AI Newsletter 互推（如 The Neuron、Ben's Bites）
