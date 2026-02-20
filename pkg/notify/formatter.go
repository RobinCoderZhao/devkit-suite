// Package notify — formatter.go provides a content adapter layer for multi-channel output.
// It converts structured change data into channel-specific formats:
//   - Email: Rich HTML with table layout, severity badges, formatted analysis
//   - Telegram: Markdown with emojis
//   - Plain: Terminal/log output
//
// Future channels (WeChat, Twitter, etc.) only need to add a new Format method.
package notify

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// DigestData holds the structured content for a monitoring digest.
// This is the channel-agnostic data model.
type DigestData struct {
	ChangeCount int
	Changes     []ChangeItem
	Unchanged   []string // competitor names that didn't change
	Date        string
}

// ChangeItem represents one detected change.
type ChangeItem struct {
	CompetitorName string
	PageType       string
	PageURL        string
	Severity       string // "critical", "important", "minor"
	Analysis       string // LLM analysis (may contain markdown)
	Additions      int
	Deletions      int
}

// ---- Formatter Interface ----

// Formatter converts DigestData into a Message for a specific channel.
type Formatter interface {
	Format(data DigestData) Message
}

// ---- Email HTML Formatter ----

// EmailFormatter produces rich HTML email matching the NewsBot template style.
type EmailFormatter struct{}

func NewEmailFormatter() *EmailFormatter { return &EmailFormatter{} }

func (f *EmailFormatter) Format(data DigestData) Message {
	var sb strings.Builder

	// Email wrapper (table-based for compatibility)
	sb.WriteString(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"></head>
<body style="margin:0;padding:0;background-color:#0f0f23;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#0f0f23;">
<tr><td align="center" style="padding:20px 10px;">
<table role="presentation" width="640" cellpadding="0" cellspacing="0" style="max-width:640px;width:100%;">
`)

	// Header with gradient
	sb.WriteString(fmt.Sprintf(`
<!-- Header -->
<tr><td style="background:linear-gradient(135deg,#e65100 0%%,#ff6d00 100%%);border-radius:16px 16px 0 0;padding:32px 40px;text-align:center;">
  <h1 style="margin:0;font-size:28px;font-weight:800;color:#ffffff;letter-spacing:-0.5px;">🔍 竞品监控报告</h1>
  <p style="margin:8px 0 0;font-size:15px;color:rgba(255,255,255,0.85);font-weight:500;">检测到 %d 个页面发生变化</p>
</td></tr>
`, data.ChangeCount))

	// Each change
	for i, c := range data.Changes {
		bgColor := "#1a1a2e"
		if i%2 == 1 {
			bgColor = "#16162a"
		}

		badge := f.severityBadgeHTML(c.Severity)
		analysisHTML := f.renderAnalysis(c.Analysis)
		statsHTML := fmt.Sprintf(`<span style="display:inline-block;background:rgba(76,175,80,0.15);color:#81c784;font-size:11px;padding:2px 8px;border-radius:10px;margin-right:4px;">+%d</span><span style="display:inline-block;background:rgba(244,67,54,0.15);color:#ef9a9a;font-size:11px;padding:2px 8px;border-radius:10px;">-%d</span>`,
			c.Additions, c.Deletions)

		sb.WriteString(fmt.Sprintf(`
<!-- Change %d -->
<tr><td style="background-color:%s;padding:24px 40px;border-bottom:1px solid rgba(255,255,255,0.04);">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
    <tr>
      <td style="vertical-align:top;width:36px;padding-top:2px;">
        <span style="display:inline-block;width:28px;height:28px;line-height:28px;text-align:center;background:rgba(255,109,0,0.12);border-radius:8px;font-size:13px;font-weight:700;color:#ff9800;">%d</span>
      </td>
      <td style="padding-left:12px;">
        <table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
          <tr><td>
            %s
            <span style="font-size:16px;font-weight:700;color:#f0f0f0;line-height:1.4;">%s</span>
            <span style="font-size:13px;color:#808090;margin-left:8px;">%s</span>
          </td></tr>
          <tr><td style="padding-top:12px;">
            %s
          </td></tr>
          <tr><td style="padding-top:10px;">
            <table role="presentation" cellpadding="0" cellspacing="0"><tr>
              <td style="padding-right:12px;">%s</td>
              <td><a href="%s" style="color:#ff9800;font-size:12px;text-decoration:none;font-weight:500;">查看原页面 →</a></td>
            </tr></table>
          </td></tr>
        </table>
      </td>
    </tr>
  </table>
</td></tr>
`, i+1, bgColor, i+1,
			badge,
			html.EscapeString(c.CompetitorName),
			html.EscapeString(c.PageType),
			analysisHTML,
			statsHTML,
			html.EscapeString(c.PageURL)))
	}

	// Unchanged section
	if len(data.Unchanged) > 0 {
		sb.WriteString(fmt.Sprintf(`
<!-- Unchanged -->
<tr><td style="background-color:#1a1a2e;padding:16px 40px;border-bottom:1px solid rgba(255,255,255,0.04);">
  <p style="margin:0;font-size:13px;color:#505070;">✅ 未发生变化：%s</p>
</td></tr>
`, html.EscapeString(strings.Join(data.Unchanged, "、"))))
	}

	// Footer
	sb.WriteString(`
<!-- Footer -->
<tr><td style="background-color:#12121f;border-radius:0 0 16px 16px;padding:24px 40px;text-align:center;">
  <p style="margin:0;font-size:12px;color:#505070;line-height:1.6;">
    <strong style="color:#ff9800;">WatchBot V2</strong> — 竞品变化监控系统<br>
    由 AI 驱动的自动分析与通知
  </p>
</td></tr>
`)

	// Close wrapper
	sb.WriteString(`
</table>
</td></tr>
</table>
</body>
</html>`)

	// Plain text version
	plain := f.formatPlainText(data)

	return Message{
		Title:    fmt.Sprintf("🔍 竞品监控报告 — 检测到 %d 个变化", data.ChangeCount),
		Body:     plain,
		HTMLBody: sb.String(),
		Format:   "html",
	}
}

// renderAnalysis converts markdown analysis to HTML paragraphs.
func (f *EmailFormatter) renderAnalysis(analysis string) string {
	if analysis == "" {
		return ""
	}

	var sb strings.Builder

	// Split by lines, convert markdown to HTML
	lines := strings.Split(analysis, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Convert markdown bold **text** to <strong>
		line = convertMarkdownBold(line)
		// Convert markdown headers
		if strings.HasPrefix(line, "### ") {
			line = fmt.Sprintf(`<strong style="color:#e0e0e0;font-size:13px;">%s</strong>`, line[4:])
		} else if strings.HasPrefix(line, "## ") {
			line = fmt.Sprintf(`<strong style="color:#e0e0e0;font-size:14px;">%s</strong>`, line[3:])
		} else if strings.HasPrefix(line, "# ") {
			line = fmt.Sprintf(`<strong style="color:#f0f0f0;font-size:15px;">%s</strong>`, line[2:])
		}
		// Convert list items
		if strings.HasPrefix(line, "- ") {
			line = fmt.Sprintf(`<span style="color:#808090;">•</span> %s`, line[2:])
		}
		// Number list items (1. 2. etc.)
		if matched, _ := regexp.MatchString(`^\d+\.\s`, line); matched {
			// Keep as-is but escape
		}

		sb.WriteString(fmt.Sprintf(`<p style="margin:4px 0;font-size:14px;line-height:1.6;color:#a0a0b8;">%s</p>`, line))
	}

	return sb.String()
}

func convertMarkdownBold(s string) string {
	// Replace **text** with <strong>text</strong>
	re := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	s = re.ReplaceAllString(s, `<strong style="color:#e0e0e0;">$1</strong>`)
	// Replace *text* with <em>text</em>
	re2 := regexp.MustCompile(`\*([^*]+)\*`)
	s = re2.ReplaceAllString(s, `<em>$1</em>`)
	return s
}

func (f *EmailFormatter) severityBadgeHTML(severity string) string {
	emoji, label, color := severityMeta(severity)
	return fmt.Sprintf(`<span style="display:inline-block;background:%s;color:#fff;padding:2px 8px;border-radius:4px;font-size:11px;font-weight:600;margin-right:8px;vertical-align:middle;">%s %s</span>`,
		color, emoji, label)
}

func (f *EmailFormatter) formatPlainText(data DigestData) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 竞品监控报告 — 检测到 %d 个变化\n\n", data.ChangeCount))
	for _, c := range data.Changes {
		emoji, label, _ := severityMeta(c.Severity)
		sb.WriteString(fmt.Sprintf("%s [%s] %s — %s\n", emoji, label, c.CompetitorName, c.PageType))
		if c.Analysis != "" {
			// Strip markdown for plain text
			plain := stripMarkdown(c.Analysis)
			sb.WriteString(plain + "\n")
		}
		sb.WriteString(fmt.Sprintf("📊 +%d / -%d 行 · 🔗 %s\n\n", c.Additions, c.Deletions, c.PageURL))
	}
	if len(data.Unchanged) > 0 {
		sb.WriteString("---\n✅ 未变化：" + strings.Join(data.Unchanged, "、") + "\n")
	}
	return sb.String()
}

// ---- Telegram Markdown Formatter ----

// TelegramFormatter produces Telegram-compatible Markdown.
type TelegramFormatter struct{}

func NewTelegramFormatter() *TelegramFormatter { return &TelegramFormatter{} }

func (f *TelegramFormatter) Format(data DigestData) Message {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 *竞品监控报告*\n检测到 %d 个变化\n\n", data.ChangeCount))

	for _, c := range data.Changes {
		emoji, label, _ := severityMeta(c.Severity)
		sb.WriteString(fmt.Sprintf("%s *%s* — %s · %s\n", emoji, label, c.CompetitorName, c.PageType))
		if c.Analysis != "" {
			// Keep markdown formatting for Telegram
			analysis := c.Analysis
			if len(analysis) > 500 {
				analysis = analysis[:500] + "..."
			}
			sb.WriteString(analysis + "\n")
		}
		sb.WriteString(fmt.Sprintf("📊 +%d / -%d · [查看原页面](%s)\n\n", c.Additions, c.Deletions, c.PageURL))
	}
	if len(data.Unchanged) > 0 {
		sb.WriteString("✅ 未变化：" + strings.Join(data.Unchanged, "、") + "\n")
	}

	return Message{
		Title:  fmt.Sprintf("🔍 竞品监控 — %d 个变化", data.ChangeCount),
		Body:   sb.String(),
		Format: "markdown",
	}
}

// ---- Shared helpers ----

func severityMeta(severity string) (emoji, label, color string) {
	switch severity {
	case "critical":
		return "🔴", "Critical", "#e53935"
	case "important":
		return "🟡", "Important", "#ff9800"
	case "minor":
		return "🟢", "Minor", "#4caf50"
	default:
		return "⚪", severity, "#607d8b"
	}
}

func stripMarkdown(s string) string {
	// Remove **bold**, *italic*, # headers, - lists
	re := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	s = re.ReplaceAllString(s, "$1")
	re2 := regexp.MustCompile(`\*([^*]+)\*`)
	s = re2.ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`(?m)^#{1,4}\s+`).ReplaceAllString(s, "")
	return s
}
