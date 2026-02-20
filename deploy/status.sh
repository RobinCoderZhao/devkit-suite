#!/bin/bash
#
# DevKit Suite — 状态检查脚本
#
set -euo pipefail

APP_DIR="/opt/devkit-suite"
DATA_DIR="${APP_DIR}/data"

echo "╔══════════════════════════════════════╗"
echo "║   DevKit Suite 状态检查              ║"
echo "╚══════════════════════════════════════╝"
echo ""

# 服务状态
echo "📦 服务状态:"
echo "---"
for svc in newsbot watchbot devkit-api devkit-frontend; do
    status=$(systemctl is-active ${svc} 2>/dev/null || echo "inactive")
    if [ "${status}" = "active" ]; then
        echo "  ✅ ${svc}: 运行中"
    else
        echo "  ❌ ${svc}: ${status}"
    fi
done
echo ""

# 系统资源
echo "💻 系统资源:"
echo "---"
echo "  CPU: $(grep -c ^processor /proc/cpuinfo) 核"
echo "  内存: $(free -h | awk '/Mem:/{printf "%s / %s (%.1f%%)", $3, $2, $3/$2*100}')"
echo "  磁盘: $(df -h / | awk 'NR==2{printf "%s / %s (%s)", $3, $2, $5}')"
echo ""

# 数据库状态
if [ -f "${DATA_DIR}/newsbot.db" ]; then
    echo "🗄️  数据库:"
    echo "---"
    echo "  文件大小: $(du -h ${DATA_DIR}/newsbot.db | cut -f1)"
    echo "  文章数量: $(sqlite3 ${DATA_DIR}/newsbot.db 'SELECT COUNT(*) FROM articles;' 2>/dev/null || echo 'N/A')"
    echo "  日报数量: $(sqlite3 ${DATA_DIR}/newsbot.db 'SELECT COUNT(*) FROM digests;' 2>/dev/null || echo 'N/A')"
    LATEST=$(sqlite3 ${DATA_DIR}/newsbot.db 'SELECT date FROM digests ORDER BY created_at DESC LIMIT 1;' 2>/dev/null || echo 'N/A')
    echo "  最新日报: ${LATEST}"
    echo ""
fi

# 备份状态
BACKUP_DIR="${DATA_DIR}/backups"
if [ -d "${BACKUP_DIR}" ]; then
    BACKUP_COUNT=$(ls -1 ${BACKUP_DIR}/*.db 2>/dev/null | wc -l)
    LATEST_BACKUP=$(ls -1t ${BACKUP_DIR}/*.db 2>/dev/null | head -1)
    echo "💾 备份:"
    echo "---"
    echo "  备份数量: ${BACKUP_COUNT}"
    echo "  最新备份: $(basename ${LATEST_BACKUP:-N/A} 2>/dev/null)"
    echo ""
fi

# 最近日志
echo "📋 最近日志 (newsbot):"
echo "---"
journalctl -u newsbot --no-pager -n 5 --output=short-iso 2>/dev/null || echo "  无日志"
echo ""
echo "📋 最近日志 (watchbot):"
echo "---"
journalctl -u watchbot --no-pager -n 5 --output=short-iso 2>/dev/null || echo "  无日志"
