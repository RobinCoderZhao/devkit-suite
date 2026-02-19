# 产品 3：MCP Server 模板包 详细设计

## 1. 产品定义

### 1.1 产品愿景
>
> "5 分钟搭建一个生产级 MCP Server，Go 语言，零依赖。"

### 1.2 目标用户

| 画像 | 需求 | 付费意愿 |
|------|------|---------|
| **AI 应用开发者** | 让 AI Agent 调用自己的内部系统 | 高（省 2-3 周开发时间） |
| **企业后端团队** | 标准化 AI 工具接入方式 | 高（企业预算） |
| **MCP 生态贡献者** | 快速构建和发布 MCP Server | 中 |
| **Go 开发者** | 需要 Go 的 MCP 参考实现 | 中 |

### 1.3 功能划分

| 模块 | Starter $29 | Professional $79 | Enterprise $299 |
|------|:-----------:|:-----------------:|:----------------:|
| 核心 JSON-RPC 引擎 | ✅ | ✅ | ✅ |
| stdio 传输 | ✅ | ✅ | ✅ |
| SSE 传输 | ❌ | ✅ | ✅ |
| WebSocket 传输 | ❌ | ✅ | ✅ |
| Session 管理 | ❌ | ✅ | ✅ |
| API Key 认证 | ✅ | ✅ | ✅ |
| OAuth 2.0 认证 | ❌ | ✅ | ✅ |
| 速率限制 | ❌ | ✅ | ✅ |
| 结构化日志 (slog) | ✅ | ✅ | ✅ |
| Prometheus 指标 | ❌ | ✅ | ✅ |
| 示例 Tool (2个) | ✅ | ✅ | ✅ |
| 示例 Tool (全部6个) | ❌ | ✅ | ✅ |
| 快速入门文档 | ✅ | ✅ | ✅ |
| 完整架构文档 | ❌ | ✅ | ✅ |
| 视频教程 | ❌ | ✅ | ✅ |
| Docker 部署 | ❌ | ✅ | ✅ |
| K8s 部署清单 | ❌ | ❌ | ✅ |
| 1 小时技术咨询 | ❌ | ❌ | ✅ |
| 6 个月更新保障 | ❌ | ❌ | ✅ |

---

## 2. 软件架构

### 2.1 核心引擎

```
┌──────────────────────────────────────────────────────────────┐
│                     MCP Server Engine                         │
│                                                              │
│  ┌──────────────────┐    ┌─────────────────────────────┐     │
│  │   Transport       │    │   Protocol Handler           │     │
│  │                   │    │                              │     │
│  │ ┌──────┐         │    │ ┌──────────────────────────┐ │     │
│  │ │stdio │◀───────▶├───▶│ │ JSON-RPC Dispatcher       │ │     │
│  │ └──────┘         │    │ │                            │ │     │
│  │ ┌──────┐         │    │ │ initialize    → handshake  │ │     │
│  │ │ SSE  │◀───────▶│    │ │ tools/list   → registry   │ │     │
│  │ └──────┘         │    │ │ tools/call   → executor   │ │     │
│  │ ┌──────┐         │    │ │ resources/*  → provider   │ │     │
│  │ │  WS  │◀───────▶│    │ │ prompts/*    → manager    │ │     │
│  │ └──────┘         │    │ └──────────────────────────┘ │     │
│  └──────────────────┘    └──────────────┬──────────────┘     │
│                                         │                     │
│  ┌──────────────────────────────────────▼──────────────────┐ │
│  │                    Middleware Chain                       │ │
│  │  Auth → RateLimit → Logging → Metrics → Recovery         │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │                    Tool Registry                          │ │
│  │  ┌────────┐ ┌──────────┐ ┌────────┐ ┌──────────────┐   │ │
│  │  │ Calc   │ │ DB Query │ │ HTTP   │ │ File Manager │   │ │
│  │  └────────┘ └──────────┘ └────────┘ └──────────────┘   │ │
│  └──────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

### 2.2 核心 API 设计

```go
// mcp.go — 框架入口，用户 5 行代码启动 Server
package mcp

type Server struct {
    name      string
    version   string
    tools     *ToolRegistry
    resources *ResourceProvider
    prompts   *PromptManager
    transport Transport
    middleware []Middleware
}

// 用户代码示例：
func main() {
    server := mcp.NewServer("my-server", "1.0.0")

    // 注册 Tool
    server.RegisterTool(mcp.Tool{
        Name:        "get_weather",
        Description: "获取指定城市的天气信息",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "city": {Type: "string", Description: "城市名称"},
            },
            Required: []string{"city"},
        },
        Handler: func(ctx context.Context, input map[string]any) (*mcp.Result, error) {
            city := input["city"].(string)
            weather := fetchWeather(city)
            return mcp.TextResult(weather), nil
        },
    })

    // 配置中间件
    server.Use(mcp.AuthMiddleware("api-key-xxx"))
    server.Use(mcp.LoggingMiddleware())

    // 启动（自动检测 stdio / SSE）
    server.Run()
}
```

### 2.3 Transport 抽象

```go
// transport.go
type Transport interface {
    Start(handler RequestHandler) error
    Stop() error
}

// stdio_transport.go — 最简传输
type StdioTransport struct {
    reader *bufio.Scanner
    writer *json.Encoder
}

// sse_transport.go — HTTP + SSE
type SSETransport struct {
    addr          string
    sessionStore  SessionStore
    router        *http.ServeMux
}

// 关键设计：SSE 传输中的 Session 管理
type SessionStore interface {
    Create() (sessionID string, err error)
    Get(sessionID string) (*Session, error)
    Delete(sessionID string) error
    Cleanup(maxAge time.Duration)          // 定时清理过期 Session
}

type Session struct {
    ID        string
    CreatedAt time.Time
    EventChan chan *SSEEvent       // SSE 事件推送通道
    Metadata  map[string]string   // 自定义元数据
}
```

### 2.4 示例 Tool 清单

| Tool | 复杂度 | 教学目的 |
|------|--------|---------|
| **Calculator** | 低 | 入门：最简单的 Tool |
| **System Info** | 低 | 展示无参数 Tool |
| **Database Query** | 中 | 参数验证 + 错误处理 |
| **Web Scraper** | 中 | 异步操作 + 超时控制 |
| **File Manager** | 中 | 安全性考虑（路径验证） |
| **API Proxy** | 高 | 认证透传 + 请求转发 |

---

## 3. 部署方案

### 3.1 Docker

```dockerfile
# Dockerfile（模板自带）
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /mcp-server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /mcp-server /usr/local/bin/mcp-server
EXPOSE 8080
ENTRYPOINT ["mcp-server"]
```

### 3.2 Kubernetes（Enterprise 版）

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-server
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mcp-server
  template:
    spec:
      containers:
      - name: mcp-server
        image: yourname/mcp-server:latest
        ports:
        - containerPort: 8080
        env:
        - name: MCP_TRANSPORT
          value: "sse"
        resources:
          requests: { memory: "64Mi", cpu: "100m" }
          limits: { memory: "256Mi", cpu: "500m" }
        livenessProbe:
          httpGet: { path: /health, port: 8080 }
        readinessProbe:
          httpGet: { path: /ready, port: 8080 }
```

---

## 4. 商业化

### 4.1 销售渠道

| 渠道 | 实施 | 启动成本 |
|------|------|---------|
| **Gumroad** | 上传 zip 包，设置价格 | $0 |
| **GitHub Sponsors Tier** | 赞助者获得私有仓库访问权 | $0 |
| **Landing Page** | Carrd.co 或自建 | $0-19/年 |

### 4.2 内容营销

| 内容 | 渠道 | 目的 |
|------|------|------|
| "Build an MCP Server in Go in 10 mins" 博客 | Medium/Dev.to | SEO 引流 |
| YouTube 视频教程（3 集） | YouTube | 展示产品能力 |
| MCP 协议详解文章（中文） | 掘金/知乎 | 中文市场 |
| GitHub 开源 Starter 版核心 | GitHub | Star → 信任 → 付费 |

### 4.3 成本结构

| 项目 | 费用 |
|------|------|
| 开发时间 | 2 周（你已有代码基础） |
| 域名 | $10/年 |
| Gumroad 手续费 | 10% |
| 视频录制 | 免费（OBS + 麦克风） |
| **总启动成本** | **< $20** |
