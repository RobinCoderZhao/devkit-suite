# LLM Pricing Reference

> Last updated: 2026-02-20

## Pricing Table (USD per 1M tokens)

| Provider | Model | Input | Output | Notes |
|----------|-------|-------|--------|-------|
| **OpenAI** | GPT-5.2 | $1.75 | $14.00 | Flagship |
| | GPT-5.2 Pro | $21.00 | $168.00 | High reasoning |
| | GPT-5 mini | $0.25 | $2.00 | Cost efficient |
| | GPT-4.1 mini | $0.80 | $3.20 | Stable |
| | GPT-4.1 nano | $0.20 | $0.80 | Ultra low cost |
| **Anthropic** | Claude Haiku 4.5 | $1.00 | $5.00 | Lightweight |
| | Claude Sonnet 4.5 | $3.00 | $15.00 | Mainstream |
| | Claude Opus 4.5 | $5.00 | $25.00 | High quality |
| **Google** | Gemini 2.0 Flash | $0.10 | $0.40 | Standard tier |
| | Gemini 2.5 Flash | $0.30 | $2.50 | Standard tier |
| | Gemini 2.5 Flash Lite | $0.10 | $0.40 | Low cost |
| | Gemini 3 Pro (≤200k) | $2.00 | $12.00 | Short context |
| | Gemini 3 Pro (>200k) | $4.00 | $18.00 | Long context |
| | Gemini 2.5 Pro (≤200k) | $1.25 | $10.00 | Standard |
| **Alibaba Cloud** | qwen-turbo (Intl) | $0.05 | $0.20 | Lowest cost tier |
| | qwen-max (Intl) | $1.60 | $6.40 | High tier |
| | qwq-plus (Intl) | $0.80 | $2.40 | Mid tier |
| **MiniMax** | M2.5 | $0.29 | $1.17 | Converted from ¥2.1/¥8.4 |
| | M2.5-highspeed | $0.58 | $2.33 | Converted from ¥4.2/¥16.8 |

## DevKit Suite Recommended Models

| Use Case | Recommended | Cost/digest | Why |
|----------|-------------|-------------|-----|
| NewsBot daily digest | Gemini 2.5 Flash Lite | ~$0.001 | Cheapest, fast enough |
| WatchBot change analysis | Gemini 2.5 Flash | ~$0.003 | Better reasoning |
| Translation | Gemini 2.0 Flash | ~$0.001 | Translation is easy |
| Content writing (Pro) | Gemini 3 Pro | ~$0.02 | High quality output |
| Budget option | qwen-turbo | ~$0.0005 | Absolute lowest cost |
| China-optimized | MiniMax M2.5 | ~$0.002 | Good Chinese support |
