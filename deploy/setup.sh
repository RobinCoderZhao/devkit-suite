#!/bin/bash
#
# DevKit Suite — 一键部署脚本
# 适用于: Ubuntu 22.04 / Debian 12 (阿里云新加坡 ECS)
#
# 使用方法:
#   curl -sSL https://raw.githubusercontent.com/RobinCoderZhao/API-Change-Sentinel/main/deploy/setup.sh | bash
#   或:
#   chmod +x deploy/setup.sh && ./deploy/setup.sh
#
set -euo pipefail

# ===========================
# 颜色输出
# ===========================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()  { echo -e "${GREEN}[✓]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
fail() { echo -e "${RED}[✗]${NC} $1"; exit 1; }
step() { echo -e "\n${BLUE}==>${NC} ${BLUE}$1${NC}"; }

# ===========================
# 配置变量
# ===========================
GO_VERSION="1.25.0"
APP_USER="deploy"
APP_DIR="/opt/devkit-suite"
DATA_DIR="${APP_DIR}/data"
LOG_DIR="/var/log/devkit-suite"
REPO_URL="https://github.com/RobinCoderZhao/API-Change-Sentinel.git"
ENV_FILE="${APP_DIR}/.env"

# ===========================
# 检查 root 权限
# ===========================
if [ "$(id -u)" -ne 0 ]; then
    fail "请使用 root 用户运行此脚本: sudo bash deploy/setup.sh"
fi

echo ""
echo "╔══════════════════════════════════════╗"
echo "║   DevKit Suite 一键部署脚本 v1.0     ║"
echo "║   目标: 阿里云新加坡 ECS             ║"
echo "╚══════════════════════════════════════╝"
echo ""

# ===========================
# Step 1: 系统更新 + 基础依赖
# ===========================
step "Step 1/8: 系统更新与基础依赖安装"

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get upgrade -y -qq
apt-get install -y -qq git wget curl htop unzip ufw fail2ban ca-certificates tzdata sqlite3

# 设置时区
timedatectl set-timezone Asia/Shanghai
log "系统更新完成，时区设为 Asia/Shanghai"

# ===========================
# Step 2: 创建应用用户
# ===========================
step "Step 2/8: 创建应用用户"

if id "${APP_USER}" &>/dev/null; then
    log "用户 ${APP_USER} 已存在，跳过"
else
    useradd -m -s /bin/bash "${APP_USER}"
    log "创建用户 ${APP_USER}"
fi

# ===========================
# Step 3: 安装 Go
# ===========================
step "Step 3/8: 安装 Go ${GO_VERSION}"

if command -v go &>/dev/null && go version | grep -q "${GO_VERSION}"; then
    log "Go ${GO_VERSION} 已安装，跳过"
else
    GO_TAR="go${GO_VERSION}.linux-amd64.tar.gz"
    wget -q "https://go.dev/dl/${GO_TAR}" -O "/tmp/${GO_TAR}"
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "/tmp/${GO_TAR}"
    rm "/tmp/${GO_TAR}"

    # 为所有用户配置 Go 环境
    cat > /etc/profile.d/golang.sh << 'GOEOF'
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
GOEOF
    chmod +x /etc/profile.d/golang.sh
    source /etc/profile.d/golang.sh
    log "Go $(go version) 安装完成"
fi

export PATH=$PATH:/usr/local/go/bin

# ===========================
# Step 4: 克隆代码 + 构建
# ===========================
step "Step 4/8: 克隆代码并构建"

mkdir -p "${APP_DIR}" "${DATA_DIR}" "${LOG_DIR}"

if [ -d "${APP_DIR}/.git" ]; then
    cd "${APP_DIR}"
    git pull -q
    log "代码已更新 (git pull)"
else
    git clone -q "${REPO_URL}" "${APP_DIR}"
    log "代码克隆完成"
fi

cd "${APP_DIR}"
/usr/local/go/bin/go build -trimpath -ldflags="-s -w" -o bin/newsbot ./cmd/newsbot
/usr/local/go/bin/go build -trimpath -ldflags="-s -w" -o bin/devkit ./cmd/devkit
/usr/local/go/bin/go build -trimpath -ldflags="-s -w" -o bin/watchbot ./cmd/watchbot
log "构建完成: newsbot=$(du -h bin/newsbot | cut -f1), devkit=$(du -h bin/devkit | cut -f1), watchbot=$(du -h bin/watchbot | cut -f1)"

# ===========================
# Step 5: 初始化数据库
# ===========================
step "Step 5/8: 初始化 SQLite 数据库"

DB_PATH="${DATA_DIR}/newsbot.db"

if [ -f "${DB_PATH}" ]; then
    log "数据库已存在: ${DB_PATH}，跳过初始化"
else
    sqlite3 "${DB_PATH}" << 'SQLEOF'
PRAGMA journal_mode=WAL;

CREATE TABLE IF NOT EXISTS articles (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT NOT NULL,
    url         TEXT NOT NULL UNIQUE,
    source      TEXT NOT NULL,
    author      TEXT,
    content     TEXT,
    published_at TIMESTAMP,
    fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tags        TEXT
);

CREATE TABLE IF NOT EXISTS digests (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    date         TEXT NOT NULL UNIQUE,
    headlines    TEXT NOT NULL,
    summary      TEXT,
    tokens_used  INTEGER DEFAULT 0,
    cost         REAL DEFAULT 0,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_articles_source ON articles(source);
CREATE INDEX IF NOT EXISTS idx_articles_fetched ON articles(fetched_at);
CREATE INDEX IF NOT EXISTS idx_digests_date ON digests(date);
SQLEOF
    log "数据库初始化完成: ${DB_PATH}"
fi

# ===========================
# Step 6: 配置环境变量
# ===========================
step "Step 6/8: 配置环境变量"

if [ -f "${ENV_FILE}" ]; then
    warn "环境文件已存在: ${ENV_FILE}，跳过（请手动编辑）"
else
    cat > "${ENV_FILE}" << 'ENVEOF'
# ====================================
# DevKit Suite 环境变量配置
# 请修改以下值后重启服务:
#   sudo systemctl restart newsbot watchbot
# ====================================

# LLM 配置（必填）
LLM_PROVIDER=openai
LLM_API_KEY=sk-your-api-key-here
LLM_MODEL=gpt-4o-mini

# Telegram 推送（可选，留空则输出到日志）
TELEGRAM_BOT_TOKEN=
TELEGRAM_CHANNEL_ID=

# NewsBot 数据库路径
NEWSBOT_DB=/opt/devkit-suite/data/newsbot.db
ENVEOF
    chmod 600 "${ENV_FILE}"
    log "环境文件已创建: ${ENV_FILE}"
    warn "⚠️  请编辑 ${ENV_FILE} 填入你的 API Key！"
fi

# ===========================
# Step 7: 配置 Systemd 服务
# ===========================
step "Step 7/8: 配置 Systemd 服务"

# NewsBot 服务
cat > /etc/systemd/system/newsbot.service << EOF
[Unit]
Description=NewsBot AI Daily Digest
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${APP_USER}
WorkingDirectory=${APP_DIR}
EnvironmentFile=${ENV_FILE}
ExecStart=${APP_DIR}/bin/newsbot serve
Restart=always
RestartSec=30
StandardOutput=journal
StandardError=journal
SyslogIdentifier=newsbot

[Install]
WantedBy=multi-user.target
EOF

# WatchBot 服务
cat > /etc/systemd/system/watchbot.service << EOF
[Unit]
Description=WatchBot Competitor Monitor
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${APP_USER}
WorkingDirectory=${APP_DIR}
EnvironmentFile=${ENV_FILE}
ExecStart=${APP_DIR}/bin/watchbot serve
Restart=always
RestartSec=30
StandardOutput=journal
StandardError=journal
SyslogIdentifier=watchbot

[Install]
WantedBy=multi-user.target
EOF

# 数据备份定时任务
cat > /etc/systemd/system/devkit-backup.service << EOF
[Unit]
Description=DevKit Suite Database Backup

[Service]
Type=oneshot
ExecStart=/bin/bash -c 'mkdir -p ${DATA_DIR}/backups && cp ${DATA_DIR}/newsbot.db ${DATA_DIR}/backups/newsbot_\$(date +%%Y%%m%%d_%%H%%M%%S).db && find ${DATA_DIR}/backups -name "*.db" -mtime +30 -delete'
EOF

cat > /etc/systemd/system/devkit-backup.timer << EOF
[Unit]
Description=Weekly DevKit Suite Database Backup

[Timer]
OnCalendar=Sun *-*-* 03:00:00
Persistent=true

[Install]
WantedBy=timers.target
EOF

# 设置文件权限
chown -R "${APP_USER}:${APP_USER}" "${APP_DIR}" "${DATA_DIR}" "${LOG_DIR}"

systemctl daemon-reload
systemctl enable newsbot watchbot devkit-backup.timer
log "Systemd 服务配置完成"

# ===========================
# Step 8: 安全加固 + 防火墙
# ===========================
step "Step 8/8: 安全加固"

# 配置 UFW 防火墙
ufw --force reset > /dev/null 2>&1
ufw default deny incoming > /dev/null
ufw default allow outgoing > /dev/null
ufw allow 22/tcp > /dev/null    # SSH
ufw allow 8080/tcp > /dev/null  # MCP Server（可选）
ufw --force enable > /dev/null
log "防火墙已配置 (SSH:22 + MCP:8080)"

# 启用 fail2ban
systemctl enable fail2ban > /dev/null 2>&1
systemctl start fail2ban > /dev/null 2>&1
log "fail2ban 防暴力破解已启用"

# ===========================
# 完成！
# ===========================
echo ""
echo "╔══════════════════════════════════════════════════════╗"
echo "║              ✅ 部署完成！                           ║"
echo "╠══════════════════════════════════════════════════════╣"
echo "║                                                      ║"
echo "║  📁 应用目录:  ${APP_DIR}                    ║"
echo "║  📁 数据目录:  ${DATA_DIR}               ║"
echo "║  📁 环境文件:  ${ENV_FILE}               ║"
echo "║                                                      ║"
echo "║  ⚠️  下一步:                                         ║"
echo "║  1. 编辑环境文件填入 API Key:                        ║"
echo "║     nano ${ENV_FILE}                     ║"
echo "║                                                      ║"
echo "║  2. 启动服务:                                        ║"
echo "║     sudo systemctl start newsbot watchbot            ║"
echo "║                                                      ║"
echo "║  3. 验证运行:                                        ║"
echo "║     sudo systemctl status newsbot                    ║"
echo "║     sudo journalctl -u newsbot -f                    ║"
echo "║                                                      ║"
echo "║  4. 手动测试:                                        ║"
echo "║     sudo -u deploy ${APP_DIR}/bin/newsbot run    ║"
echo "║     sudo -u deploy ${APP_DIR}/bin/watchbot check ║"
echo "║                                                      ║"
echo "╚══════════════════════════════════════════════════════╝"
echo ""
