package watchbot

import (
	"fmt"
	"strings"

	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/notify"
)

// ComposeDigest creates a single aggregated notification for a subscriber.
// Merges all changes from one round into one message.
func ComposeDigest(changes []Change, subscriber SubscriberWithCompetitors) notify.Message {
	if len(changes) == 0 {
		return notify.Message{}
	}

	var body strings.Builder
	var htmlBody strings.Builder

	// Plain text
	body.WriteString(fmt.Sprintf("检测到 %d 个页面发生变化：\n\n", len(changes)))

	// HTML
	htmlBody.WriteString(`<!DOCTYPE html><html><body style="font-family:Arial,sans-serif;background:#1a1a2e;color:#e0e0e0;padding:20px;">`)
	htmlBody.WriteString(fmt.Sprintf(`<h2 style="color:#eee;">🔔 竞品监控报告</h2>`))
	htmlBody.WriteString(fmt.Sprintf(`<p style="color:#aaa;">检测到 %d 个页面发生变化</p><hr style="border-color:#333;">`, len(changes)))

	for _, c := range changes {
		emoji := severityEmoji(c.Severity)
		label := severityLabel(c.Severity)

		// Plain text
		body.WriteString(fmt.Sprintf("%s [%s] %s — %s\n", emoji, label, c.CompetitorName, c.PageType))
		if c.Analysis != "" {
			// Truncate analysis to 200 chars for notification
			analysis := c.Analysis
			if len(analysis) > 200 {
				analysis = analysis[:200] + "..."
			}
			body.WriteString(analysis + "\n")
		}
		body.WriteString(fmt.Sprintf("📊 +%d / -%d 行 · 🔗 %s\n\n", c.Additions, c.Deletions, c.PageURL))

		// HTML
		bgColor := severityBgColor(c.Severity)
		badgeColor := severityBadgeColor(c.Severity)
		htmlBody.WriteString(fmt.Sprintf(`<div style="background:%s;border-radius:8px;padding:16px;margin:12px 0;">`, bgColor))
		htmlBody.WriteString(fmt.Sprintf(`<span style="background:%s;color:#fff;padding:2px 8px;border-radius:4px;font-size:12px;">%s %s</span>`, badgeColor, emoji, label))
		htmlBody.WriteString(fmt.Sprintf(`<h3 style="color:#eee;margin:8px 0 4px;">%s — %s</h3>`, c.CompetitorName, c.PageType))
		if c.Analysis != "" {
			analysis := c.Analysis
			if len(analysis) > 300 {
				analysis = analysis[:300] + "..."
			}
			htmlBody.WriteString(fmt.Sprintf(`<p style="color:#ccc;font-size:14px;">%s</p>`, analysis))
		}
		htmlBody.WriteString(fmt.Sprintf(`<p style="color:#888;font-size:12px;">📊 +%d / -%d 行 · <a href="%s" style="color:#64b5f6;">查看原页面 →</a></p>`,
			c.Additions, c.Deletions, c.PageURL))
		htmlBody.WriteString(`</div>`)
	}

	// Find unchanged competitors
	changedCompIDs := make(map[int]bool)
	for _, c := range changes {
		// Find competitor ID from the change's page
		for i, name := range subscriber.CompetitorNames {
			if name == c.CompetitorName && i < len(subscriber.CompetitorIDs) {
				changedCompIDs[subscriber.CompetitorIDs[i]] = true
			}
		}
	}
	var unchanged []string
	for i, name := range subscriber.CompetitorNames {
		if i < len(subscriber.CompetitorIDs) && !changedCompIDs[subscriber.CompetitorIDs[i]] {
			unchanged = append(unchanged, name)
		}
	}
	if len(unchanged) > 0 {
		body.WriteString("---\n未变化：" + strings.Join(unchanged, ", ") + " ✅\n")
		htmlBody.WriteString(fmt.Sprintf(`<hr style="border-color:#333;"><p style="color:#666;">未变化：%s ✅</p>`, strings.Join(unchanged, ", ")))
	}

	htmlBody.WriteString(`</body></html>`)

	return notify.Message{
		Title:    fmt.Sprintf("🔔 竞品监控报告 — 检测到 %d 个变化", len(changes)),
		Body:     body.String(),
		HTMLBody: htmlBody.String(),
		Format:   "html",
	}
}

func severityEmoji(s string) string {
	switch s {
	case "critical":
		return "🔴"
	case "important":
		return "🟡"
	case "minor":
		return "🟢"
	default:
		return "⚪"
	}
}

func severityLabel(s string) string {
	switch s {
	case "critical":
		return "Critical"
	case "important":
		return "Important"
	case "minor":
		return "Minor"
	default:
		return s
	}
}

func severityBgColor(s string) string {
	switch s {
	case "critical":
		return "#2d1b1b"
	case "important":
		return "#2d2a1b"
	case "minor":
		return "#1b2d1b"
	default:
		return "#1a1a2e"
	}
}

func severityBadgeColor(s string) string {
	switch s {
	case "critical":
		return "#e53935"
	case "important":
		return "#ff9800"
	case "minor":
		return "#4caf50"
	default:
		return "#607d8b"
	}
}
