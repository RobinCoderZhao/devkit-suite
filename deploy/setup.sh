#!/bin/bash
#
# DevKit Suite — 一键部署脚本 (Multi-OS)
# 支持: macOS (Homebrew) / Ubuntu / Debian
#
# 使用方法:
#   chmod +x deploy/setup.sh && ./deploy/setup.sh
#
# 选项:
#   --local    本地开发模式 (不创建系统用户/systemd，数据存当前目录)
#   --server   服务器模式 (创建 deploy 用户/systemd/防火墙，默认在 Linux)
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
# 检测系统
# ===========================
detect_os() {
    case "$(uname -s)" in
        Darwin)  OS="macos" ;;
        Linux)   OS="linux" ;;
        *)       fail "不支持的操作系统: $(uname -s)" ;;
    esac

    ARCH="$(uname -m)"
    case "${ARCH}" in
        x86_64)  GOARCH="amd64" ;;
        aarch64|arm64) GOARCH="arm64" ;;
        *)       fail "不支持的架构: ${ARCH}" ;;
    esac

    if [ "${OS}" = "linux" ]; then
        if [ -f /etc/os-release ]; then
            . /etc/os-release
            DISTRO="${ID}"
        else
            DISTRO="unknown"
        fi
    else
        DISTRO="macos"
    fi
}

# ===========================
# 解析参数
# ===========================
MODE=""
parse_args() {
    for arg in "$@"; do
        case "${arg}" in
            --local)  MODE="local" ;;
            --server) MODE="server" ;;
            --help|-h)
                echo "Usage: ./deploy/setup.sh [--local|--server]"
                echo ""
                echo "  --local   本地开发模式 (数据存当前目录，不配置系统服务)"
                echo "  --server  服务器部署模式 (创建用户、systemd、防火墙)"
                echo ""
                echo "如不指定: macOS 默认 --local, Linux 默认 --server"
                exit 0
                ;;
        esac
    done

    # 自动选择默认模式
    if [ -z "${MODE}" ]; then
        if [ "${OS}" = "macos" ]; then
            MODE="local"
        else
            MODE="server"
        fi
    fi
}

# ===========================
# 配置变量
# ===========================
GO_VERSION="1.25.0"

setup_paths() {
    if [ "${MODE}" = "local" ]; then
        # 本地模式: 使用当前项目目录
        APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
        DATA_DIR="${APP_DIR}/data"
        ENV_FILE="${APP_DIR}/.env"
    else
        # 服务器模式: 使用系统目录
        APP_USER="deploy"
        APP_DIR="/opt/devkit-suite"
        DATA_DIR="${APP_DIR}/data"
        LOG_DIR="/var/log/devkit-suite"
        ENV_FILE="${APP_DIR}/.env"
    fi
}

# ===========================
# 安装系统依赖
# ===========================
install_deps_macos() {
    step "安装系统依赖 (macOS / Homebrew)"

    if ! command -v brew &>/dev/null; then
        warn "Homebrew 未安装，正在安装..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    fi

    # 安装依赖 (跳过已安装的)
    for pkg in git sqlite3; do
        if brew list "${pkg}" &>/dev/null; then
            log "${pkg} 已安装"
        else
            brew install "${pkg}"
            log "${pkg} 安装完成"
        fi
    done
}

install_deps_linux() {
    step "安装系统依赖 (Linux)"

    if [ "$(id -u)" -ne 0 ]; then
        fail "服务器模式请使用 root 用户运行: sudo bash deploy/setup.sh"
    fi

    export DEBIAN_FRONTEND=noninteractive
    case "${DISTRO}" in
        ubuntu|debian)
            apt-get update -qq
            apt-get install -y -qq git wget curl htop unzip sqlite3 ca-certificates tzdata
            if [ "${MODE}" = "server" ]; then
                apt-get install -y -qq ufw fail2ban
            fi
            ;;
        centos|rhel|fedora|rocky|almalinux)
            yum install -y git wget curl htop unzip sqlite ca-certificates
            if [ "${MODE}" = "server" ]; then
                yum install -y firewalld fail2ban
            fi
            ;;
        *)
            warn "未知 Linux 发行版: ${DISTRO}，尝试使用 apt..."
            apt-get update -qq && apt-get install -y -qq git wget curl sqlite3 ca-certificates
            ;;
    esac
    log "系统依赖安装完成"
}

install_deps() {
    if [ "${OS}" = "macos" ]; then
        install_deps_macos
    else
        install_deps_linux
    fi
}

# ===========================
# 系统初始化 (仅服务器模式)
# ===========================
init_system() {
    if [ "${MODE}" != "server" ]; then
        return
    fi

    step "系统初始化"

    # 设置时区
    timedatectl set-timezone Asia/Shanghai 2>/dev/null || true
    log "时区设为 Asia/Shanghai"

    # 创建应用用户
    if id "${APP_USER}" &>/dev/null; then
        log "用户 ${APP_USER} 已存在"
    else
        useradd -m -s /bin/bash "${APP_USER}"
        log "创建用户 ${APP_USER}"
    fi
}

# ===========================
# 安装 Go
# ===========================
install_go() {
    step "安装 Go ${GO_VERSION}"

    # 检查是否已安装正确版本
    if command -v go &>/dev/null; then
        CURRENT_GO=$(go version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "")
        if [ "${CURRENT_GO}" = "${GO_VERSION}" ]; then
            log "Go ${GO_VERSION} 已安装，跳过"
            return
        fi
    fi

    if [ "${OS}" = "macos" ]; then
        install_go_macos
    else
        install_go_linux
    fi
}

install_go_macos() {
    # macOS: 优先使用 goenv，其次 brew，最后直接下载
    if command -v goenv &>/dev/null; then
        goenv install "${GO_VERSION}" 2>/dev/null || true
        goenv local "${GO_VERSION}"
        log "Go $(go version) (via goenv)"
    elif brew list go &>/dev/null; then
        log "Go $(go version) (via Homebrew，版本可能不同)"
        warn "如需精确版本 ${GO_VERSION}，请使用 goenv"
    else
        # 直接下载
        GO_TAR="go${GO_VERSION}.darwin-${GOARCH}.tar.gz"
        curl -sSL "https://go.dev/dl/${GO_TAR}" -o "/tmp/${GO_TAR}"
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf "/tmp/${GO_TAR}"
        rm "/tmp/${GO_TAR}"
        export PATH="/usr/local/go/bin:${PATH}"

        # 写入 shell profile
        SHELL_RC="${HOME}/.zshrc"
        if ! grep -q '/usr/local/go/bin' "${SHELL_RC}" 2>/dev/null; then
            echo 'export PATH="/usr/local/go/bin:$PATH"' >> "${SHELL_RC}"
        fi
        log "Go $(go version) 安装完成"
    fi
}

install_go_linux() {
    GO_TAR="go${GO_VERSION}.linux-${GOARCH}.tar.gz"
    wget -q "https://go.dev/dl/${GO_TAR}" -O "/tmp/${GO_TAR}"
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "/tmp/${GO_TAR}"
    rm "/tmp/${GO_TAR}"

    # 配置全局环境
    cat > /etc/profile.d/golang.sh << 'GOEOF'
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
GOEOF
    chmod +x /etc/profile.d/golang.sh
    export PATH="/usr/local/go/bin:${PATH}"
    log "Go $(/usr/local/go/bin/go version) 安装完成"
}

# ===========================
# 获取代码 + 构建
# ===========================
build_app() {
    step "构建应用"

    if [ "${MODE}" = "server" ]; then
        REPO_URL="https://github.com/RobinCoderZhao/API-Change-Sentinel.git"
        mkdir -p "${APP_DIR}"
        if [ -d "${APP_DIR}/.git" ]; then
            cd "${APP_DIR}" && git pull -q
            log "代码已更新 (git pull)"
        else
            git clone -q "${REPO_URL}" "${APP_DIR}"
            log "代码克隆完成"
        fi
    fi

    cd "${APP_DIR}"
    mkdir -p "${DATA_DIR}" bin

    # 检测 go 路径
    GO_BIN=$(command -v go 2>/dev/null || echo "/usr/local/go/bin/go")

    ${GO_BIN} build -trimpath -ldflags="-s -w" -o bin/newsbot ./cmd/newsbot
    ${GO_BIN} build -trimpath -ldflags="-s -w" -o bin/devkit ./cmd/devkit
    ${GO_BIN} build -trimpath -ldflags="-s -w" -o bin/watchbot ./cmd/watchbot

    log "构建完成: newsbot=$(du -h bin/newsbot | cut -f1), devkit=$(du -h bin/devkit | cut -f1), watchbot=$(du -h bin/watchbot | cut -f1)"
}

# ===========================
# 初始化数据库
# ===========================
init_database() {
    step "初始化 SQLite 数据库"

    DB_PATH="${DATA_DIR}/newsbot.db"

    if [ -f "${DB_PATH}" ]; then
        log "数据库已存在: ${DB_PATH}"
        return
    fi

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
}

# ===========================
# 生成环境文件
# ===========================
create_env() {
    step "配置环境变量"

    if [ -f "${ENV_FILE}" ]; then
        warn "环境文件已存在: ${ENV_FILE}（跳过，请手动编辑）"
        return
    fi

    cat > "${ENV_FILE}" << 'ENVEOF'
# ====================================
# DevKit Suite 环境变量配置
# 修改后:
#   本地: source .env && ./bin/newsbot run
#   服务器: sudo systemctl restart newsbot watchbot
# ====================================

# LLM 配置（必填）
LLM_PROVIDER=openai
LLM_API_KEY=sk-your-api-key-here
LLM_MODEL=gpt-4o-mini

# Telegram 推送（可选，留空则输出到 stdout）
TELEGRAM_BOT_TOKEN=
TELEGRAM_CHANNEL_ID=

# NewsBot 数据库路径 (相对或绝对路径)
NEWSBOT_DB=data/newsbot.db
ENVEOF
    chmod 600 "${ENV_FILE}"
    log "环境文件已创建: ${ENV_FILE}"
    warn "⚠️  请编辑 ${ENV_FILE} 填入你的 API Key"
}

# ===========================
# macOS: 创建 launchd 服务
# ===========================
setup_launchd() {
    if [ "${MODE}" != "server" ] || [ "${OS}" != "macos" ]; then
        return
    fi

    step "配置 launchd 服务 (macOS)"

    PLIST_DIR="${HOME}/Library/LaunchAgents"
    mkdir -p "${PLIST_DIR}"

    # NewsBot
    cat > "${PLIST_DIR}/com.devkit.newsbot.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.devkit.newsbot</string>
    <key>ProgramArguments</key>
    <array>
        <string>${APP_DIR}/bin/newsbot</string>
        <string>serve</string>
    </array>
    <key>WorkingDirectory</key>
    <string>${APP_DIR}</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>LLM_API_KEY</key>
        <string>\${LLM_API_KEY}</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/newsbot.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/newsbot.err</string>
</dict>
</plist>
EOF
    log "launchd plist 已创建 (未加载，需手动: launchctl load ${PLIST_DIR}/com.devkit.newsbot.plist)"
}

# ===========================
# Linux: 配置 Systemd 服务
# ===========================
setup_systemd() {
    if [ "${MODE}" != "server" ] || [ "${OS}" != "linux" ]; then
        return
    fi

    step "配置 Systemd 服务"

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

    # 备份定时任务
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

    chown -R "${APP_USER}:${APP_USER}" "${APP_DIR}" "${DATA_DIR}" "${LOG_DIR}"
    systemctl daemon-reload
    systemctl enable newsbot watchbot devkit-backup.timer
    log "Systemd 服务配置完成"
}

# ===========================
# 安全加固 (仅 Linux server 模式)
# ===========================
setup_security() {
    if [ "${MODE}" != "server" ] || [ "${OS}" != "linux" ]; then
        return
    fi

    step "安全加固"

    # UFW 防火墙 (Debian/Ubuntu)
    if command -v ufw &>/dev/null; then
        ufw --force reset > /dev/null 2>&1
        ufw default deny incoming > /dev/null
        ufw default allow outgoing > /dev/null
        ufw allow 22/tcp > /dev/null
        ufw allow 8080/tcp > /dev/null
        ufw --force enable > /dev/null
        log "UFW 防火墙已配置 (SSH:22 + MCP:8080)"
    fi

    # fail2ban
    if command -v fail2ban-client &>/dev/null; then
        systemctl enable fail2ban > /dev/null 2>&1
        systemctl start fail2ban > /dev/null 2>&1
        log "fail2ban 已启用"
    fi
}

# ===========================
# 运行测试
# ===========================
run_tests() {
    step "运行测试"

    cd "${APP_DIR}"
    GO_BIN=$(command -v go 2>/dev/null || echo "/usr/local/go/bin/go")
    ${GO_BIN} test ./pkg/... -count=1 2>&1 | tail -20
    log "测试完成"
}

# ===========================
# 打印完成信息
# ===========================
print_done() {
    echo ""
    echo "╔══════════════════════════════════════════════════════╗"
    echo "║              ✅ 部署完成！                           ║"
    echo "╠══════════════════════════════════════════════════════╣"
    echo "║  系统: ${OS} (${ARCH}) / 模式: ${MODE}             "
    echo "║  应用: ${APP_DIR}                                   "
    echo "║  数据: ${DATA_DIR}                                  "
    echo "║  配置: ${ENV_FILE}                                  "
    echo "╚══════════════════════════════════════════════════════╝"
    echo ""

    if [ "${MODE}" = "local" ]; then
        echo "📋 本地使用方法:"
        echo ""
        echo "  # 1. 编辑环境变量"
        echo "  nano ${ENV_FILE}"
        echo ""
        echo "  # 2. 加载环境变量并运行"
        echo "  export \$(grep -v '^#' ${ENV_FILE} | xargs)"
        echo "  ./bin/newsbot run         # 运行一次新闻抓取"
        echo "  ./bin/watchbot check      # 运行一次竞品检查"
        echo "  ./bin/devkit commit       # AI 生成 commit message"
        echo "  ./bin/devkit review       # AI 代码审查"
        echo ""
    else
        echo "📋 下一步:"
        echo ""
        echo "  # 1. 编辑环境变量"
        echo "  nano ${ENV_FILE}"
        echo ""
        echo "  # 2. 启动服务"
        echo "  sudo systemctl start newsbot watchbot"
        echo ""
        echo "  # 3. 查看日志"
        echo "  sudo journalctl -u newsbot -f"
        echo ""
    fi
}

# ===========================
# 主流程
# ===========================
main() {
    detect_os
    parse_args "$@"

    echo ""
    echo "╔══════════════════════════════════════════════╗"
    echo "║   DevKit Suite 一键部署 v2.0                 ║"
    echo "║   系统: ${OS} (${ARCH})  模式: ${MODE}      "
    echo "╚══════════════════════════════════════════════╝"
    echo ""

    setup_paths
    install_deps
    init_system
    install_go
    build_app
    init_database
    create_env
    setup_systemd
    setup_launchd
    setup_security
    run_tests
    print_done
}

main "$@"
