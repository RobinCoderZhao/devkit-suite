// Package watchbot implements the Competitor Monitoring Bot.
//
// WatchBot monitors competitor websites, API documentation, and changelog pages,
// detecting changes and generating AI-powered analysis alerts.
package watchbot

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/differ"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/llm"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/notify"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/scraper"
)

// Target represents a monitoring target (legacy, kept for backward compatibility).
type Target struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	URL      string `json:"url" yaml:"url"`
	Category string `json:"category" yaml:"category"`
	Interval string `json:"interval" yaml:"interval"`
	Selector string `json:"selector,omitempty" yaml:"selector,omitempty"`
}

// GlobalPipeline orchestrates the two-phase check flow:
//
//	Phase 1: Global fetch + diff (per URL, not per user)
//	Phase 2: Per-user aggregated notifications
type GlobalPipeline struct {
	store      *Store
	fetcher    scraper.Fetcher
	llmClient  llm.Client
	dispatcher *notify.Dispatcher
	channels   []notify.Channel
	logger     *slog.Logger
}

// NewGlobalPipeline creates a new global monitoring pipeline.
func NewGlobalPipeline(
	store *Store,
	fetcher scraper.Fetcher,
	llmClient llm.Client,
	dispatcher *notify.Dispatcher,
	channels []notify.Channel,
) *GlobalPipeline {
	return &GlobalPipeline{
		store:      store,
		fetcher:    fetcher,
		llmClient:  llmClient,
		dispatcher: dispatcher,
		channels:   channels,
		logger:     slog.Default(),
	}
}

// RunCheck executes a full monitoring round: fetch all pages, diff, analyze, notify.
func (gp *GlobalPipeline) RunCheck(ctx context.Context) error {
	// Phase 1: Global fetch (per URL, deduplicated)
	pages, err := gp.store.GetAllActivePages(ctx)
	if err != nil {
		return fmt.Errorf("get pages: %w", err)
	}

	gp.logger.Info("starting check", "pages", len(pages))

	var changesThisRound []Change
	for _, page := range pages {
		change, err := gp.checkPage(ctx, page)
		if err != nil {
			gp.logger.Error("check page failed", "page", page.URL, "error", err)
			continue
		}
		if change != nil {
			changesThisRound = append(changesThisRound, *change)
		}
	}

	gp.logger.Info("phase 1 complete", "pages_checked", len(pages), "changes_detected", len(changesThisRound))

	if len(changesThisRound) == 0 {
		gp.logger.Info("no changes detected, skipping notifications")
		return nil
	}

	// Phase 2: Per-user aggregated notifications
	subscribers, err := gp.store.GetActiveSubscribers(ctx)
	if err != nil {
		return fmt.Errorf("get subscribers: %w", err)
	}

	for _, sub := range subscribers {
		// Filter changes for this subscriber's competitors
		userChanges := filterBySubscription(changesThisRound, sub)
		if len(userChanges) == 0 {
			continue
		}

		// Compose one digest message (use email formatter for rich HTML)
		formatter := notify.NewEmailFormatter()
		msg := ComposeDigest(userChanges, sub, formatter)

		// Send via email
		if gp.dispatcher != nil && gp.dispatcher.EmailConfig().SMTPHost != "" {
			emailNotifier := notify.NewEmailNotifierForRecipient(gp.dispatcher.EmailConfig(), sub.Email)
			if err := emailNotifier.Send(ctx, msg); err != nil {
				gp.logger.Error("email send failed", "email", sub.Email, "error", err)
			} else {
				gp.logger.Info("digest sent", "email", sub.Email, "changes", len(userChanges))
			}
		} else if len(gp.channels) > 0 {
			// Fallback to dispatcher channels (Telegram)
			if err := gp.dispatcher.Dispatch(ctx, gp.channels, msg); err != nil {
				gp.logger.Error("notify failed", "email", sub.Email, "error", err)
			}
		} else {
			// stdout fallback
			fmt.Printf("\n📧 → %s\n%s\n", sub.Email, msg.Body)
		}
	}

	gp.logger.Info("phase 2 complete", "subscribers_notified", len(subscribers))
	return nil
}

// checkPage fetches a page, diffs against latest snapshot, and returns a Change if detected.
func (gp *GlobalPipeline) checkPage(ctx context.Context, page PageWithMeta) (*Change, error) {
	// Fetch
	result, err := gp.fetcher.Fetch(ctx, page.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", page.URL, err)
	}

	currentContent := result.CleanText
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(currentContent)))

	// Update last checked
	_ = gp.store.UpdateLastChecked(ctx, page.ID)

	// Get latest snapshot
	_, _, prevChecksum, err := gp.store.GetLatestSnapshot(ctx, page.ID)
	if err != nil {
		return nil, err
	}

	// First snapshot
	if prevChecksum == "" {
		gp.logger.Info("first snapshot", "page", page.CompetitorName, "url", page.URL, "size", len(currentContent))
		_, _ = gp.store.SaveSnapshot(ctx, page.ID, currentContent, checksum)
		return nil, nil
	}

	// No changes
	if checksum == prevChecksum {
		gp.logger.Info("no changes", "page", page.CompetitorName)
		return nil, nil
	}

	// Save new snapshot
	newSnapID, _ := gp.store.SaveSnapshot(ctx, page.ID, currentContent, checksum)

	// Get previous content for diff
	prevSnapID, prevContent, _, _ := gp.store.GetLatestSnapshot(ctx, page.ID)
	// Note: after saving new, "latest" is the new one. We need the one before.
	// Actually we should get prev before saving. Let me fix the logic:
	// We already checked prevChecksum != "" and checksum != prevChecksum.
	// The prev snapshot was fetched before saving. Let's use a different approach.

	// Re-fetch the second latest (the previous one)
	_ = prevSnapID // we need prev content which we didn't save. Let me refactor.
	_ = prevContent

	// Simpler: get prev snapshot content before saving new one was the right approach.
	// But we already saved. For now, compute diff from the fetch result.
	// This is fine because we checked prevChecksum != checksum.

	// Actually let's fix: we should diff the old content with current content.
	// Let me query the second-to-last snapshot.
	row := gp.store.db.QueryRowContext(ctx,
		`SELECT id, content FROM snapshots WHERE page_id = ? ORDER BY captured_at DESC LIMIT 1 OFFSET 1`,
		page.ID)
	var oldSnapID int
	var oldContent string
	if err := row.Scan(&oldSnapID, &oldContent); err != nil {
		// Can't find old snapshot, skip
		return nil, nil
	}

	// Diff
	diff := differ.TextDiff(oldContent, currentContent)
	if !diff.HasChanges {
		return nil, nil
	}

	gp.logger.Info("changes detected",
		"page", page.CompetitorName,
		"url", page.URL,
		"additions", diff.Stats.Additions,
		"deletions", diff.Stats.Deletions)

	// LLM analysis
	analysis, severity := gp.analyzeDiff(ctx, page, diff)

	// Save change
	changeID, _ := gp.store.SaveChange(ctx, page.ID, oldSnapID, newSnapID,
		severity, analysis, diff.Unified, diff.Stats.Additions, diff.Stats.Deletions)

	return &Change{
		ID:             changeID,
		PageID:         page.ID,
		OldSnapshotID:  oldSnapID,
		NewSnapshotID:  newSnapID,
		Severity:       severity,
		Analysis:       analysis,
		DiffUnified:    diff.Unified,
		Additions:      diff.Stats.Additions,
		Deletions:      diff.Stats.Deletions,
		DetectedAt:     time.Now(),
		CompetitorName: page.CompetitorName,
		PageURL:        page.URL,
		PageType:       page.PageType,
	}, nil
}

// analyzeDiff uses LLM to analyze a change.
func (gp *GlobalPipeline) analyzeDiff(ctx context.Context, page PageWithMeta, diff differ.DiffResult) (string, string) {
	if gp.llmClient == nil {
		return diff.Summary(), "important"
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

用简洁中文回答（150字以内）。同时在最后用一行标注严重性：CRITICAL / IMPORTANT / MINOR`,
		page.CompetitorName, page.PageType,
		diff.Summary(),
		diff.Stats.Additions, diff.Stats.Deletions,
		truncate(diff.Unified, 3000),
	)

	resp, err := gp.llmClient.Generate(ctx, &llm.Request{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   512,
		Temperature: 0.3,
	})
	if err != nil {
		gp.logger.Warn("LLM analysis failed", "error", err)
		return diff.Summary(), "important"
	}

	// Extract severity
	severity := "important"
	content := resp.Content
	for _, s := range []string{"CRITICAL", "IMPORTANT", "MINOR"} {
		if strings.Contains(strings.ToUpper(content), s) {
			severity = strings.ToLower(s)
			break
		}
	}
	return content, severity
}

// filterBySubscription filters changes to only those for a subscriber's competitors.
func filterBySubscription(changes []Change, sub SubscriberWithCompetitors) []Change {
	compNames := make(map[string]bool)
	for _, name := range sub.CompetitorNames {
		compNames[name] = true
	}
	var result []Change
	for _, c := range changes {
		if compNames[c.CompetitorName] {
			result = append(result, c)
		}
	}
	return result
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}
