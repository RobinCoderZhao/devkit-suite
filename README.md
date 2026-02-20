# DevKit Suite

**AI-powered developer toolkit** — news digest, competitor monitoring, and benchmark tracking in a single Go binary.

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GitHub Stars](https://img.shields.io/github/stars/RobinCoderZhao/devkit-suite?style=social)](https://github.com/RobinCoderZhao/devkit-suite)

---

## 🚀 Three Products, One Repo

| Product | What it does | Status |
|---------|-------------|--------|
| **📰 NewsBot** | AI-curated tech news digest → multi-language email | ✅ Production |
| **🔍 WatchBot** | Competitor website monitoring + AI change analysis | ✅ Production |
| **📊 Benchmark Tracker** | Live AI model benchmark scraping + comparison report | ✅ Production |

### How They Work Together

```
NewsBot (free)  ──→  Build email list  ──→  WatchBot (SaaS)
                                              ↑
Benchmark Tracker (free)  ──→  SEO traffic ──┘
```

---

## ⚡ Quick Start

### 1. Clone & Build

```bash
git clone https://github.com/RobinCoderZhao/devkit-suite.git
cd devkit-suite
go build -o bin/newsbot ./cmd/newsbot
go build -o bin/watchbot ./cmd/watchbot
```

### 2. Configure

```bash
cp .env.example .env
# Edit .env with your API keys:
#   LLM_PROVIDER=minimax       (or openai, gemini, claude)
#   LLM_API_KEY=sk-xxx
#   SMTP_HOST=smtp.gmail.com
#   SMTP_FROM=you@gmail.com
#   SMTP_PASSWORD=xxxx
source .env
```

### 3. Run

```bash
# 📰 Subscribe to daily AI news
./bin/newsbot subscribe --email=you@email.com --lang=en
./bin/newsbot run

# 🔍 Monitor a competitor
./bin/watchbot add https://competitor.com/pricing
./bin/watchbot check

# 📊 Generate benchmark report
./bin/watchbot benchmark --scrape=live --email=you@email.com
```

---

## 📰 NewsBot — AI News Digest

Aggregates from **28 sources** across 5 categories, analyzes with LLM, translates to 6 languages, delivers via email.

**Sources include:** HackerNews, TechCrunch, Wired, VentureBeat, Reddit ML, Anthropic Blog, 机器之心, 量子位, and more.

**Key Features:**

- 🔄 Smart deduplication — only new articles are analyzed (saves tokens)
- 🌍 Auto-language detection via IP geolocation
- 📧 Beautiful HTML email newsletters
- 💰 Cost: ~$0.01 per digest (MiniMax M2.5)

```bash
./bin/newsbot subscribe --email=team@company.com --name=Team --lang=zh
./bin/newsbot run    # Fetch → Analyze → Translate → Email
```

---

## 🔍 WatchBot — Competitor Monitor

Tracks competitor websites for changes, uses LLM to explain what changed and why it matters.

**Key Features:**

- 🕸️ Auto-discovers key pages (/pricing, /features, /blog)
- 📊 HTML diff + LLM analysis ("price dropped 20%")
- 📧 Alert emails with change summary
- ⏰ Scheduled monitoring (every 6h in `serve` mode)

```bash
./bin/watchbot add https://vercel.com/pricing
./bin/watchbot add https://competitor.com/features
./bin/watchbot check                                # One-time check
./bin/watchbot serve                                # Continuous monitoring
```

---

## 📊 Benchmark Tracker

Live-scrapes AI model benchmarks from [llm-stats.com](https://llm-stats.com), generates professional comparison reports.

**Key Features:**

- 🔴 Highlights top scores per benchmark
- 📊 16 benchmarks × 8+ models (Gemini, GPT, Claude, etc.)
- 🖼️ PNG output for social sharing
- 📧 HTML email delivery
- 🔄 Auto decimal→percentage conversion

```bash
./bin/watchbot benchmark --scrape=live --output=png --file=report.png
./bin/watchbot benchmark --scrape=live --email=you@email.com
```

---

## 🏗️ Architecture

```
devkit-suite/
├── cmd/
│   ├── newsbot/        # AI news digest CLI
│   ├── watchbot/       # Competitor monitor + benchmark CLI
│   └── devkit/         # Developer tools CLI
├── pkg/                # Shared libraries (importable)
│   ├── llm/            # Multi-model LLM client (OpenAI/MiniMax/Gemini)
│   ├── scraper/        # Web scraper with Jina Reader
│   ├── notify/         # Email/Telegram/Webhook notifications
│   ├── i18n/           # 6-language i18n + IP geolocation
│   ├── benchmarks/     # Benchmark tracker + image renderer
│   │   └── parsers/    # llm-stats.com table parsers
│   ├── differ/         # Text diff engine
│   └── storage/        # Storage abstraction
├── internal/           # Private business logic
├── deploy/             # One-click deployment scripts
├── docs/               # Product & architecture docs
└── .env                # Configuration
```

## 🔧 Supported LLM Providers

| Provider | Models | Cost |
|----------|--------|------|
| **MiniMax** | M2.5 | ~$0.01/call ⭐ Cheapest |
| **OpenAI** | GPT-4o, GPT-4o-mini | $0.01-0.03/call |
| **Google** | Gemini 2.5 Pro | Varies |
| **Anthropic** | Claude 3.7 Sonnet | Varies |

---

## 🚀 One-Click Deploy (Aliyun Singapore ECS)

```bash
ssh root@<your-server>
git clone https://github.com/RobinCoderZhao/devkit-suite.git /tmp/devkit
bash /tmp/devkit/deploy/setup.sh    # ~2-3 minutes
nano /opt/devkit-suite/.env         # Add API keys
sudo systemctl start newsbot watchbot
```

> **Why Singapore?** OpenAI/Gemini APIs don't support Hong Kong. Singapore has direct access to all major LLM APIs. See [deployment guide](docs/aliyun_sg_deployment.md).

---

## 📄 License

[MIT](LICENSE) — free for personal and commercial use.

## 🤝 Contributing

Issues and PRs are welcome! See the [docs/](docs/) directory for architecture and development plans.

---

**Built with ❤️ by [RobinCoderZhao](https://github.com/RobinCoderZhao)**
