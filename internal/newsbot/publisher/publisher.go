// Package publisher formats and distributes the daily digest.
package publisher

import (
	"context"
	"fmt"
	"strings"

	"github.com/RobinCoderZhao/devkit-suite/internal/newsbot/analyzer"
	"github.com/RobinCoderZhao/devkit-suite/internal/newsbot/i18n"
	"github.com/RobinCoderZhao/devkit-suite/pkg/notify"
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
	formatter := notify.NewNewsEmailFormatter()
	data := toNewsDigestData(digest, lang)
	msg := formatter.Format(data)

	notifier := notify.NewEmailNotifier(notify.EmailConfig{
		SMTPHost: p.dispatcher.EmailConfig().SMTPHost,
		SMTPPort: p.dispatcher.EmailConfig().SMTPPort,
		From:     p.dispatcher.EmailConfig().From,
		Password: p.dispatcher.EmailConfig().Password,
		To:       email,
	})
	return notifier.Send(ctx, msg)
}

// PublishToTelegram sends a digest in the specified language via Telegram.
func (p *Publisher) PublishToTelegram(ctx context.Context, digest *analyzer.DailyDigest, lang i18n.Language, channels []notify.Channel) error {
	formatter := notify.NewNewsTelegramFormatter()
	data := toNewsDigestData(digest, lang)
	msg := formatter.Format(data)
	return p.dispatcher.Dispatch(ctx, channels, msg)
}

// Publish sends a digest via all configured channels (backward compat).
func (p *Publisher) Publish(ctx context.Context, digest *analyzer.DailyDigest, lang i18n.Language, channels []notify.Channel) error {
	formatter := notify.NewNewsEmailFormatter()
	data := toNewsDigestData(digest, lang)
	msg := formatter.Format(data)
	return p.dispatcher.Dispatch(ctx, channels, msg)
}

// FormatDigest converts a DailyDigest into plain Markdown text.
func FormatDigest(digest *analyzer.DailyDigest, lang i18n.Language) string {
	formatter := notify.NewNewsEmailFormatter()
	data := toNewsDigestData(digest, lang)
	msg := formatter.Format(data)
	return msg.Body
}

// FormatDigestHTML generates HTML from a DailyDigest.
func FormatDigestHTML(digest *analyzer.DailyDigest, lang i18n.Language) string {
	formatter := notify.NewNewsEmailFormatter()
	data := toNewsDigestData(digest, lang)
	msg := formatter.Format(data)
	return msg.HTMLBody
}

// toNewsDigestData converts the analyzer model to the formatter data model.
func toNewsDigestData(digest *analyzer.DailyDigest, lang i18n.Language) notify.NewsDigestData {
	labels := i18n.GetLabels(lang)

	headlines := make([]notify.NewsHeadline, len(digest.Headlines))
	for i, h := range digest.Headlines {
		headlines[i] = notify.NewsHeadline{
			Title:      h.Title,
			Summary:    h.Summary,
			URL:        h.URL,
			Source:     h.Source,
			Importance: h.Importance,
			Tags:       h.Tags,
		}
	}

	return notify.NewsDigestData{
		Title:      labels.DailyTitle,
		Date:       digest.Date,
		Summary:    digest.Summary,
		Headlines:  headlines,
		TokensUsed: digest.TokensUsed,
		Cost:       digest.Cost,
		Labels: notify.NewsLabels{
			DailyTitle:  labels.DailyTitle,
			Overview:    labels.Overview,
			ReadMore:    labels.ReadMore,
			Source:      labels.Source,
			Important:   labels.Important,
			Watch:       labels.Watch,
			Info:        labels.Info,
			GeneratedBy: labels.GeneratedBy,
			TokenUsage:  labels.TokenUsage,
		},
	}
}

func importanceEmoji(importance string) string {
	return notify.ImportanceEmoji(importance)
}

func importanceBadge(importance string, labels i18n.Labels) string {
	switch importance {
	case "high":
		return notify.ImportanceBadgeHTML(importance, labels.Important)
	case "medium":
		return notify.ImportanceBadgeHTML(importance, labels.Watch)
	case "low":
		return notify.ImportanceBadgeHTML(importance, labels.Info)
	default:
		return ""
	}
}

// splitEnglishSentences splits text at ". " boundaries.
func splitEnglishSentences(s string) string {
	var result strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		result.WriteRune(runes[i])
		if runes[i] == '.' && i+1 < len(runes) && runes[i+1] == ' ' {
			if i > 0 && !isDigit(runes[i-1]) {
				result.WriteRune('\n')
				i++
				continue
			}
		}
	}
	return result.String()
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// formatSummaryLines is kept for backward compatibility — now delegates to shared formatter.
func formatSummaryLines(summary string) string {
	f := notify.NewNewsEmailFormatter()
	// Delegate to the exported method by calling Format with minimal data
	// Actually just inline the logic here for simplicity
	_ = f
	normalized := strings.ReplaceAll(summary, "。", "\n")
	parts := strings.Split(normalized, "\n")

	var lines []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		lines = append(lines, p)
	}
	if len(lines) <= 1 {
		return fmt.Sprintf(`<p style="margin:0;font-size:15px;line-height:1.8;color:#e0e0e0;">%s</p>`, summary)
	}

	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 6px;font-size:14px;line-height:1.7;color:#e0e0e0;">
          <span style="color:#667eea;margin-right:6px;">▸</span>%s</p>`, line))
	}
	return sb.String()
}
