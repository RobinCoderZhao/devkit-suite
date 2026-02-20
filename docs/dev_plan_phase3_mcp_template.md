# 开发计划 — Phase 3：MCP Server 模板包

> 前置依赖：无（独立开发，但可复用 robotIM 已有代码）
>
> 项目路径：`devkit-suite/templates/mcp-server/`

---

## Step 3.1：模板包目录结构

```
templates/mcp-server/
├── cmd/
│   └── server/
│       └── main.go                  # 入口：5 行代码启动 Server
├── pkg/
│   ├── mcp/
│   │   ├── server.go               # Server 核心（注册 Tool/Resource/Prompt）
│   │   ├── types.go                # JSON-RPC 类型定义
│   │   ├── handler.go              # 请求分发器
│   │   ├── tool.go                 # Tool 注册表 + 执行器
│   │   ├── resource.go             # Resource Provider
│   │   ├── prompt.go               # Prompt Manager
│   │   └── errors.go              # MCP 标准错误码
│   ├── transport/
│   │   ├── transport.go            # Transport 接口
│   │   ├── stdio.go                # stdio 传输实现
│   │   ├── sse.go                  # SSE (HTTP + Server-Sent Events)
│   │   └── websocket.go           # WebSocket（Enterprise）
│   ├── middleware/
│   │   ├── middleware.go           # Middleware 接口
│   │   ├── auth.go                 # API Key / OAuth 认证
│   │   ├── ratelimit.go           # 速率限制
│   │   ├── logging.go             # 结构化日志 (slog)
│   │   ├── metrics.go             # Prometheus 指标
│   │   └── recovery.go           # panic 恢复
│   └── session/
│       ├── session.go              # Session 接口
│       ├── memory.go               # 内存 Session Store
│       └── redis.go                # Redis Session Store（Enterprise）
├── examples/
│   ├── calculator/                 # 示例 1：计算器（最简）
│   │   └── main.go
│   ├── database/                   # 示例 2：数据库查询
│   │   └── main.go
│   ├── web_scraper/               # 示例 3：网页抓取
│   │   └── main.go
│   ├── file_manager/              # 示例 4：文件管理
│   │   └── main.go
│   ├── api_proxy/                 # 示例 5：API 代理（Pro）
│   │   └── main.go
│   └── multi_tool/                # 示例 6：组合多 Tool（Pro）
│       └── main.go
├── deploy/
│   ├── Dockerfile                  # Docker 构建
│   └── k8s/                       # K8s 清单（Enterprise）
│       ├── deployment.yaml
│       └── service.yaml
├── docs/
│   ├── quickstart.md              # 快速入门（所有版本）
│   ├── architecture.md            # 架构详解（Pro+）
│   ├── transport_guide.md         # 传输层指南（Pro+）
│   └── deployment.md             # 部署指南（Pro+）
├── go.mod                          # 独立 go.mod (module 名: mcp-server-template)
├── go.sum
├── LICENSE
├── README.md
└── Makefile
```

## Step 3.2：核心 Server API 设计

```go
// pkg/mcp/server.go
package mcp

type Server struct {
    name       string
    version    string
    tools      *ToolRegistry
    resources  *ResourceProvider
    prompts    *PromptManager
    transport  transport.Transport
    middleware []Middleware
    logger     *slog.Logger
}

func NewServer(name, version string) *Server

// Tool 注册
func (s *Server) RegisterTool(tool Tool) error
func (s *Server) RegisterTools(tools ...Tool) error

// Resource 注册
func (s *Server) RegisterResource(res Resource) error

// Prompt 注册
func (s *Server) RegisterPrompt(prompt Prompt) error

// 中间件
func (s *Server) Use(mw ...Middleware)

// 传输层配置
func (s *Server) WithTransport(t transport.Transport) *Server
func (s *Server) WithStdio() *Server                              // 快捷方法
func (s *Server) WithSSE(addr string) *Server                     // 快捷方法

// 启动
func (s *Server) Run() error
func (s *Server) RunContext(ctx context.Context) error

// Tool 类型定义
type Tool struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    InputSchema Schema                 `json:"inputSchema"`
    Handler     ToolHandler            `json:"-"`
}

type ToolHandler func(ctx context.Context, input map[string]any) (*ToolResult, error)

type ToolResult struct {
    Content []ContentBlock `json:"content"`
    IsError bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
    Type string `json:"type"`          // "text" / "image" / "resource"
    Text string `json:"text,omitempty"`
}

// 便捷构造
func TextResult(text string) *ToolResult
func ErrorResult(err error) *ToolResult
```

## Step 3.3：JSON-RPC 类型

```go
// pkg/mcp/types.go
type JSONRPCRequest struct {
    JSONRPC string          `json:"jsonrpc"`     // "2.0"
    ID      any             `json:"id"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
    JSONRPC string       `json:"jsonrpc"`
    ID      any          `json:"id"`
    Result  any          `json:"result,omitempty"`
    Error   *RPCError    `json:"error,omitempty"`
}

type RPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}

// MCP 标准方法
const (
    MethodInitialize  = "initialize"
    MethodToolsList   = "tools/list"
    MethodToolsCall   = "tools/call"
    MethodResourcesList = "resources/list"
    MethodResourcesRead = "resources/read"
    MethodPromptsList = "prompts/list"
    MethodPromptsGet  = "prompts/get"
)
```

## Step 3.4：从 robotIM 提取要点

需要从你现有的 robotIM MCP 实现中提取以下关键代码：

| 功能 | robotIM 位置 | 提取要点 |
|------|-------------|---------|
| JSON-RPC 分发 | `mcp_simple.go` | HandleJSONRPC + 方法路由 |
| SSE 传输 | `mcp_simple.go` | SSE 事件推送 + session ID |
| Session 管理 | `mcp_simple.go` | Session 创建/查询/过期清理 |
| Tool 注册 | `mcp_simple.go` | ToolRegistry 动态注册 |
| 错误处理 | `mcp_simple.go` | JSON-RPC 错误码规范 |

### 重构原则

1. **去除 robotIM 业务代码**：只保留 MCP 协议层
2. **泛型化**：Tool Handler 使用 `map[string]any` 而非具体类型
3. **配置化**：所有参数可通过配置传入
4. **文档化**：每个导出函数都要有详细注释

## Step 3.5：Starter / Professional / Enterprise 版本区分

```
# 构建脚本：根据 tag 打包不同版本
Makefile:
  build-starter:       # 只编译 stdio + 2 example
  build-professional:  # stdio + SSE + WS + 全部 example + 文档
  build-enterprise:    # 全部 + K8s + Redis session + 咨询凭证

# 通过 Go build tags 控制
// +build !starter
// SSE transport 代码只在 professional+ 版本中编译
```

## Step 3.6：开发顺序 & 验证

| 序号 | 任务 | 验证标准 | 预计时间 |
|------|------|---------|---------|
| 1 | 初始化 `templates/mcp-server/` 独立 go.mod | `go build` 通过 | 30min |
| 2 | 实现 `types.go` + `errors.go` | 编译通过 | 1h |
| 3 | 实现 `server.go` + `tool.go` + `handler.go` | 注册 Tool 可执行 | 3h |
| 4 | 实现 `transport/stdio.go` | Claude Desktop 可连接 | 2h |
| 5 | 实现 `transport/sse.go` + Session | HTTP 端点可用 | 3h |
| 6 | 实现 Middleware 链 | auth + logging 可用 | 2h |
| 7 | 编写 Calculator + Database 示例 | 示例可运行 | 2h |
| 8 | 编写文档 (quickstart + architecture) | Markdown 完整 | 2h |
| 9 | Gumroad 上架 | 购买链接可用 | 1h |
| **总计** | | | **约 16h（2-3 天）** |
