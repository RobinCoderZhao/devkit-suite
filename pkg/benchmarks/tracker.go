package benchmarks

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Tracker coordinates periodic benchmark data collection and notification.
type Tracker struct {
	store    *Store
	scraper  *Scraper
	interval time.Duration
	OnUpdate func(report *BenchmarkReport) // called when new data is found
}

// NewTracker creates a new benchmark tracker with the given interval.
func NewTracker(store *Store, scraper *Scraper, interval time.Duration) *Tracker {
	return &Tracker{
		store:    store,
		scraper:  scraper,
		interval: interval,
	}
}

// Run starts the periodic scraping loop. Blocks until context is cancelled.
func (t *Tracker) Run(ctx context.Context, models []ModelConfig) {
	log.Printf("[benchmark-tracker] Starting with interval %v", t.interval)

	// Run once immediately
	t.runOnce(ctx, models)

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[benchmark-tracker] Shutting down")
			return
		case <-ticker.C:
			t.runOnce(ctx, models)
		}
	}
}

// RunOnce runs a single scrape cycle and triggers notification if new data found.
func (t *Tracker) runOnce(ctx context.Context, models []ModelConfig) {
	oldCount, _ := t.store.ScoreCount(ctx)

	n, err := t.scraper.ScrapeAll(ctx)
	if err != nil {
		log.Printf("[benchmark-tracker] Scrape error: %v", err)
	}

	newCount, _ := t.store.ScoreCount(ctx)
	newScores := newCount - oldCount

	log.Printf("[benchmark-tracker] Scrape complete: %d processed, %d new scores", n, newScores)

	if newScores > 0 && t.OnUpdate != nil {
		date := time.Now().Format("2006-01-02")
		report, err := t.store.GetScoresForReport(ctx, models, date)
		if err != nil {
			log.Printf("[benchmark-tracker] Report error: %v", err)
			return
		}
		report.FilterEmptyModels(1, 10)
		t.OnUpdate(report)
	}
}

// QuickReport generates a benchmark report without scraping.
func (t *Tracker) QuickReport(ctx context.Context, models []ModelConfig) (*BenchmarkReport, error) {
	date := time.Now().Format("2006-01-02")
	report, err := t.store.GetScoresForReport(ctx, models, date)
	if err != nil {
		return nil, fmt.Errorf("build report: %w", err)
	}
	report.FilterEmptyModels(1, 10)
	return report, nil
}
