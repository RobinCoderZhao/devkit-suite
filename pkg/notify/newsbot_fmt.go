// Package notify — newsbot_fmt.go provides NewsBot-specific formatters.
//
// NewsBot data: daily news headlines with importance, tags, source, and summary.
// Style: purple accent, newspaper/digest theme, multi-language support.
package notify

import (
	"fmt"
	"html"
	"strings"
)

// ---- NewsBot Data Model ----

// NewsDigestData holds NewsBot daily digest content.
type NewsDigestData struct {
	Title     string // e.g. "AI 日报" or "AI Daily"
	Date      string
	Summary   string // today's overview
	Headlines []NewsHeadline
	// Cost tracking
	TokensUsed int
	Cost       float64
	// Labels (i18n)
	Labels NewsLabels
}

// NewsHeadline represents one news item.
type NewsHeadline struct {
	Title      string
	Summary    string
	URL        string
	Source     string
	Importance string // "high", "medium", "low"
	Tags       []string
}

// NewsLabels holds i18n labels for rendering.
type NewsLabels struct {
	DailyTitle  string // "AI 日报"
	Overview    string // "今日概要"
	ReadMore    string // "阅读原文"
	Source      string // "来源"
	Important   string // "重要"
	Watch       string // "关注"
	Info        string // "资讯"
	GeneratedBy string // "由 NewsBot 生成"
	TokenUsage  string // "Token 消耗: %d · 费用: $%.4f"
}

// ---- NewsBot Email Formatter ----

// NewsEmailFormatter produces rich HTML email for NewsBot daily digests.
type NewsEmailFormatter struct{}

func NewNewsEmailFormatter() *NewsEmailFormatter { return &NewsEmailFormatter{} }

func (f *NewsEmailFormatter) Format(data NewsDigestData) Message {
	var sb strings.Builder

	sb.WriteString(EmailWrapperOpen())
	sb.WriteString(EmailHeader(
		"🤖 "+data.Labels.DailyTitle,
		data.Date,
		"#667eea", "#764ba2",
	))

	// Summary section
	if data.Summary != "" {
		summaryHTML := f.formatSummary(data.Summary)
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
`, html.EscapeString(data.Labels.Overview), summaryHTML))
	}

	// Headlines
	for i, h := range data.Headlines {
		badge := f.importanceBadge(h.Importance, data.Labels)
		tags := TagsHTML(h.Tags)

		linkHTML := ""
		if h.URL != "" {
			linkHTML = fmt.Sprintf(`<a href="%s" style="color:#667eea;font-size:12px;text-decoration:none;font-weight:500;">%s</a>`,
				html.EscapeString(h.URL), html.EscapeString(data.Labels.ReadMore))
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
`, i+1, EmailRowBgColor(i), i+1,
			badge,
			html.EscapeString(h.Title),
			html.EscapeString(h.Summary),
			linkHTML, html.EscapeString(h.Source),
			tags))
	}

	// Footer with token usage
	tokenInfo := ""
	if data.TokensUsed > 0 {
		tokenInfo = fmt.Sprintf("<br>"+data.Labels.TokenUsage, data.TokensUsed, data.Cost)
	}
	sb.WriteString(fmt.Sprintf(`
<!-- Footer -->
<tr><td style="background-color:#12121f;border-radius:0 0 16px 16px;padding:24px 40px;text-align:center;">
  <p style="margin:0;font-size:12px;color:#505070;line-height:1.6;">
    <strong style="color:#667eea;">%s</strong>%s
  </p>
</td></tr>
`, html.EscapeString(data.Labels.GeneratedBy), tokenInfo))

	sb.WriteString(EmailWrapperClose())

	return Message{
		Title:    fmt.Sprintf("🤖 %s — %s", data.Labels.DailyTitle, data.Date),
		Body:     f.formatPlainText(data),
		HTMLBody: sb.String(),
		Format:   "html",
	}
}

func (f *NewsEmailFormatter) formatPlainText(data NewsDigestData) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# 🤖 %s — %s\n\n", data.Labels.DailyTitle, data.Date))
	if data.Summary != "" {
		sb.WriteString(fmt.Sprintf("📝 %s\n%s\n\n---\n\n", data.Labels.Overview, data.Summary))
	}
	for i, h := range data.Headlines {
		emoji := ImportanceEmoji(h.Importance)
		sb.WriteString(fmt.Sprintf("%s **%d. %s**\n", emoji, i+1, h.Title))
		if h.Summary != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", h.Summary))
		}
		if h.URL != "" {
			sb.WriteString(fmt.Sprintf("   🔗 [%s](%s) | %s: %s\n", data.Labels.ReadMore, h.URL, data.Labels.Source, h.Source))
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("---\n*%s*\n", data.Labels.GeneratedBy))
	return sb.String()
}

func (f *NewsEmailFormatter) formatSummary(summary string) string {
	// Split by Chinese period (。) and English sentences
	normalized := strings.ReplaceAll(summary, "。", "\n")
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

func (f *NewsEmailFormatter) importanceBadge(importance string, labels NewsLabels) string {
	switch importance {
	case "high":
		return ImportanceBadgeHTML(importance, labels.Important)
	case "medium":
		return ImportanceBadgeHTML(importance, labels.Watch)
	case "low":
		return ImportanceBadgeHTML(importance, labels.Info)
	default:
		return ""
	}
}

// ---- NewsBot Telegram Formatter ----

// NewsTelegramFormatter produces Telegram Markdown for NewsBot.
type NewsTelegramFormatter struct{}

func NewNewsTelegramFormatter() *NewsTelegramFormatter { return &NewsTelegramFormatter{} }

func (f *NewsTelegramFormatter) Format(data NewsDigestData) Message {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🤖 *%s* — %s\n\n", data.Labels.DailyTitle, data.Date))

	if data.Summary != "" {
		sb.WriteString(fmt.Sprintf("📝 *%s*\n%s\n\n---\n\n", data.Labels.Overview, data.Summary))
	}

	for i, h := range data.Headlines {
		emoji := ImportanceEmoji(h.Importance)
		sb.WriteString(fmt.Sprintf("%s *%d. %s*\n", emoji, i+1, h.Title))
		if h.Summary != "" {
			sb.WriteString(h.Summary + "\n")
		}
		if h.URL != "" {
			sb.WriteString(fmt.Sprintf("[%s](%s) | %s\n", data.Labels.ReadMore, h.URL, h.Source))
		}
		if len(h.Tags) > 0 {
			sb.WriteString("#" + strings.Join(h.Tags, " #") + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("_%s_\n", data.Labels.GeneratedBy))

	return Message{
		Title:  fmt.Sprintf("🤖 %s — %s", data.Labels.DailyTitle, data.Date),
		Body:   sb.String(),
		Format: "markdown",
	}
}
