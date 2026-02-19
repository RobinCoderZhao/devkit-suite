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

| 功能 | 优先级 | MVP | V2 |
|------|--------|-----|-----|
| 多源数据采集 | P0 | ✅ 5 个源 | 20+ 个源 |
| LLM 智能摘要 | P0 | ✅ 每日 1 篇 | 多篇 + 分类 |
| Telegram 推送 | P0 | ✅ | ✅ |
| 邮件推送 | P1 | ❌ | ✅ |
| 历史搜索 | P1 | ❌ | ✅ |
| 自定义关键词订阅 | P2 | ❌ | ✅ |
| 每周深度分析报告 | P2 | ❌ | ✅（付费） |
| Slack/Discord 集成 | P2 | ❌ | ✅（付费） |

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

// 每个数据源实现该接口
type GitHubTrendingScraper struct { ... }
type OpenAIBlogScraper struct { ... }
type HackerNewsScraper struct { ... }
```

#### 分析层 — LLM Summarizer

```go
// internal/newsbot/summarizer.go
type DailyDigest struct {
    Date       string          `json:"date"`
    Headline   string          `json:"headline"`    // 今日一句话总结
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

#### 分发层 — Publisher

```go
// pkg/notify/telegram.go
type TelegramPublisher struct {
    BotToken string
    ChatIDs  []string  // 支持多个频道/群组
}

func (t *TelegramPublisher) Publish(digest DailyDigest) error {
    // 格式化为 Telegram Markdown
    // 发送到所有订阅频道
}

// 消息格式示例：
// 📰 AI 日报 | 2026-02-24
//
// 🔥 今日焦点：OpenAI 发布 GPT-5 Turbo，API 价格下降 40%
//
// 🔴 [高] GPT-5 Turbo 发布
// OpenAI 推出 GPT-5 Turbo，上下文窗口扩展至 256K...
// 👉 对开发者：需要更新 model 参数，注意新的 response_format
//
// 🟡 [中] Claude 4 预览版开放申请
// Anthropic 开放 Claude 4 早期访问...
//
// 📎 完整版：https://your-site.com/digest/2026-02-24
```

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
| LLM API (GPT-4o-mini) | ~$3（每天 1 次摘要，约 5K tokens） |
| 域名 | ~$1 |
| **总计** | **~$9/月** |

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
