# 部署与运维指南

## 1. 本地开发

### 1.1 环境准备

```bash
# 安装 Go 1.25 (via goenv)
goenv install 1.25.0
goenv local 1.25.0
go version  # → go1.25.0

# 克隆并构建
git clone https://github.com/RobinCoderZhao/API-Change-Sentinel.git
cd API-Change-Sentinel
make all
make test
```

### 1.2 配置 LLM

支持 5 种 LLM 提供商，通过环境变量配置：

```bash
# OpenAI (默认)
export LLM_PROVIDER=openai
export LLM_API_KEY=sk-xxxxxxxxxxxxxxxx
export LLM_MODEL=gpt-4o-mini

# Google Gemini
export LLM_PROVIDER=gemini
export LLM_API_KEY=AIzaXXXXXXXXXXXXXX
export LLM_MODEL=gemini-2.0-flash

# Anthropic Claude
export LLM_PROVIDER=claude
export LLM_API_KEY=sk-ant-XXXXXXXX
export LLM_MODEL=claude-3-5-sonnet-20241022

# 本地 Ollama (无需 API Key)
export LLM_PROVIDER=ollama
export LLM_MODEL=llama3
# OLLAMA_BASE_URL 默认 http://localhost:11434

# MiniMax (推荐，成本低，OpenAI 兼容 API)
export LLM_PROVIDER=minimax
export LLM_API_KEY=sk-api-XXXXXXXX
export LLM_MODEL=MiniMax-M2.5
```

### 1.3 配置邮件通知

```bash
# Gmail SMTP (需要应用专用密码，不是登录密码)
export SMTP_HOST=smtp.gmail.com
export SMTP_PORT=587
export SMTP_FROM=your-email@gmail.com
export SMTP_PASSWORD="xxxx xxxx xxxx xxxx"  # Gmail 应用专用密码
```

### 1.4 管理订阅者

```bash
# 添加订阅者（支持多语言）
./bin/newsbot subscribe --email=user@example.com --lang=zh,en,ja

# 查看所有订阅者
./bin/newsbot subscribers

# 取消订阅
./bin/newsbot unsubscribe --email=user@example.com

# 支持的语言：zh, en, ja, ko, de, es
```

---

## 2. Docker 部署

### 2.1 Dockerfile

在项目根目录创建 `Dockerfile`：

```dockerfile
# === 构建阶段 ===
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/newsbot ./cmd/newsbot && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/watchbot ./cmd/watchbot

# === 运行阶段 ===
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai

COPY --from=builder /bin/newsbot /bin/newsbot
COPY --from=builder /bin/watchbot /bin/watchbot

ENTRYPOINT ["/bin/newsbot"]
CMD ["serve"]
```

### 2.2 Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  newsbot:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: newsbot
    restart: unless-stopped
    entrypoint: ["/bin/newsbot", "serve"]
    environment:
      - LLM_PROVIDER=openai
      - LLM_API_KEY=${LLM_API_KEY}
      - LLM_MODEL=gpt-4o-mini
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TELEGRAM_CHANNEL_ID=${TELEGRAM_CHANNEL_ID}
      - NEWSBOT_DB=/data/newsbot.db
    volumes:
      - newsbot-data:/data

  watchbot:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: watchbot
    restart: unless-stopped
    entrypoint: ["/bin/watchbot", "serve"]
    environment:
      - LLM_PROVIDER=openai
      - LLM_API_KEY=${LLM_API_KEY}
      - LLM_MODEL=gpt-4o-mini
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TELEGRAM_CHANNEL_ID=${TELEGRAM_CHANNEL_ID}

volumes:
  newsbot-data:
```

### 2.3 启动

```bash
# 创建 .env 文件
cat > .env << 'EOF'
LLM_API_KEY=sk-xxxxxxxxxxxxxxxx
TELEGRAM_BOT_TOKEN=123456:ABC-XXXXX
TELEGRAM_CHANNEL_ID=@my_channel
EOF

# 启动服务
docker compose up -d

# 查看日志
docker compose logs -f newsbot
docker compose logs -f watchbot
```

---

## 3. 服务器直接部署

### 3.1 Systemd 服务

```ini
# /etc/systemd/system/newsbot.service
[Unit]
Description=NewsBot AI Daily Digest
After=network.target

[Service]
Type=simple
User=deploy
WorkingDirectory=/opt/devkit-suite
ExecStart=/opt/devkit-suite/bin/newsbot serve
Restart=always
RestartSec=30
Environment="LLM_API_KEY=sk-xxx"
Environment="LLM_PROVIDER=openai"
Environment="LLM_MODEL=gpt-4o-mini"
Environment="TELEGRAM_BOT_TOKEN=xxx"
Environment="TELEGRAM_CHANNEL_ID=@channel"
Environment="NEWSBOT_DB=/opt/devkit-suite/data/newsbot.db"

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable newsbot
sudo systemctl start newsbot
sudo journalctl -u newsbot -f
```

### 3.2 Crontab 部署

如果不需要 serve 模式，可以用 crontab 定时执行：

```crontab
# 每天早上 8 点运行 NewsBot
0 8 * * * /opt/devkit-suite/bin/newsbot run >> /var/log/newsbot.log 2>&1

# 每 6 小时运行 WatchBot
0 */6 * * * /opt/devkit-suite/bin/watchbot check >> /var/log/watchbot.log 2>&1
```

---

## 4. DevKit CLI 安装

DevKit 是本地开发工具，推荐安装到 `$GOPATH/bin`：

```bash
# 方式一：从源码安装
go install github.com/RobinCoderZhao/API-Change-Sentinel/cmd/devkit@latest

# 方式二：直接复制二进制
cp bin/devkit /usr/local/bin/

# 方式三：项目别名
echo 'alias devkit="./bin/devkit"' >> ~/.zshrc
source ~/.zshrc
```

### 4.1 初始配置

```bash
# 必须：设置 LLM API Key
export LLM_API_KEY=sk-xxx

# 可选：创建全局配置
cat > ~/.devkit.yaml << 'EOF'
llm:
  provider: openai
  model: gpt-4o-mini
commit:
  language: en
  max_length: 72
EOF
```

---

## 5. MCP Server 部署

`pkg/mcpserver` 是一个 Go 库，需要嵌入你的项目中使用。

### 5.1 作为 stdio MCP Server（用于 Claude Desktop / Cursor）

```json
// claude_desktop_config.json
{
  "mcpServers": {
    "my-server": {
      "command": "/path/to/your-mcp-binary",
      "args": []
    }
  }
}
```

### 5.2 作为 HTTP MCP Server

```go
server := mcpserver.New("my-server", "1.0.0")
server.RegisterTool(myTool)
server.RunHTTP(":8080")  // 监听 8080 端口
```

端点：

- `POST /mcp` — JSON-RPC 2.0 + SSE
- `GET /api/tools` — 工具列表
- `POST /api/tools/{name}` — 工具调用
- `GET /health` — 健康检查

---

## 6. 监控与运维

### 6.1 健康检查

```bash
# MCP Server 健康检查
curl http://localhost:8080/health

# 查看 NewsBot 数据库状态
sqlite3 newsbot.db "SELECT COUNT(*) FROM articles; SELECT date FROM digests ORDER BY created_at DESC LIMIT 5;"
```

### 6.2 日志

所有产品使用 Go 标准 `log/slog`，输出结构化 JSON 日志：

```bash
# 查看关键事件
journalctl -u newsbot --since "1 hour ago" | grep -E "INFO|ERROR"
```

### 6.3 成本监控

NewsBot 和 DevKit 在每次 LLM 调用后输出 token 消耗和成本：

```
📊 Tokens: 1234 in / 567 out | Cost: $0.0012
```

生产环境建议按月统计 token 消耗，gpt-4o-mini 参考价格：

- 输入：$0.15 / 1M tokens
- 输出：$0.60 / 1M tokens

---

## 7. 环境变量速查表

| 变量 | 适用产品 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `LLM_API_KEY` | 全部 | — | LLM API 密钥 |
| `LLM_PROVIDER` | 全部 | `openai` | 提供商: openai/gemini/claude/ollama/minimax |
| `LLM_MODEL` | 全部 | `gpt-4o-mini` | 模型名称 |
| `OPENAI_API_KEY` | DevKit | — | OpenAI 密钥（备选） |
| `TELEGRAM_BOT_TOKEN` | NewsBot, WatchBot | — | Telegram Bot Token |
| `TELEGRAM_CHANNEL_ID` | NewsBot, WatchBot | — | 频道 ID |
| `NEWSBOT_DB` | NewsBot | `newsbot.db` | NewsBot 数据库路径 |
| `WATCHBOT_DB` | WatchBot | `data/watchbot.db` | WatchBot 数据库路径 |
| `SMTP_HOST` | NewsBot, WatchBot | — | SMTP 服务器 |
| `SMTP_PORT` | NewsBot, WatchBot | `587` | SMTP 端口 (587=STARTTLS, 465=TLS) |
| `SMTP_FROM` | NewsBot, WatchBot | — | 发送者邮箱 |
| `SMTP_PASSWORD` | NewsBot, WatchBot | — | SMTP 密码/应用专用密码 |
| `SMTP_TO` | NewsBot | — | 默认收件人（推荐用 subscribe 命令） |
| `GOOGLE_API_KEY` | WatchBot | — | Google Custom Search API 密钥 |
| `GOOGLE_CX` | WatchBot | — | Google Custom Search Engine ID |
| `BING_API_KEY` | WatchBot | — | Bing Web Search API 密钥 |
| `DEVKIT_LICENSE_KEY` | DevKit | — | 许可证密钥 |

---

## 8. 故障排查

| 问题 | 排查步骤 |
| --- | --- |
| LLM 请求超时 | 检查 `LLM_API_KEY` 是否正确，网络是否可达 |
| Telegram 推送失败 | 确认 Bot 已加入频道且有发送权限 |
| SQLite 锁冲突 | 确保只有一个进程写入，WAL 模式默认开启 |
| MCP Session 404 | 客户端需重新发送 `initialize` 请求 |
| RSS 解析失败 | 部分 RSS 源可能变更格式，检查日志 |
| WatchBot 页面抓取失败 | 部分网站屏蔽爬虫，检查 URL 是否可正常访问 |
| 邮件发送失败 | 确认 SMTP_HOST/SMTP_FROM/SMTP_PASSWORD 配置正确 |
