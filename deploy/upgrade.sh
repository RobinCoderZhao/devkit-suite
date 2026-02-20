#!/bin/bash
#
# DevKit Suite — 升级脚本
# 拉取最新代码、重新构建、重启服务
#
set -euo pipefail

APP_DIR="/opt/devkit-suite"

echo "🔄 使用 rsync 同步了最新代码 (跳过 git pull)..."
cd "${APP_DIR}"
# git pull

echo "🔨 重新构建 Go Services..."
/usr/local/go/bin/go build -trimpath -ldflags="-s -w" -o bin/newsbot ./cmd/newsbot
/usr/local/go/bin/go build -trimpath -ldflags="-s -w" -o bin/devkit ./cmd/devkit
/usr/local/go/bin/go build -trimpath -ldflags="-s -w" -o bin/watchbot ./cmd/watchbot
/usr/local/go/bin/go build -trimpath -ldflags="-s -w" -o bin/api ./cmd/api

echo "📦 构建 Frontend (Next.js)..."
if command -v npm &> /dev/null; then
  cd "${APP_DIR}/frontend"
  npm install
  npm run build
  cd "${APP_DIR}"
else
  echo "⚠️ npm is not installed, skipping frontend build."
fi

echo "♻️ 重启服务..."
sudo systemctl restart newsbot watchbot devkit-api devkit-frontend || true

echo "✅ 升级完成！"
sudo systemctl status newsbot watchbot devkit-api devkit-frontend --no-pager || true
