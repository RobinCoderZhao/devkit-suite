// Package publisher formats and distributes the daily digest.
package publisher

import (
	"context"
	"fmt"
	"strings"

	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/analyzer"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/notify"
)

// Publisher formats digests and sends them via notification channels.
type Publisher struct {
	dispatcher *notify.Dispatcher
	channels   []notify.Channel
}

// NewPublisher creates a new publisher with the given dispatcher and target channels.
func NewPublisher(dispatcher *notify.Dispatcher, channels []notify.Channel) *Publisher {
	return &Publisher{
		dispatcher: dispatcher,
		channels:   channels,
	}
}

// Publish formats a DailyDigest and sends it via configured channels.
func (p *Publisher) Publish(ctx context.Context, digest *analyzer.DailyDigest) error {
	msg := notify.Message{
		Title:  fmt.Sprintf("🤖 AI 热点日报 — %s", digest.Date),
		Body:   FormatDigest(digest),
		Format: "markdown",
	}

	return p.dispatcher.Dispatch(ctx, p.channels, msg)
}

// FormatDigest converts a DailyDigest into a Markdown-formatted message.
func FormatDigest(digest *analyzer.DailyDigest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# 🤖 AI 热点日报 — %s\n\n", digest.Date))

	if digest.Summary != "" {
		sb.WriteString(fmt.Sprintf("📝 **今日概览**\n%s\n\n", digest.Summary))
	}

	sb.WriteString("---\n\n")

	for i, h := range digest.Headlines {
		emoji := importanceEmoji(h.Importance)
		sb.WriteString(fmt.Sprintf("%s **%d. %s**\n", emoji, i+1, h.Title))
		if h.Summary != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", h.Summary))
		}
		if h.URL != "" {
			sb.WriteString(fmt.Sprintf("   🔗 [原文](%s) | 来源: %s\n", h.URL, h.Source))
		}
		if len(h.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("   🏷 %s\n", strings.Join(h.Tags, ", ")))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("*由 NewsBot 自动生成 | 消耗 Token: %d | 成本: $%.4f*\n",
		digest.TokensUsed, digest.Cost))

	return sb.String()
}

func importanceEmoji(importance string) string {
	switch importance {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}
