# DevKit Suite — SaaS Intelligence Command Center

**AI-powered developer toolkit** — news digest, competitor monitoring, and benchmark tracking, now evolved into a fully-fledged SaaS web application.

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Next.js](https://img.shields.io/badge/Next.js-15-black?logo=next.js)](https://nextjs.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## 🚀 The Three Pillars of Intelligence

DevKit Suite has evolved from standalone CLI scripts into a centralized, beautifully designed web platform with multi-tenant capabilities, subscriptions, and an intuitive dashboard.

| Module | What it does | Status |
|---------|-------------|--------|
| **🔍 WatchBot** | Core product. Competitor website monitoring, HTML Diffing, and Gemini-powered change summaries. | ✅ Production SaaS |
| **📰 NewsBot** | Daily AI-curated tech news digest, translated into 6 languages. Displayed as an intelligence waterfall feed. | ✅ Production |
| **📊 Benchmark** | Live-scraped AI model benchmarks tracker. Generates auto-updating comparative visual reports. | ✅ Production |

---

## ✨ Key SaaS Features

1. **Frictionless Onboarding**: Sign up and select an industry to instantly auto-provision curated competitors to track.
2. **Global Battlecards**: A high-level, dashboard matrix visualizing all tracked competitors and their severity scores for the week.
3. **Smart Timeline & Diff Viewer**: Drill down into specific competitor changes. See the exact HTML/Text diffs side-by-side with an LLM-generated tactical breakdown.
4. **Smart Alerts**: Pro users can define custom alert rules (e.g., `Severity >= High` or `Contains "pricing"`).
5. **Stripe Integration**: Automated checkout sessions, subscription tier gatekeeping, and lifecycle webhooks.

---

## 🏗️ Architecture

The monorepo structure contains both the high-performance Go backend and the highly interactive Next.js frontend frontend.

```text
devkit-suite/
├── cmd/                # Go daemon entrypoints
│   ├── api/            # 🚀 New: REST API Server (port 8080)
│   ├── newsbot/        # News crawler job
│   ├── watchbot/       # Page crawler + diff analyzer job
│   └── devkit/         # Legacy Dev tools
├── frontend/           # 🚀 New: Next.js 15 App Router Frontend
│   ├── src/app/        # Pages (Dashboard, Onboarding, Pricing, Settings)
│   └── src/components/ # Shadcn UI and custom compoents
├── internal/           # Private business logic
│   ├── api/            # REST HTTP Handlers, Auth, Stripe Webhooks
│   ├── user/           # Multi-tenant user management
│   ├── watchbot/       # Pipeline, DB stores, Rules Engine
│   └── newsbot/        
├── pkg/                # Shared internal libraries (LLM, Notifications, Scraper)
├── docs/               # Technical and Product Documentation
└── deploy/             # One-click shell deployment scripts
```

## ⚡ Quick Start (Local Development)

### 1. Backend (Go API & Workers)

```bash
git clone https://github.com/RobinCoderZhao/devkit-suite.git
cd devkit-suite

# Set up configuration
cp .env.example .env
# Edit .env with your LLM_API_KEY, STRIPE_SECRET_KEY, etc.

# Run the API server
go run ./cmd/api

# In a separate terminal, run the workers manually if needed
go run ./cmd/watchbot check
```

### 2. Frontend (Next.js)

```bash
cd frontend
npm install
npm run dev
```

Navigate to `http://localhost:3000` to interact with the SaaS platform.

---

## 🚀 One-Click Production Deploy

Deploy the entire suite (API, Frontend, Workers, Systemd Services, SQLite Databases) securely to a Linux VPS (e.g., Aliyun ECS).

```bash
git clone https://github.com/RobinCoderZhao/devkit-suite.git /tmp/devkit
bash /tmp/devkit/deploy/setup.sh    # Installs Go, Node, Nginx, configures systemd
```

> **Note**: For comprehensive instructions on operating the application, refer to the [User Manual](docs/user_manual.md) (Chinese).

---

## 📄 License & Contributing

[MIT](LICENSE) — free for personal and commercial use. Issues and PRs are welcome!

Built with ❤️ by [RobinCoderZhao](https://github.com/RobinCoderZhao)
