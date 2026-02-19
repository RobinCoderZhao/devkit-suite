// Package publisher formats and distributes the daily digest.
package publisher

import (
	"context"
	"fmt"
	"html"
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
		Title:    fmt.Sprintf("🤖 AI 热点日报 — %s", digest.Date),
		Body:     FormatDigest(digest),
		HTMLBody: FormatDigestHTML(digest),
		Format:   "markdown",
	}

	return p.dispatcher.Dispatch(ctx, p.channels, msg)
}

// FormatDigest converts a DailyDigest into a Markdown-formatted message (for Telegram/stdout).
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

// FormatDigestHTML generates a professional HTML newsletter from DailyDigest.
func FormatDigestHTML(digest *analyzer.DailyDigest) string {
	var sb strings.Builder

	// Email wrapper
	sb.WriteString(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"></head>
<body style="margin:0;padding:0;background-color:#0f0f23;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#0f0f23;">
<tr><td align="center" style="padding:20px 10px;">
<table role="presentation" width="640" cellpadding="0" cellspacing="0" style="max-width:640px;width:100%;">
`)

	// Header
	sb.WriteString(fmt.Sprintf(`
<!-- Header -->
<tr><td style="background:linear-gradient(135deg,#667eea 0%%,#764ba2 100%%);border-radius:16px 16px 0 0;padding:32px 40px;text-align:center;">
  <h1 style="margin:0;font-size:28px;font-weight:800;color:#ffffff;letter-spacing:-0.5px;">🤖 AI 热点日报</h1>
  <p style="margin:8px 0 0;font-size:15px;color:rgba(255,255,255,0.85);font-weight:500;">%s</p>
</td></tr>
`, digest.Date))

	// Summary section
	if digest.Summary != "" {
		summaryHTML := formatSummaryLines(digest.Summary)
		sb.WriteString(fmt.Sprintf(`
<!-- Summary -->
<tr><td style="background-color:#1a1a2e;padding:28px 40px;border-bottom:1px solid rgba(255,255,255,0.06);">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
    <tr>
      <td style="width:4px;background:linear-gradient(180deg,#667eea,#764ba2);border-radius:2px;"></td>
      <td style="padding-left:16px;">
        <p style="margin:0 0 12px;font-size:11px;font-weight:700;text-transform:uppercase;letter-spacing:1.5px;color:#667eea;">今日概览</p>
        %s
      </td>
    </tr>
  </table>
</td></tr>
`, summaryHTML))
	}

	// Headlines
	for i, h := range digest.Headlines {
		badge := importanceBadge(h.Importance)
		bgColor := "#1a1a2e"
		if i%2 == 1 {
			bgColor = "#16162a"
		}

		// Tags HTML
		tagsHTML := ""
		if len(h.Tags) > 0 {
			var tagParts []string
			for _, tag := range h.Tags {
				tagParts = append(tagParts,
					fmt.Sprintf(`<span style="display:inline-block;background:rgba(102,126,234,0.15);color:#8b9cf7;font-size:11px;padding:2px 8px;border-radius:10px;margin:2px 4px 2px 0;">%s</span>`,
						html.EscapeString(tag)))
			}
			tagsHTML = strings.Join(tagParts, "")
		}

		// Source + link
		linkHTML := ""
		if h.URL != "" {
			linkHTML = fmt.Sprintf(`<a href="%s" style="color:#667eea;font-size:12px;text-decoration:none;font-weight:500;">阅读原文 →</a>`,
				html.EscapeString(h.URL))
		}

		sb.WriteString(fmt.Sprintf(`
<!-- Article %d -->
<tr><td style="background-color:%s;padding:24px 40px;border-bottom:1px solid rgba(255,255,255,0.04);">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
    <tr>
      <td style="vertical-align:top;width:36px;padding-top:2px;">
        <span style="display:inline-block;width:28px;height:28px;line-height:28px;text-align:center;background:rgba(102,126,234,0.12);border-radius:8px;font-size:13px;font-weight:700;color:#8b9cf7;">%d</span>
      </td>
      <td style="padding-left:12px;">
        <table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
          <tr><td>
            %s
            <span style="font-size:16px;font-weight:700;color:#f0f0f0;line-height:1.4;">%s</span>
          </td></tr>
          <tr><td style="padding-top:8px;">
            <p style="margin:0;font-size:14px;line-height:1.6;color:#a0a0b8;">%s</p>
          </td></tr>
          <tr><td style="padding-top:10px;">
            <table role="presentation" cellpadding="0" cellspacing="0"><tr>
              <td style="padding-right:12px;">%s</td>
              <td><span style="font-size:11px;color:#606080;">%s</span></td>
            </tr></table>
          </td></tr>
          <tr><td style="padding-top:6px;">%s</td></tr>
        </table>
      </td>
    </tr>
  </table>
</td></tr>
`, i+1, bgColor, i+1,
			badge, html.EscapeString(h.Title),
			html.EscapeString(h.Summary),
			linkHTML, html.EscapeString(h.Source),
			tagsHTML))
	}

	// Footer
	sb.WriteString(fmt.Sprintf(`
<!-- Footer -->
<tr><td style="background-color:#12121f;border-radius:0 0 16px 16px;padding:24px 40px;text-align:center;">
  <p style="margin:0;font-size:12px;color:#505070;line-height:1.6;">
    由 <strong style="color:#667eea;">DevKit NewsBot</strong> 自动生成<br>
    Token: %d · 成本: $%.4f · Powered by MiniMax M2.5
  </p>
</td></tr>
`, digest.TokensUsed, digest.Cost))

	// Close wrapper
	sb.WriteString(`
</table>
</td></tr>
</table>
</body>
</html>`)

	return sb.String()
}

// formatSummaryLines splits a Chinese summary into individual sentences for readable display.
func formatSummaryLines(summary string) string {
	// Split by Chinese period, then filter empty items
	parts := strings.Split(summary, "。")
	var lines []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		lines = append(lines, html.EscapeString(p))
	}
	if len(lines) == 0 {
		return fmt.Sprintf(`<p style="margin:0;font-size:15px;line-height:1.8;color:#e0e0e0;">%s</p>`, html.EscapeString(summary))
	}

	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 6px;font-size:14px;line-height:1.7;color:#e0e0e0;">
          <span style="color:#667eea;margin-right:6px;">▸</span>%s</p>`, line))
	}
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

func importanceBadge(importance string) string {
	switch importance {
	case "high":
		return `<span style="display:inline-block;background:#ff4757;color:#fff;font-size:10px;font-weight:700;padding:1px 6px;border-radius:3px;margin-right:6px;vertical-align:middle;text-transform:uppercase;">重要</span>`
	case "medium":
		return `<span style="display:inline-block;background:#ffa502;color:#fff;font-size:10px;font-weight:700;padding:1px 6px;border-radius:3px;margin-right:6px;vertical-align:middle;text-transform:uppercase;">关注</span>`
	case "low":
		return `<span style="display:inline-block;background:#2ed573;color:#fff;font-size:10px;font-weight:700;padding:1px 6px;border-radius:3px;margin-right:6px;vertical-align:middle;text-transform:uppercase;">了解</span>`
	default:
		return ""
	}
}
