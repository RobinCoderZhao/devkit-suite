# WatchBot 用户手册 (V2)

> 竞品监控产品 — 多用户架构，智能 URL 解析，聚合通知

## 快速开始

```bash
# 构建
make build-watchbot

# 添加竞品（直接 URL）
./bin/watchbot add https://platform.openai.com/docs/changelog

# 添加竞品（自然语言，需配置 LLM_API_KEY）
./bin/watchbot add "监控 Gemini API 文档变化"

# 订阅通知
./bin/watchbot subscribe --email=you@example.com --competitors="platform.openai.com"

# 运行检查
./bin/watchbot check

# 守护进程（每 6 小时自动检查）
./bin/watchbot serve
```

## CLI 命令

| 命令 | 说明 | 示例 |
| --- | --- | --- |
| `add <url/text>` | 添加监控目标 | `watchbot add https://stripe.com/pricing` |
| `remove --name=<name>` | 删除竞品 | `watchbot remove --name=OpenAI` |
| `list` | 列出所有竞品及页面 | `watchbot list` |
| `subscribe` | 添加订阅者 | `watchbot subscribe --email=x --competitors=a,b` |
| `unsubscribe` | 取消订阅 | `watchbot unsubscribe --email=x` |
| `subscribers` | 列出订阅者 | `watchbot subscribers` |
| `check` | 运行一次全量检查 | `watchbot check` |
| `serve` | 守护进程（6h 间隔） | `watchbot serve` |
| `version` | 显示版本 | `watchbot version` |

## 智能添加

### 直接 URL

```bash
$ watchbot add https://stripe.com/pricing
🔍 验证 URL: https://stripe.com/pricing
竞品名称 (默认: stripe.com): Stripe
✅ 已添加: Stripe [pricing] https://stripe.com/pricing
```

自动处理：补全 `https://`、去掉末尾斜杠和 `#fragment`、DNS 检查、HTTP 状态检查（软验证）。

### 自然语言

需配置 `LLM_API_KEY`。三层解析：LLM 回忆 → Google Custom Search → Bing Web Search。

```bash
$ watchbot add "监控 Gemini API 文档变化"
🤖 分析: "监控 Gemini API 文档变化"

🤖 建议监控 (来源: llm)：
  [api_docs] Gemini API
  https://ai.google.dev/gemini-api/docs
确认添加？[Y/n]: y
✅ 已添加: Gemini API (1 个页面)
```

## 架构

### 两阶段检查

```text
Phase 1: 全局抓取（按 URL 去重）
  ┌─────────────────────────┐
  │ 所有 active pages       │ → Fetch → Diff → LLM 分析
  │ (同一 URL 只抓取一次)    │ → 保存 Change 记录
  └─────────────────────────┘

Phase 2: 按用户聚合通知
  ┌─────────────────────────┐
  │ 每个 subscriber         │ → 筛选订阅的竞品变化
  │                         │ → 合并为一条 Digest
  │                         │ → 发送邮件/Telegram
  └─────────────────────────┘
```

### 数据库

SQLite 持久化存储，6 张表：

| 表 | 说明 |
| --- | --- |
| `competitors` | 竞品（全局，按 domain 去重） |
| `pages` | 监控页面（全局，按 URL 去重） |
| `snapshots` | 内容快照 |
| `changes` | 检测到的变化记录 |
| `subscribers` | 订阅者（email） |
| `subscriptions` | 订阅关系（多对多） |

### 去重效果

| 场景 | 100 用户 | 1000 用户 |
| --- | --- | --- |
| 平均每人 3 个竞品 | 300 订阅 | 3000 订阅 |
| 去重后唯一竞品 | ~80 个 | ~300 个 |
| 每竞品 3 个页面 | 240 URL | 900 URL |
| **每轮抓取** | **240 次** ✅ | **900 次** ✅ |
| 无去重抓取 | 900 次 ❌ | 9000 次 ❌ |
| **节省** | **73%** | **90%** |

## 环境变量

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `LLM_API_KEY` | 分析时必填 | — | LLM API 密钥 |
| `LLM_PROVIDER` | 否 | `openai` | LLM 提供商 |
| `LLM_MODEL` | 否 | `gpt-4o-mini` | 模型名称 |
| `WATCHBOT_DB` | 否 | `data/watchbot.db` | 数据库路径 |
| `TELEGRAM_BOT_TOKEN` | 否 | — | Telegram 通知 |
| `TELEGRAM_CHANNEL_ID` | 否 | — | Telegram 频道 ID |
| `SMTP_HOST` | 否 | — | SMTP 服务器（启用邮件通知） |
| `SMTP_PORT` | 否 | `587` | SMTP 端口 |
| `SMTP_FROM` | 否 | — | 发件邮箱 |
| `SMTP_PASSWORD` | 否 | — | SMTP 密码 |
| `GOOGLE_API_KEY` | 否 | — | Google Custom Search API |
| `GOOGLE_CX` | 否 | — | Google CSE Engine ID |
| `BING_API_KEY` | 否 | — | Bing Web Search API |

## 部署

### Crontab

```crontab
# 每 6 小时检查一次
0 */6 * * * cd /opt/devkit-suite && export $(grep -v '^#' .env | xargs) && ./bin/watchbot check >> /var/log/watchbot.log 2>&1
```

### Systemd

见 [部署指南](deployment_guide.md) 第 3 节。

### 一键部署

```bash
chmod +x deploy/setup.sh && ./deploy/setup.sh
```

自动创建 WatchBot 数据库、配置环境变量、设置 systemd 服务。
