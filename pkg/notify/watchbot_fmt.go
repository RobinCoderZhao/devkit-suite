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
	Groups      []CompetitorGroup // changes grouped by competitor
	Unchanged   []string          // competitor names without changes
	Date        string
}

// CompetitorGroup holds all changes for a single competitor.
type CompetitorGroup struct {
	CompetitorName string
	MaxSeverity    string // highest severity among all pages
	Changes        []WatchChangeItem
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

// GroupChanges groups a flat list of changes by CompetitorName, preserving order.
func GroupChanges(items []WatchChangeItem) []CompetitorGroup {
	orderMap := make(map[string]int) // competitor -> first-seen index
	groupMap := make(map[string]*CompetitorGroup)
	var order []string

	for _, item := range items {
		name := item.CompetitorName
		if _, exists := orderMap[name]; !exists {
			orderMap[name] = len(order)
			order = append(order, name)
			groupMap[name] = &CompetitorGroup{
				CompetitorName: name,
				MaxSeverity:    item.Severity,
			}
		}
		g := groupMap[name]
		g.Changes = append(g.Changes, item)
		if severityRank(item.Severity) > severityRank(g.MaxSeverity) {
			g.MaxSeverity = item.Severity
		}
	}

	groups := make([]CompetitorGroup, len(order))
	for i, name := range order {
		groups[i] = *groupMap[name]
	}
	return groups
}

func severityRank(s string) int {
	switch s {
	case "critical":
		return 3
	case "important":
		return 2
	case "minor":
		return 1
	default:
		return 0
	}
}

// ---- WatchBot Email Formatter ----

// WatchEmailFormatter produces rich HTML email for WatchBot digests.
type WatchEmailFormatter struct{}

func NewWatchEmailFormatter() *WatchEmailFormatter { return &WatchEmailFormatter{} }

func (f *WatchEmailFormatter) Format(data WatchDigestData) Message {
	var sb strings.Builder

	// Count total page changes
	totalPages := 0
	for _, g := range data.Groups {
		totalPages += len(g.Changes)
	}

	sb.WriteString(EmailWrapperOpen())
	sb.WriteString(EmailHeader(
		"🔍 竞品监控报告",
		fmt.Sprintf("检测到 %d 个竞品共 %d 个页面发生变化", len(data.Groups), totalPages),
		"#e65100", "#ff6d00",
	))

	for gi, group := range data.Groups {
		emoji := ImportanceEmoji(group.MaxSeverity)

		// ── Competitor header row ──
		sb.WriteString(fmt.Sprintf(`
<!-- Competitor %d: %s -->
<tr><td style="background-color:%s;padding:20px 40px 8px 40px;border-bottom:none;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0"><tr>
    <td style="vertical-align:middle;">
      <span style="font-size:20px;font-weight:800;color:#f0f0f0;letter-spacing:0.3px;">%s %s</span>
      <span style="display:inline-block;margin-left:10px;padding:2px 10px;background:rgba(255,109,0,0.15);border-radius:10px;font-size:12px;color:#ff9800;font-weight:600;">%d 个页面变化</span>
    </td>
  </tr></table>
</td></tr>
`, gi+1, html.EscapeString(group.CompetitorName),
			EmailRowBgColor(gi),
			emoji,
			html.EscapeString(group.CompetitorName),
			len(group.Changes)))

		// ── Page changes under this competitor ──
		for pi, c := range group.Changes {
			severityEmoji := ImportanceEmoji(c.Severity)
			label := severityLabel(c.Severity)
			badge := ImportanceBadgeHTML(c.Severity, severityEmoji+" "+label)
			analysisHTML := MarkdownToHTML(c.Analysis)
			stats := DiffStatsHTML(c.Additions, c.Deletions)

			// Separator between pages (not before first)
			borderTop := ""
			if pi > 0 {
				borderTop = "border-top:1px solid rgba(255,255,255,0.06);"
			}

			sb.WriteString(fmt.Sprintf(`
<tr><td style="background-color:%s;padding:12px 40px 16px 56px;%s">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
    <tr><td>
      %s
      <span style="font-size:14px;font-weight:600;color:#d0d0e0;margin-left:4px;">%s</span>
    </td></tr>
    <tr><td style="padding-top:10px;">
      %s
    </td></tr>
    <tr><td style="padding-top:8px;">
      <table role="presentation" cellpadding="0" cellspacing="0"><tr>
        <td style="padding-right:12px;">%s</td>
        <td><a href="%s" style="color:#ff9800;font-size:12px;text-decoration:none;font-weight:500;">查看原页面 →</a></td>
      </tr></table>
    </td></tr>
  </table>
</td></tr>
`, EmailRowBgColor(gi), borderTop,
				badge,
				html.EscapeString(c.PageType),
				analysisHTML,
				stats,
				html.EscapeString(c.PageURL)))
		}

		// Bottom border after each competitor group
		sb.WriteString(fmt.Sprintf(`
<tr><td style="background-color:%s;padding:0;border-bottom:2px solid rgba(255,152,0,0.15);height:4px;"></td></tr>
`, EmailRowBgColor(gi)))
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
		Title:    fmt.Sprintf("🔍 竞品监控报告 — %d 个竞品发生变化", len(data.Groups)),
		Body:     f.formatPlainText(data),
		HTMLBody: sb.String(),
		Format:   "html",
	}
}

func (f *WatchEmailFormatter) formatPlainText(data WatchDigestData) string {
	var sb strings.Builder
	totalPages := 0
	for _, g := range data.Groups {
		totalPages += len(g.Changes)
	}
	sb.WriteString(fmt.Sprintf("🔍 竞品监控报告 — %d 个竞品 %d 个页面发生变化\n\n", len(data.Groups), totalPages))

	for _, group := range data.Groups {
		emoji := ImportanceEmoji(group.MaxSeverity)
		sb.WriteString(fmt.Sprintf("━━ %s %s (%d 个页面) ━━\n", emoji, group.CompetitorName, len(group.Changes)))
		for _, c := range group.Changes {
			sb.WriteString(fmt.Sprintf("\n  📄 %s [%s]\n", c.PageType, severityLabel(c.Severity)))
			if c.Analysis != "" {
				// Indent analysis lines
				for _, line := range strings.Split(StripMarkdown(c.Analysis), "\n") {
					if strings.TrimSpace(line) != "" {
						sb.WriteString("  " + line + "\n")
					}
				}
			}
			sb.WriteString(fmt.Sprintf("  📊 +%d / -%d 行 · 🔗 %s\n", c.Additions, c.Deletions, c.PageURL))
		}
		sb.WriteString("\n")
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
	totalPages := 0
	for _, g := range data.Groups {
		totalPages += len(g.Changes)
	}
	sb.WriteString(fmt.Sprintf("🔍 *竞品监控报告*\n%d 个竞品 %d 个页面变化\n\n", len(data.Groups), totalPages))

	for _, group := range data.Groups {
		emoji := ImportanceEmoji(group.MaxSeverity)
		sb.WriteString(fmt.Sprintf("*%s %s*\n", emoji, group.CompetitorName))
		for _, c := range group.Changes {
			sb.WriteString(fmt.Sprintf("  📄 *%s* · %s\n", c.PageType, severityLabel(c.Severity)))
			if c.Analysis != "" {
				analysis := c.Analysis
				if len(analysis) > 500 {
					analysis = analysis[:500] + "..."
				}
				sb.WriteString(analysis + "\n")
			}
			sb.WriteString(fmt.Sprintf("  📊 +%d / -%d · [查看原页面](%s)\n", c.Additions, c.Deletions, c.PageURL))
		}
		sb.WriteString("\n")
	}
	if len(data.Unchanged) > 0 {
		sb.WriteString("✅ 未变化：" + strings.Join(data.Unchanged, "、") + "\n")
	}

	return Message{
		Title:  fmt.Sprintf("🔍 竞品监控 — %d 个竞品变化", len(data.Groups)),
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
