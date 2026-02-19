// Package watchbot implements the Competitor Monitoring Bot.
//
// WatchBot monitors competitor websites, API documentation, and changelog pages,
// detecting changes and generating AI-powered analysis alerts.
package watchbot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/differ"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/llm"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/notify"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/scraper"
)

// Target represents a monitoring target.
type Target struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	URL      string `json:"url" yaml:"url"`
	Category string `json:"category" yaml:"category"`                     // "api_docs", "changelog", "pricing", "blog"
	Interval string `json:"interval" yaml:"interval"`                     // "1h", "6h", "24h"
	Selector string `json:"selector,omitempty" yaml:"selector,omitempty"` // CSS selector for specific content
}

// Snapshot holds a point-in-time snapshot of a target's content.
type Snapshot struct {
	TargetID  string    `json:"target_id"`
	Content   string    `json:"content"`
	FetchedAt time.Time `json:"fetched_at"`
}

// ChangeAlert represents a detected change with AI analysis.
type ChangeAlert struct {
	TargetID   string            `json:"target_id"`
	TargetName string            `json:"target_name"`
	TargetURL  string            `json:"target_url"`
	Diff       differ.DiffResult `json:"diff"`
	Analysis   string            `json:"analysis"`
	Severity   string            `json:"severity"` // "critical", "important", "minor"
	DetectedAt time.Time         `json:"detected_at"`
}

// Pipeline orchestrates the fetch → diff → analyze → alert flow for a single target.
type Pipeline struct {
	fetcher    scraper.Fetcher
	llmClient  llm.Client
	dispatcher *notify.Dispatcher
	channels   []notify.Channel
	logger     *slog.Logger
}

// NewPipeline creates a new monitoring pipeline.
func NewPipeline(
	fetcher scraper.Fetcher,
	llmClient llm.Client,
	dispatcher *notify.Dispatcher,
	channels []notify.Channel,
) *Pipeline {
	return &Pipeline{
		fetcher:    fetcher,
		llmClient:  llmClient,
		dispatcher: dispatcher,
		channels:   channels,
		logger:     slog.Default(),
	}
}

// Check performs a single check on the target, comparing against the previous content.
func (p *Pipeline) Check(ctx context.Context, target Target, previousContent string) (*ChangeAlert, string, error) {
	p.logger.Info("checking target", "name", target.Name, "url", target.URL)

	// 1. Fetch current content
	result, err := p.fetcher.Fetch(ctx, target.URL, nil)
	if err != nil {
		return nil, previousContent, fmt.Errorf("fetch %s: %w", target.Name, err)
	}

	currentContent := result.CleanText
	if previousContent == "" {
		p.logger.Info("first snapshot captured", "target", target.Name, "size", len(currentContent))
		return nil, currentContent, nil
	}

	// 2. Diff
	diff := differ.TextDiff(previousContent, currentContent)
	if !diff.HasChanges {
		p.logger.Info("no changes detected", "target", target.Name)
		return nil, currentContent, nil
	}

	p.logger.Info("changes detected", "target", target.Name, "additions", diff.Stats.Additions, "deletions", diff.Stats.Deletions)

	// 3. Analyze with LLM
	analysis, severity, err := p.analyzeDiff(ctx, target, diff)
	if err != nil {
		p.logger.Warn("LLM analysis failed, using diff summary", "error", err)
		analysis = diff.Summary()
		severity = "important"
	}

	alert := &ChangeAlert{
		TargetID:   target.ID,
		TargetName: target.Name,
		TargetURL:  target.URL,
		Diff:       diff,
		Analysis:   analysis,
		Severity:   severity,
		DetectedAt: time.Now(),
	}

	// 4. Notify
	if err := p.sendAlert(ctx, alert); err != nil {
		p.logger.Warn("failed to send alert", "error", err)
	}

	return alert, currentContent, nil
}

func (p *Pipeline) analyzeDiff(ctx context.Context, target Target, diff differ.DiffResult) (string, string, error) {
	if p.llmClient == nil {
		return diff.Summary(), "important", nil
	}

	prompt := fmt.Sprintf(`你是竞品分析专家。以下是 "%s" (%s) 的页面变化：

变化类型：%s
新增 %d 行，删除 %d 行

Diff:
%s

请分析：
1. 这个变化的含义是什么？
2. 对我们的竞争策略有什么影响？
3. 建议的应对措施

用简洁中文回答（100字以内）。同时在最后用一行标注严重性：CRITICAL / IMPORTANT / MINOR`,
		target.Name, target.Category,
		diff.Summary(),
		diff.Stats.Additions, diff.Stats.Deletions,
		diff.Unified,
	)

	resp, err := p.llmClient.Generate(ctx, &llm.Request{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
	})
	if err != nil {
		return "", "", err
	}

	// Extract severity from response
	severity := "important"
	content := resp.Content
	for _, s := range []string{"CRITICAL", "IMPORTANT", "MINOR"} {
		if containsIgnoreCase(content, s) {
			severity = map[string]string{"CRITICAL": "critical", "IMPORTANT": "important", "MINOR": "minor"}[s]
			break
		}
	}

	return content, severity, nil
}

func (p *Pipeline) sendAlert(ctx context.Context, alert *ChangeAlert) error {
	emoji := map[string]string{"critical": "🔴", "important": "🟡", "minor": "🟢"}
	e := emoji[alert.Severity]
	if e == "" {
		e = "⚪"
	}

	msg := notify.Message{
		Title:  fmt.Sprintf("%s 竞品变动: %s", e, alert.TargetName),
		Body:   fmt.Sprintf("**%s** 检测到变化\n\n%s\n\n📊 +%d / -%d 行\n🔗 %s", alert.TargetName, alert.Analysis, alert.Diff.Stats.Additions, alert.Diff.Stats.Deletions, alert.TargetURL),
		Format: "markdown",
		URL:    alert.TargetURL,
	}

	return p.dispatcher.Dispatch(ctx, p.channels, msg)
}

func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			a, b := s[i+j], substr[j]
			if a >= 'a' && a <= 'z' {
				a -= 32
			}
			if b >= 'a' && b <= 'z' {
				b -= 32
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
