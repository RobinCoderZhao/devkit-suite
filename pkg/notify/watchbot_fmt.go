// Package notify — watchbot_fmt.go provides WatchBot-specific formatters.
//
// WatchBot data: competitor changes with LLM analysis, diff stats, severity.
// Style: orange accent, monitoring/surveillance theme.
package notify

import (
	"fmt"
	"html"
	"strings"
)

// ---- WatchBot Data Model ----

// WatchDigestData holds WatchBot monitoring results.
type WatchDigestData struct {
	ChangeCount int
	Changes     []WatchChangeItem
	Unchanged   []string // competitor names without changes
	Date        string
}

// WatchChangeItem represents one detected page change.
type WatchChangeItem struct {
	CompetitorName string
	PageType       string
	PageURL        string
	Severity       string // "critical", "important", "minor"
	Analysis       string // LLM analysis (may contain markdown)
	Additions      int
	Deletions      int
}

// ---- WatchBot Email Formatter ----

// WatchEmailFormatter produces rich HTML email for WatchBot digests.
type WatchEmailFormatter struct{}

func NewWatchEmailFormatter() *WatchEmailFormatter { return &WatchEmailFormatter{} }

func (f *WatchEmailFormatter) Format(data WatchDigestData) Message {
	var sb strings.Builder

	sb.WriteString(EmailWrapperOpen())
	sb.WriteString(EmailHeader(
		"🔍 竞品监控报告",
		fmt.Sprintf("检测到 %d 个页面发生变化", data.ChangeCount),
		"#e65100", "#ff6d00",
	))

	for i, c := range data.Changes {
		emoji := ImportanceEmoji(c.Severity)
		label := severityLabel(c.Severity)
		badge := ImportanceBadgeHTML(c.Severity, emoji+" "+label)
		analysisHTML := MarkdownToHTML(c.Analysis)
		stats := DiffStatsHTML(c.Additions, c.Deletions)

		// Single change: show emoji instead of numbered badge
		// Multiple changes: show numbered badge
		var indexBadge string
		if data.ChangeCount == 1 {
			indexBadge = fmt.Sprintf(
				`<span style="display:inline-block;width:28px;height:28px;line-height:28px;text-align:center;font-size:18px;">%s</span>`,
				emoji)
		} else {
			indexBadge = fmt.Sprintf(
				`<span style="display:inline-block;width:28px;height:28px;line-height:28px;text-align:center;background:rgba(255,109,0,0.12);border-radius:8px;font-size:13px;font-weight:700;color:#ff9800;">%d</span>`,
				i+1)
		}

		sb.WriteString(fmt.Sprintf(`
<!-- Change %d -->
<tr><td style="background-color:%s;padding:24px 40px;border-bottom:1px solid rgba(255,255,255,0.04);">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
    <tr>
      <td style="vertical-align:top;width:36px;padding-top:2px;">
        %s
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
`, i+1, EmailRowBgColor(i),
			indexBadge,
			badge,
			html.EscapeString(c.CompetitorName),
			html.EscapeString(c.PageType),
			analysisHTML,
			stats,
			html.EscapeString(c.PageURL)))
	}

	if len(data.Unchanged) > 0 {
		sb.WriteString(fmt.Sprintf(`
<tr><td style="background-color:#1a1a2e;padding:16px 40px;border-bottom:1px solid rgba(255,255,255,0.04);">
  <p style="margin:0;font-size:13px;color:#505070;">✅ 未发生变化：%s</p>
</td></tr>
`, html.EscapeString(strings.Join(data.Unchanged, "、"))))
	}

	sb.WriteString(EmailFooter("WatchBot V2", "竞品变化监控系统", "#ff9800"))
	sb.WriteString(EmailWrapperClose())

	return Message{
		Title:    fmt.Sprintf("🔍 竞品监控报告 — 检测到 %d 个变化", data.ChangeCount),
		Body:     f.formatPlainText(data),
		HTMLBody: sb.String(),
		Format:   "html",
	}
}

func (f *WatchEmailFormatter) formatPlainText(data WatchDigestData) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 竞品监控报告 — 检测到 %d 个变化\n\n", data.ChangeCount))
	for _, c := range data.Changes {
		emoji := ImportanceEmoji(c.Severity)
		label := severityLabel(c.Severity)
		sb.WriteString(fmt.Sprintf("%s [%s] %s — %s\n", emoji, label, c.CompetitorName, c.PageType))
		if c.Analysis != "" {
			sb.WriteString(StripMarkdown(c.Analysis) + "\n")
		}
		sb.WriteString(fmt.Sprintf("📊 +%d / -%d 行 · 🔗 %s\n\n", c.Additions, c.Deletions, c.PageURL))
	}
	if len(data.Unchanged) > 0 {
		sb.WriteString("---\n✅ 未变化：" + strings.Join(data.Unchanged, "、") + "\n")
	}
	return sb.String()
}

// ---- WatchBot Telegram Formatter ----

// WatchTelegramFormatter produces Telegram Markdown for WatchBot.
type WatchTelegramFormatter struct{}

func NewWatchTelegramFormatter() *WatchTelegramFormatter { return &WatchTelegramFormatter{} }

func (f *WatchTelegramFormatter) Format(data WatchDigestData) Message {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 *竞品监控报告*\n检测到 %d 个变化\n\n", data.ChangeCount))

	for _, c := range data.Changes {
		emoji := ImportanceEmoji(c.Severity)
		label := severityLabel(c.Severity)
		sb.WriteString(fmt.Sprintf("%s *%s* — %s · %s\n", emoji, label, c.CompetitorName, c.PageType))
		if c.Analysis != "" {
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

// ---- helpers ----

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
