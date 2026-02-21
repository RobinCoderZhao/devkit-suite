# DevKit Suite O&M Manual (运维手册)

本手册是 DevKit Suite 的官方运维部署及排障指南，涵盖所有的架构模块、启动配置、服务命令、以及多语言（i18n）云反向代理环境。

---

## 1. 架构拓扑 (Architecture)

DevKit Suite 由 1个前端引擎、1个后端接口层、和 2 个核心定时任务系统构成：

1. `devkit-frontend` (Next.js 15)
   - 监听端口: `3000`
   - 提供页面、i18n 多语言路由 (`/[locale]`)、认证拦截等 UI 服务。

2. `devkit-api` (Go REST API)
   - 监听端口: `8080`
   - 提供 Stripe 计费钩子、WatchBot / NewsBot JSON API、数据拉取接口。

3. `watchbot` (Go Worker)
   - 定时触发的无头监控爬虫，对比竞品变化、调用大语言模型进行差异摘要 (Diff Analysis) 并触发 Smart Alerts。

4. `newsbot` (Go Script)
   - 托管在 Systemd Timer 中，每天自动爬取 HackerNews 和 RSS 流以生成 AI 科技日报。

5. `Nginx` (Reverse Proxy)
   - 监听端口: `80 / 443`
   - 将外网顶级域名 (`devkit-suite.com`) 的真实流量根据前缀 (`/api`) 拆分给 `8080` (Go) 和 `3000` (Next.js)。

---

## 2. 部署环境及安装路径

**所有服务及配置文件默认安装在云服务器 (如阿里云 ECS) 的以下位置：**

- **项目根目录**: `/opt/devkit-suite`
- **前端目录**: `/opt/devkit-suite/frontend`
- **数据库路径 (SQLite)**:
  - WatchBot 数据: `/opt/devkit-suite/data/watchbot.db`
  - NewsBot 数据: `/opt/devkit-suite/data/newsbot.db`
- **环境配置文件**: `/opt/devkit-suite/.env` *(存储 API Keys, 数据库密钥等)*
- **部署脚本目录**: `/opt/devkit-suite/deploy/`

### Systemd 后台服务单元 (Service Units)

所有的长驻服务都托管在 Linux systemd 中监控：

- `/etc/systemd/system/devkit-frontend.service`
- `/etc/systemd/system/devkit-api.service`
- `/etc/systemd/system/watchbot.service`
- `/etc/systemd/system/newsbot.timer` 和 `newsbot.service`

### Nginx 配置文件

- `/etc/nginx/conf.d/devkit-suite.nginx.conf`

---

## 3. 服务器日常管理命令清单

### 3.1 服务启停与查看 (Systemctl)

无论想重启哪个模块，均使用 `systemctl` 命令完成。

| 模块名称 | 重启命令 | 状态查询 |
|---------|---------|----------|
| **Next.js 前端** | `sudo systemctl restart devkit-frontend` | `sudo systemctl status devkit-frontend` |
| **Go REST API** | `sudo systemctl restart devkit-api` | `sudo systemctl status devkit-api` |
| **WatchBot 引擎** | `sudo systemctl restart watchbot` | `sudo systemctl status watchbot` |
| **Nginx 代理** | `sudo systemctl restart nginx` | `sudo systemctl status nginx` |

*提示: 如果需要强制关闭某个服务，请使用 `stop`，如 `sudo systemctl stop watchbot`。*

### 3.2 查看实时运行日志 (Journalctl)

如果出现 500 Server Error 等内部错误，日志是排错的第一手段。

- **查看前台 UI 报错**:
  `sudo journalctl -u devkit-frontend -n 100 -f`
- **查看后端 API 调用/鉴权报错**:
  `sudo journalctl -u devkit-api -n 100 -f`
- **查看 WatchBot 运行爬虫或大模型失败日志**:
  `sudo journalctl -u watchbot -n 200 -f`

*(* `-n` *代表结尾行数，* `-f` *代表跟随追加模式。按 `Ctrl+C` 退出)*

---

## 4. 升级方案 (Upgrades & CI/CD)

### 4.1 常规更新流程

当本地完成了 BugFix 或 Feature，并已经成功推送（push）到自己的 GitHub 仓库后，**登录服务器执行下述自动部署脚本**即可完成 0-downtime（平滑）更新。

```bash
# 进入云服务器的项目目录
cd /opt/devkit-suite/

# 一步触发全自动升级脚本
sudo /bin/bash deploy/upgrade.sh
```

**`upgrade.sh` 的内部工作管线**:

1. 执行 `git pull` 拉取 main 分支更新。
2. 调用 `/usr/local/go/bin/go build` 重构建三个 Go 可执行二进制文件。
3. 如果 `frontend` 发生变动（此目录已从 git ignore 但如果通过 rsync 增量过去会触发），则进入 frontend 执行 `npm install && npm run build`。
4. 调用 `systemctl restart` 重启四大核心服务以挂载最新的二进制内存。

### 4.2 UI 快速热修补 (Rsync 同步)

如果你不想使用 Git 作为中间件，而是需要将本地修改的前端代码**直接热发至外网**，可以使用如下的 rsync 命令：

```bash
# 在你自己的 Mac / PC 本地终端执行
sshpass -p '你的密码' rsync -avz -e ssh --exclude '.git' --exclude 'frontend/.next' --exclude 'frontend/node_modules' /Users/当前目录路径/API-Change-Sentinel/ root@43.98.84.1:/opt/devkit-suite/

# 登录服务器后重启 Next.js 重新打包
ssh root@43.98.84.1
cd /opt/devkit-suite/frontend
npm run build
systemctl restart devkit-frontend
```

---

## 5. 常见进阶排障 (Troubleshooting)

### Q: 发现域名访问出现 Cloudflare 522

A: 522 错误代表源站超时。检查服务器的 Nginx 服务是否存活：

```bash
sudo systemctl status nginx
```

如果挂掉，检查 `/var/log/nginx/error.log` 或执行 `nginx -t` 检测配置文件语法。同时检查服务器云盾/安全组，确认公网 **80 和 443** 端口是对外放行的。

### Q: Next.js 页面 500 或者 `ECONNREFUSED`

A: 这意味着 Node Server 崩溃。

1. `journalctl -u devkit-frontend -n 50` 查看错误栈。
2. 可能是 Nginx 没有将 `proxy_pass http://127.0.0.1:3000/` 正确转发，或是 Next.js 服务卡在了 `3001` 备用端口。通过 `lsof -ti :3000` 查看端口占用。

### Q: WatchBot / 商业监控告警不触发了

A: WatchBot 大部分时间静默运行。遇到不触发问题，请检查：

1. `journalctl -u watchbot | grep ERROR`。
2. 确认 `store/data/watchbot.db` 权限属于当前启动的用户 (`root` 或 `deploy`)。
3. 检查环境变量文件（`.env`）中 OpenAI / Anthropic 的 Key 余额是否耗尽，导致大模型对比摘要生成失败被拦截。
