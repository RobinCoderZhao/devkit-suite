// Package notify — formatter.go provides shared utilities for multi-channel formatting.
//
// Architecture:
//
//	formatter.go      — shared email skeleton, markdown→HTML, badge helpers
//	watchbot_fmt.go   — WatchBot-specific: DigestData + WatchEmailFormatter
//	newsbot_fmt.go    — NewsBot-specific: NewsDigestData + NewsEmailFormatter
//
// Each product defines its own data model and formatters.
// Adding a new channel (WeChat, Twitter, etc.) means adding a Format method per product.
package notify

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// Message is the shared output type for all formatters.
// Already defined in message.go / types — referenced here for documentation.

// ---- Shared HTML Email Skeleton ----

// EmailHeader renders the gradient header section of an HTML email.
func EmailHeader(title, subtitle string, gradientFrom, gradientTo string) string {
	return fmt.Sprintf(`
<!-- Header -->
<tr><td style="background:linear-gradient(135deg,%s 0%%,%s 100%%);border-radius:16px 16px 0 0;padding:32px 40px;text-align:center;">
  <h1 style="margin:0;font-size:28px;font-weight:800;color:#ffffff;letter-spacing:-0.5px;">%s</h1>
  <p style="margin:8px 0 0;font-size:15px;color:rgba(255,255,255,0.85);font-weight:500;">%s</p>
</td></tr>
`, gradientFrom, gradientTo, html.EscapeString(title), html.EscapeString(subtitle))
}

// EmailWrapperOpen renders the opening HTML for an email body.
func EmailWrapperOpen() string {
	return `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"></head>
<body style="margin:0;padding:0;background-color:#0f0f23;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#0f0f23;">
<tr><td align="center" style="padding:20px 10px;">
<table role="presentation" width="640" cellpadding="0" cellspacing="0" style="max-width:640px;width:100%;">
`
}

// EmailWrapperClose renders the closing HTML for an email body.
func EmailWrapperClose() string {
	return `
</table>
</td></tr>
</table>
</body>
</html>`
}

// EmailFooter renders the footer section.
func EmailFooter(productName, tagline string, accentColor string) string {
	return fmt.Sprintf(`
<!-- Footer -->
<tr><td style="background-color:#12121f;border-radius:0 0 16px 16px;padding:24px 40px;text-align:center;">
  <p style="margin:0;font-size:12px;color:#505070;line-height:1.6;">
    <strong style="color:%s;">%s</strong> — %s
  </p>
</td></tr>
`, accentColor, html.EscapeString(productName), html.EscapeString(tagline))
}

// EmailRowBgColor returns alternating row colors.
func EmailRowBgColor(index int) string {
	if index%2 == 1 {
		return "#16162a"
	}
	return "#1a1a2e"
}

// ---- Shared Markdown → HTML Conversion ----

// MarkdownToHTML converts simple markdown to inline HTML for email bodies.
// Handles: **bold**, *italic*, # headings, - lists, numbered lists.
func MarkdownToHTML(md string) string {
	if md == "" {
		return ""
	}

	var sb strings.Builder
	lines := strings.Split(md, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Bold and italic
		line = ConvertMarkdownInline(line)

		// Headers
		if strings.HasPrefix(line, "### ") {
			line = fmt.Sprintf(`<strong style="color:#e0e0e0;font-size:13px;">%s</strong>`, line[4:])
		} else if strings.HasPrefix(line, "## ") {
			line = fmt.Sprintf(`<strong style="color:#e0e0e0;font-size:14px;">%s</strong>`, line[3:])
		} else if strings.HasPrefix(line, "# ") {
			line = fmt.Sprintf(`<strong style="color:#f0f0f0;font-size:15px;">%s</strong>`, line[2:])
		}

		// Bullet lists
		if strings.HasPrefix(line, "- ") {
			line = fmt.Sprintf(`<span style="color:#808090;">•</span> %s`, line[2:])
		}

		sb.WriteString(fmt.Sprintf(`<p style="margin:4px 0;font-size:14px;line-height:1.6;color:#a0a0b8;">%s</p>`, line))
	}
	return sb.String()
}

// ConvertMarkdownInline converts **bold** and *italic* to HTML.
func ConvertMarkdownInline(s string) string {
	re := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	s = re.ReplaceAllString(s, `<strong style="color:#e0e0e0;">$1</strong>`)
	re2 := regexp.MustCompile(`\*([^*]+)\*`)
	s = re2.ReplaceAllString(s, `<em>$1</em>`)
	return s
}

// StripMarkdown removes markdown formatting for plain text output.
func StripMarkdown(s string) string {
	re := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	s = re.ReplaceAllString(s, "$1")
	re2 := regexp.MustCompile(`\*([^*]+)\*`)
	s = re2.ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`(?m)^#{1,4}\s+`).ReplaceAllString(s, "")
	return s
}

// ---- Shared Badge Helpers ----

// ImportanceBadgeHTML returns styled HTML for high/medium/low importance.
func ImportanceBadgeHTML(level, label string) string {
	color := "#607d8b"
	switch level {
	case "high", "critical":
		color = "#e53935"
	case "medium", "important":
		color = "#ff9800"
	case "low", "minor":
		color = "#4caf50"
	}
	return fmt.Sprintf(`<span style="display:inline-block;background:%s;color:#fff;padding:2px 8px;border-radius:4px;font-size:11px;font-weight:600;margin-right:8px;vertical-align:middle;">%s</span>`,
		color, html.EscapeString(label))
}

// ImportanceEmoji returns an emoji for a level string.
func ImportanceEmoji(level string) string {
	switch level {
	case "high", "critical":
		return "🔴"
	case "medium", "important":
		return "🟡"
	case "low", "minor":
		return "🟢"
	default:
		return "⚪"
	}
}

// DiffStatsHTML returns styled +N / -N badges.
func DiffStatsHTML(additions, deletions int) string {
	return fmt.Sprintf(`<span style="display:inline-block;background:rgba(76,175,80,0.15);color:#81c784;font-size:11px;padding:2px 8px;border-radius:10px;margin-right:4px;">+%d</span><span style="display:inline-block;background:rgba(244,67,54,0.15);color:#ef9a9a;font-size:11px;padding:2px 8px;border-radius:10px;">-%d</span>`,
		additions, deletions)
}

// TagsHTML renders a list of tags as styled pills.
func TagsHTML(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	var parts []string
	for _, tag := range tags {
		parts = append(parts,
			fmt.Sprintf(`<span style="display:inline-block;background:rgba(102,126,234,0.15);color:#8b9cf7;font-size:11px;padding:2px 8px;border-radius:10px;margin:2px 4px 2px 0;">%s</span>`,
				html.EscapeString(tag)))
	}
	return strings.Join(parts, "")
}
