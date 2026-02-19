// Package publisher formats and distributes the daily digest.
package publisher

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/analyzer"
	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/i18n"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/notify"
)

// Publisher formats digests and sends them via notification channels.
type Publisher struct {
	dispatcher *notify.Dispatcher
}

// NewPublisher creates a new publisher with the given dispatcher.
func NewPublisher(dispatcher *notify.Dispatcher) *Publisher {
	return &Publisher{dispatcher: dispatcher}
}

// PublishToEmail sends a digest in the specified language to the given email.
func (p *Publisher) PublishToEmail(ctx context.Context, digest *analyzer.DailyDigest, lang i18n.Language, email string) error {
	labels := i18n.GetLabels(lang)
	msg := notify.Message{
		Title:    fmt.Sprintf("🤖 %s — %s", labels.DailyTitle, digest.Date),
		Body:     FormatDigest(digest, lang),
		HTMLBody: FormatDigestHTML(digest, lang),
		Format:   "html",
	}

	// Create a one-off email notifier for this recipient
	notifier := notify.NewEmailNotifier(notify.EmailConfig{
		SMTPHost: p.dispatcher.EmailConfig().SMTPHost,
		SMTPPort: p.dispatcher.EmailConfig().SMTPPort,
		From:     p.dispatcher.EmailConfig().From,
		Password: p.dispatcher.EmailConfig().Password,
		To:       email,
	})
	return notifier.Send(ctx, msg)
}

// FormatDigest converts a DailyDigest into a Markdown-formatted message.
func FormatDigest(digest *analyzer.DailyDigest, lang i18n.Language) string {
	labels := i18n.GetLabels(lang)
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# 🤖 %s — %s\n\n", labels.DailyTitle, digest.Date))

	if digest.Summary != "" {
		sb.WriteString(fmt.Sprintf("📝 **%s**\n%s\n\n", labels.Overview, digest.Summary))
	}

	sb.WriteString("---\n\n")

	for i, h := range digest.Headlines {
		emoji := importanceEmoji(h.Importance)
		sb.WriteString(fmt.Sprintf("%s **%d. %s**\n", emoji, i+1, h.Title))
		if h.Summary != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", h.Summary))
		}
		if h.URL != "" {
			sb.WriteString(fmt.Sprintf("   🔗 [%s](%s) | %s: %s\n", labels.ReadMore, h.URL, labels.Source, h.Source))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("*%s*\n", labels.GeneratedBy))

	return sb.String()
}

// FormatDigestHTML generates a professional HTML newsletter from DailyDigest.
func FormatDigestHTML(digest *analyzer.DailyDigest, lang i18n.Language) string {
	labels := i18n.GetLabels(lang)
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
  <h1 style="margin:0;font-size:28px;font-weight:800;color:#ffffff;letter-spacing:-0.5px;">🤖 %s</h1>
  <p style="margin:8px 0 0;font-size:15px;color:rgba(255,255,255,0.85);font-weight:500;">%s</p>
</td></tr>
`, html.EscapeString(labels.DailyTitle), digest.Date))

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
        <p style="margin:0 0 12px;font-size:11px;font-weight:700;text-transform:uppercase;letter-spacing:1.5px;color:#667eea;">%s</p>
        %s
      </td>
    </tr>
  </table>
</td></tr>
`, html.EscapeString(labels.Overview), summaryHTML))
	}

	// Headlines
	for i, h := range digest.Headlines {
		badge := importanceBadge(h.Importance, labels)
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
			linkHTML = fmt.Sprintf(`<a href="%s" style="color:#667eea;font-size:12px;text-decoration:none;font-weight:500;">%s</a>`,
				html.EscapeString(h.URL), html.EscapeString(labels.ReadMore))
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
    <strong style="color:#667eea;">%s</strong><br>
    %s
  </p>
</td></tr>
`, html.EscapeString(labels.GeneratedBy),
		fmt.Sprintf(labels.TokenUsage, digest.TokensUsed, digest.Cost)))

	// Close wrapper
	sb.WriteString(`
</table>
</td></tr>
</table>
</body>
</html>`)

	return sb.String()
}

// formatSummaryLines splits summary into individual sentences for any language.
func formatSummaryLines(summary string) string {
	// Split by both Chinese period (。) and English period followed by space (. )
	// First normalize: replace 。 with a sentinel, then split
	normalized := strings.ReplaceAll(summary, "。", "\n")
	// Split English sentences: ". " followed by uppercase letter or emoji
	normalized = splitEnglishSentences(normalized)
	parts := strings.Split(normalized, "\n")

	var lines []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		lines = append(lines, html.EscapeString(p))
	}
	if len(lines) <= 1 {
		return fmt.Sprintf(`<p style="margin:0;font-size:15px;line-height:1.8;color:#e0e0e0;">%s</p>`, html.EscapeString(summary))
	}

	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 6px;font-size:14px;line-height:1.7;color:#e0e0e0;">
          <span style="color:#667eea;margin-right:6px;">▸</span>%s</p>`, line))
	}
	return sb.String()
}

// splitEnglishSentences splits text at ". " boundaries (English sentence endings).
func splitEnglishSentences(s string) string {
	var result strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		result.WriteRune(runes[i])
		// Check for ". " pattern (period + space, not inside numbers like "4.5")
		if runes[i] == '.' && i+1 < len(runes) && runes[i+1] == ' ' {
			// Make sure the character before '.' is a letter (not a digit)
			if i > 0 && !isDigit(runes[i-1]) {
				result.WriteRune('\n')
				i++ // skip the space
				continue
			}
		}
	}
	return result.String()
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
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

func importanceBadge(importance string, labels i18n.Labels) string {
	switch importance {
	case "high":
		return fmt.Sprintf(`<span style="display:inline-block;background:#ff4757;color:#fff;font-size:10px;font-weight:700;padding:1px 6px;border-radius:3px;margin-right:6px;vertical-align:middle;text-transform:uppercase;">%s</span>`, html.EscapeString(labels.Important))
	case "medium":
		return fmt.Sprintf(`<span style="display:inline-block;background:#ffa502;color:#fff;font-size:10px;font-weight:700;padding:1px 6px;border-radius:3px;margin-right:6px;vertical-align:middle;text-transform:uppercase;">%s</span>`, html.EscapeString(labels.Watch))
	case "low":
		return fmt.Sprintf(`<span style="display:inline-block;background:#2ed573;color:#fff;font-size:10px;font-weight:700;padding:1px 6px;border-radius:3px;margin-right:6px;vertical-align:middle;text-transform:uppercase;">%s</span>`, html.EscapeString(labels.Info))
	default:
		return ""
	}
}
