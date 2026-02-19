#!/bin/bash
#
# DevKit Suite — 升级脚本
# 拉取最新代码、重新构建、重启服务
#
set -euo pipefail

APP_DIR="/opt/devkit-suite"

echo "🔄 拉取最新代码..."
cd "${APP_DIR}"
git pull

echo "🔨 重新构建..."
/usr/local/go/bin/go build -trimpath -ldflags="-s -w" -o bin/newsbot ./cmd/newsbot
/usr/local/go/bin/go build -trimpath -ldflags="-s -w" -o bin/devkit ./cmd/devkit
/usr/local/go/bin/go build -trimpath -ldflags="-s -w" -o bin/watchbot ./cmd/watchbot

echo "♻️  重启服务..."
sudo systemctl restart newsbot watchbot

echo "✅ 升级完成！"
sudo systemctl status newsbot watchbot --no-pager
