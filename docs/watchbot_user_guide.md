# WatchBot 竞品监控 — 使用手册

## 产品概述

WatchBot 是一款 AI 驱动的竞品监控工具。它自动抓取竞品网页（API 文档、Changelog、定价页）、检测变化，并使用 LLM 生成智能分析报告。

**核心流程：**

```
目标网页  →  定时抓取  →  文本提取  →  Diff 对比  →  LLM 分析  →  告警通知
```

---

## 1. 快速开始

### 1.1 构建

```bash
cd API-Change-Sentinel
go build -trimpath -ldflags="-s -w" -o bin/watchbot ./cmd/watchbot
```

### 1.2 最简运行

```bash
# 无需任何配置即可运行（使用内置默认目标，无 LLM 分析）
./bin/watchbot check
```

### 1.3 完整运行（含 LLM 分析）

```bash
export LLM_PROVIDER=minimax
export LLM_API_KEY=sk-xxx
export LLM_MODEL=MiniMax-M2.5

./bin/watchbot check
```

---

## 2. CLI 命令

| 命令 | 说明 |
|------|------|
| `watchbot check` | 运行一次全量检查，对比所有目标 |
| `watchbot serve` | 以守护进程模式运行，每 6 小时自动检查 |
| `watchbot targets` | 列出当前所有监控目标 |
| `watchbot version` | 显示版本号 |

### 2.1 `watchbot check` — 单次检查

```bash
$ ./bin/watchbot check

# 输出示例：
2026/02/19 INFO starting WatchBot check targets=5
2026/02/19 INFO first snapshot captured target="OpenAI API Docs" size=45231
2026/02/19 INFO no changes detected target="OpenAI Changelog"
2026/02/19 INFO changes detected target="Anthropic API Docs" additions=12 deletions=3

🟡 [important] Anthropic API Docs
Anthropic 新增了 claude-4-sonnet 模型参数说明，支持 128K 上下文...

2026/02/19 INFO check complete targets=5 alerts=1
```

**注意**：首次运行时所有目标都是"首次抓取"，不会产生 diff。至少需要运行 **两次** 才能检测变化。

### 2.2 `watchbot serve` — 守护进程模式

```bash
$ ./bin/watchbot serve

# 立即运行一次，之后每 6 小时重复
2026/02/19 INFO WatchBot serving interval=6h0m0s targets=5
```

使用 `Ctrl+C` 优雅停止。

### 2.3 `watchbot targets` — 查看监控目标

```bash
$ ./bin/watchbot targets

监控目标 (5):

  1. [api_docs] OpenAI API Docs
     URL: https://platform.openai.com/docs/api-reference
     间隔: 6h

  2. [changelog] OpenAI Changelog
     URL: https://platform.openai.com/docs/changelog
     间隔: 6h

  3. [api_docs] Anthropic API Docs
     URL: https://docs.anthropic.com/en/api
     间隔: 6h

  4. [api_docs] Gemini API Docs
     URL: https://ai.google.dev/gemini-api/docs
     间隔: 6h

  5. [blog] HuggingFace Blog
     URL: https://huggingface.co/blog
     间隔: 24h
```

---

## 3. 默认监控目标

WatchBot 内置了 5 个 AI 行业关键监控目标：

| 目标 | 类型 | URL | 检查间隔 | 监控重点 |
|------|------|-----|---------|---------|
| OpenAI API Docs | api_docs | platform.openai.com/docs/api-reference | 6h | API 接口变更、新模型上线 |
| OpenAI Changelog | changelog | platform.openai.com/docs/changelog | 6h | 版本更新、弃用通知 |
| Anthropic API Docs | api_docs | docs.anthropic.com/en/api | 6h | Claude 模型变更 |
| Gemini API Docs | api_docs | ai.google.dev/gemini-api/docs | 6h | Gemini 接口变化 |
| HuggingFace Blog | blog | huggingface.co/blog | 24h | 开源模型发布 |

---

## 4. 检测流程详解

### 4.1 Pipeline 四步流程

```
  ┌─────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
  │  Fetch   │────▶│   Diff   │────▶│ Analyze  │────▶│  Alert   │
  │ 抓取页面  │     │ 文本对比  │     │ LLM 分析  │     │ 推送通知  │
  └─────────┘     └──────────┘     └──────────┘     └──────────┘
```

1. **Fetch**：HTTP 抓取目标页面，提取干净文本（去掉导航、脚本、CSS）
2. **Diff**：与上次快照做文本 diff，计算增删行数
3. **Analyze**：如果有变化且配置了 LLM，生成竞品分析报告
4. **Alert**：通过 Telegram 或 stdout 发送告警

### 4.2 告警级别

| 级别 | Emoji | 触发场景 |
|------|-------|---------|
| 🔴 Critical | 重大 API 变更、破坏性更新 |
| 🟡 Important | 功能新增、模型上线 |
| 🟢 Minor | 文档措辞调整、排版变化 |

**无 LLM 时**默认所有变化为 `important` 级别。

### 4.3 LLM 分析 Prompt

WatchBot 会将 diff 发送给 LLM，请求分析：

1. 变化的含义是什么？
2. 对竞争策略有什么影响？
3. 建议的应对措施
4. 严重性分类（CRITICAL / IMPORTANT / MINOR）

---

## 5. 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `LLM_PROVIDER` | `openai` | LLM 提供商: openai/minimax/gemini/claude/ollama |
| `LLM_API_KEY` | — | LLM API 密钥 |
| `LLM_MODEL` | `gpt-4o-mini` | 模型名称 |
| `TELEGRAM_BOT_TOKEN` | — | Telegram Bot Token（可选） |
| `TELEGRAM_CHANNEL_ID` | — | Telegram 频道 ID（可选） |

**不配置 LLM 时**：WatchBot 仍可运行，但不生成 AI 分析，只输出 diff 统计。

**不配置 Telegram 时**：告警输出到 stdout。

---

## 6. 部署

### 6.1 Crontab（推荐简单部署）

```crontab
# 每 6 小时运行 WatchBot
0 */6 * * * /opt/devkit-suite/bin/watchbot check >> /var/log/watchbot.log 2>&1
```

### 6.2 Systemd 守护进程

```ini
# /etc/systemd/system/watchbot.service
[Unit]
Description=WatchBot Competitor Monitor
After=network.target

[Service]
Type=simple
User=deploy
ExecStart=/opt/devkit-suite/bin/watchbot serve
Restart=always
RestartSec=30
Environment="LLM_PROVIDER=minimax"
Environment="LLM_API_KEY=sk-xxx"
Environment="LLM_MODEL=MiniMax-M2.5"

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable watchbot
sudo systemctl start watchbot
sudo journalctl -u watchbot -f
```

---

## 7. 当前实现状态

| 功能 | 状态 | 说明 |
|------|------|------|
| CLI 命令 (check/serve/targets) | ✅ | 完整可用 |
| 内置 5 个 AI 监控目标 | ✅ | OpenAI/Anthropic/Gemini/HuggingFace |
| HTTP 页面抓取 + 文本提取 | ✅ | `pkg/scraper` |
| 文本 Diff 引擎 | ✅ | `pkg/differ`，unified diff 格式 |
| LLM 智能分析 | ✅ | 5 家 LLM 提供商可选 |
| Telegram 告警通知 | ✅ | 含 severity emoji |
| stdout 告警输出 | ✅ | 无 Telegram 时自动 fallback |
| 内存快照缓存 | ✅ | MVP 实现，重启后丢失 |
| **自定义监控目标** | 🔜 | 需代码中添加，计划支持 YAML 配置 |
| **持久化快照存储** | 🔜 | 计划 SQLite 存储 |
| **邮件通知** | 🔜 | 可复用 NewsBot 邮件基础设施 |
| **Web 仪表盘** | 🔜 | 设计文档已有，待开发 |
| **变更历史时间线** | 🔜 | 依赖持久化存储 |

---

## 8. 架构

```
cmd/watchbot/main.go           入口、CLI 命令、配置加载
internal/watchbot/watchbot.go  Pipeline: fetch → diff → analyze → alert
pkg/scraper/                   HTTP 抓取 + 文本提取
pkg/differ/                    文本 Diff 引擎
pkg/llm/                      LLM 统一客户端（5 家提供商）
pkg/notify/                   通知分发（Telegram）
```
