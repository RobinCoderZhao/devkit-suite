# 🚀 DevKit Suite — AI 驱动的开发者工具套件

> Go 1.25 Monorepo · 4 产品 · 7 共享包 · 21 测试全通过

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![Build Status](https://img.shields.io/badge/build-passing-success)](.)
[![Tests](https://img.shields.io/badge/tests-21%20passed-success)](.)

---

## 📦 产品矩阵

| 产品 | 说明 | 命令 | 类型 |
|------|------|------|------|
| **NewsBot** | AI 热点日报 Bot | `newsbot run \| serve` | CLI + Service |
| **DevKit** | AI 开发者 CLI 工具 | `devkit commit \| review` | CLI |
| **MCP Template** | MCP 服务器框架 | `import pkg/mcpserver` | Library |
| **WatchBot** | 竞品监控 Bot | `watchbot check \| serve` | CLI + Service |

## ⚡ 快速开始

### 环境要求

- **Go 1.25+**（通过 `goenv` 管理）
- Git 2.x+
- LLM API Key（OpenAI / Gemini / Claude 任选一个）

### 安装

```bash
# 克隆仓库
git clone https://github.com/RobinCoderZhao/API-Change-Sentinel.git
cd API-Change-Sentinel

# 构建所有产品
make all

# 验证
./bin/newsbot version
./bin/devkit version
./bin/watchbot version
```

### 一键运行

```bash
# 🤖 NewsBot — 抓取 AI 新闻并生成日报
export LLM_API_KEY="your-api-key"
./bin/newsbot run

# 🛠 DevKit — AI 生成 commit message
git add .
./bin/devkit commit

# 🔍 DevKit — AI 代码审查
./bin/devkit review

# 👀 WatchBot — 检查竞品变动
./bin/watchbot check
```

---

## 🏗 架构概览

```
┌─────────────────────────────────────────────────────┐
│                    CLI 入口 (cmd/)                    │
│      newsbot        devkit        watchbot           │
└──────┬──────────────┬──────────────┬─────────────────┘
       │              │              │
┌──────▼──────┐ ┌─────▼─────┐ ┌─────▼──────┐
│  NewsBot    │ │  DevKit   │ │  WatchBot  │
│  internal/  │ │  internal/│ │  internal/ │
│  newsbot/   │ │  devkit/  │ │  watchbot/ │
└──────┬──────┘ └─────┬─────┘ └─────┬──────┘
       │              │              │
┌──────▼──────────────▼──────────────▼─────────────────┐
│               共享基础设施 (pkg/)                       │
│                                                       │
│  ┌─────┐ ┌────────┐ ┌───────┐ ┌────────┐ ┌────────┐ │
│  │ llm │ │scraper │ │differ │ │ notify │ │storage │ │
│  │     │ │        │ │       │ │        │ │        │ │
│  │4 LLM│ │HTML    │ │行级   │ │Telegram│ │SQLite  │ │
│  │提供商│ │解析    │ │Diff   │ │Webhook │ │Postgres│ │
│  └─────┘ └────────┘ └───────┘ └────────┘ └────────┘ │
│                                                       │
│  ┌────────┐ ┌───────────┐                            │
│  │ config │ │ mcpserver │                            │
│  │        │ │           │                            │
│  │YAML+Env│ │stdio+HTTP │                            │
│  │加载    │ │MCP 框架   │                            │
│  └────────┘ └───────────┘                            │
└───────────────────────────────────────────────────────┘
```

> 详细架构文档见 [docs/architecture.md](docs/architecture.md)

---

## 📁 目录结构

```
API-Change-Sentinel/
├── cmd/                        # 入口程序
│   ├── newsbot/main.go         # AI 新闻日报
│   ├── devkit/main.go          # 开发者 CLI
│   └── watchbot/main.go        # 竞品监控
├── internal/                   # 产品专属逻辑
│   ├── newsbot/
│   │   ├── sources/            # 数据源 (HackerNews + RSS)
│   │   ├── analyzer/           # LLM 分析器
│   │   ├── publisher/          # 日报发布
│   │   ├── store/              # SQLite 持久化
│   │   └── scheduler/          # 定时调度
│   ├── devkit/
│   │   ├── git/                # Git 操作封装
│   │   ├── prompt/             # LLM Prompt 模板
│   │   └── config/             # 项目/全局配置
│   └── watchbot/               # 监控 Pipeline
├── pkg/                        # 共享包（可被外部引用）
│   ├── llm/                    # 统一 LLM 客户端
│   ├── scraper/                # HTTP 爬虫 + 文本提取
│   ├── differ/                 # 文本 Diff 引擎
│   ├── notify/                 # 通知调度器
│   ├── storage/                # 数据库抽象层
│   ├── config/                 # 配置加载器
│   └── mcpserver/              # MCP 服务器框架
├── docs/                       # 产品文档
├── deploy/                     # 部署脚本
├── configs/                    # 配置文件模板
├── Makefile                    # 构建脚本
├── go.mod                      # Go modules (1.25)
└── .go-version                 # goenv 版本锁定
```

---

## 🤖 产品详细说明

### NewsBot — AI 热点日报

从 HackerNews、TechCrunch、MIT Tech Review 等源抓取 AI 新闻，通过 LLM 去重、评分和摘要，生成中文日报推送到 Telegram。

```bash
# 单次运行（抓取 → 分析 → 推送）
LLM_API_KEY=sk-xxx TELEGRAM_BOT_TOKEN=xxx TELEGRAM_CHANNEL_ID=@channel ./bin/newsbot run

# 定时服务模式（每 24 小时自动执行）
./bin/newsbot serve
```

**环境变量：**

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `LLM_API_KEY` | ✅ | — | LLM API 密钥 |
| `LLM_PROVIDER` | ❌ | `openai` | `openai` / `gemini` / `claude` / `ollama` |
| `LLM_MODEL` | ❌ | `gpt-4o-mini` | 模型名称 |
| `TELEGRAM_BOT_TOKEN` | ❌ | — | Telegram Bot Token |
| `TELEGRAM_CHANNEL_ID` | ❌ | — | Telegram 频道 ID |
| `NEWSBOT_DB` | ❌ | `newsbot.db` | SQLite 数据库路径 |

### DevKit — AI 开发者 CLI

**`devkit commit`** — 分析 git diff，生成 Conventional Commits 规范的 commit message：

```bash
git add .
./bin/devkit commit           # 交互模式，确认后提交
./bin/devkit commit -a        # 自动 stage 所有文件
./bin/devkit commit -y        # 跳过确认直接提交
```

**`devkit review`** — AI 代码审查，输出评分和建议：

```bash
./bin/devkit review           # 格式化输出
./bin/devkit review --json    # JSON 格式输出
```

**配置文件** `.devkit.yaml`（项目级或 `~/.devkit.yaml` 全局）：

```yaml
llm:
  provider: openai
  model: gpt-4o-mini
  api_key: ${OPENAI_API_KEY}
commit:
  language: en
  max_length: 72
  auto_stage: false
review:
  output_format: text
```

### MCP Server Template — MCP 服务器框架

从 [npinterface-mcp](https://github.com/RobinCoderZhao/npinterface-mcp) 提取的通用 MCP（Model Context Protocol）服务器框架：

```go
package main

import (
    "github.com/RobinCoderZhao/API-Change-Sentinel/pkg/mcpserver"
)

// 定义工具
type GreetTool struct { mcpserver.BaseTool }

func NewGreetTool() *GreetTool {
    return &GreetTool{BaseTool: mcpserver.BaseTool{
        ToolName:        "greet",
        ToolDescription: "Say hello",
        ToolSchema:      map[string]any{
            "type": "object",
            "properties": map[string]any{
                "name": map[string]any{"type": "string"},
            },
        },
    }}
}

func (t *GreetTool) Execute(args map[string]any) (*mcpserver.ToolCallResult, error) {
    name, _ := args["name"].(string)
    return mcpserver.TextResult("Hello, " + name + "!"), nil
}

func main() {
    s := mcpserver.New("my-mcp-server", "1.0.0")
    s.Use(mcpserver.LoggingMiddleware(nil))   // 请求日志
    s.Use(mcpserver.RecoveryMiddleware())      // Panic 恢复
    s.RegisterTool(NewGreetTool())

    // 选择传输方式
    s.RunStdio()          // stdio 模式（Claude/Cursor 等）
    // s.RunHTTP(":8080") // HTTP + SSE 模式
}
```

**特性：** JSON-RPC 2.0、stdio + HTTP/SSE 双传输、Middleware 链、Session 管理、BaseTool 基类

### WatchBot — 竞品监控

监控竞品网站变化，自动生成 AI 分析报告：

```bash
# 查看监控目标
./bin/watchbot targets

# 单次检查
./bin/watchbot check

# 定时服务模式（每 6 小时检查）
./bin/watchbot serve
```

**默认监控目标：** OpenAI API Docs、OpenAI Changelog、Anthropic API、Gemini API、HuggingFace Blog

---

## 🛠 开发指南

```bash
# 构建
make all                    # 构建所有二进制
make build-newsbot          # 单独构建

# 测试
make test                   # 全量测试
make test-pkg               # 只测试共享包

# 代码质量
make lint                   # golangci-lint
make tidy                   # go mod tidy

# 清理
make clean                  # 删除 bin/
```

### 共享包 API

```go
// LLM — 统一客户端
client, _ := llm.NewClient(llm.Config{Provider: llm.OpenAI, APIKey: "sk-xxx"})
resp, _ := client.Generate(ctx, &llm.Request{Messages: []llm.Message{{Role: "user", Content: "Hello"}}})

// Scraper — 网页抓取
fetcher := scraper.NewHTTPFetcher()
result, _ := fetcher.Fetch(ctx, "https://example.com", nil)

// Differ — 文本比较
diff := differ.TextDiff(oldText, newText)
fmt.Println(diff.Summary()) // "3 additions, 1 deletions"

// Notify — 通知发送
dispatcher := notify.NewDispatcher()
dispatcher.Register(notify.NewTelegramNotifier(cfg))
dispatcher.SendAll(ctx, notify.Message{Title: "Alert", Body: "Content"})

// Config — 配置加载
var cfg MyConfig
config.Load("config.yaml", &cfg) // YAML + 环境变量覆盖
```

---

## 📄 文档索引

| 文档 | 说明 |
|------|------|
| [架构设计](docs/architecture.md) | 系统架构、数据流、包依赖 |
| [部署指南](docs/deployment_guide.md) | Docker 部署、环境配置、生产运维 |
| [阿里云新加坡一键部署](docs/aliyun_sg_deployment.md) | 选型购买 + 一键脚本部署 |
| [产品总览](docs/product_detail_overview.md) | 产品矩阵与技术栈 |
| [NewsBot 设计](docs/product_1_newsbot.md) | 新闻 Bot 详细设计 |
| [DevKit 设计](docs/product_2_devkit.md) | CLI 工具详细设计 |
| [MCP Template 设计](docs/product_3_mcp_template.md) | MCP 框架设计 |
| [WatchBot 设计](docs/product_4_watchbot.md) | 监控 Bot 详细设计 |
| [共享基础设施](docs/shared_infrastructure.md) | 共享包设计文档 |

## 📊 技术栈

| 层 | 技术 |
|----|------|
| 语言 | Go 1.25 |
| CLI 框架 | Cobra |
| LLM | OpenAI / Gemini / Claude / Ollama |
| 数据库 | SQLite (modernc.org/sqlite, 纯 Go) |
| 通知 | Telegram Bot API + Webhook |
| 协议 | MCP (JSON-RPC 2.0) |
| HTML 解析 | golang.org/x/net/html |
| 配置 | YAML + 环境变量 |

## 📜 License

MIT License © 2026 RobinCoderZhao
