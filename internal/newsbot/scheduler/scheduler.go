// Package scheduler provides the cron scheduler for NewsBot.
package scheduler

import (
	"context"
	"log/slog"
	"time"
)

// Job represents a scheduled task.
type Job struct {
	Name     string
	Schedule string // cron expression like "0 8 * * *" or simple like "every 1h"
	Fn       func(ctx context.Context) error
}

// Scheduler runs jobs at specified intervals.
type Scheduler struct {
	jobs   []Job
	logger *slog.Logger
	done   chan struct{}
}

// NewScheduler creates a new scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		logger: slog.Default(),
		done:   make(chan struct{}),
	}
}

// Add registers a job with the scheduler.
func (s *Scheduler) Add(job Job) {
	s.jobs = append(s.jobs, job)
}

// RunOnce executes all registered jobs once (useful for testing).
func (s *Scheduler) RunOnce(ctx context.Context) error {
	for _, job := range s.jobs {
		s.logger.Info("running job", "name", job.Name)
		start := time.Now()
		if err := job.Fn(ctx); err != nil {
			s.logger.Error("job failed", "name", job.Name, "error", err, "duration", time.Since(start))
			return err
		}
		s.logger.Info("job completed", "name", job.Name, "duration", time.Since(start))
	}
	return nil
}

// Start begins the scheduler loop. It runs jobs at their configured intervals.
// For MVP, uses simple interval-based scheduling.
func (s *Scheduler) Start(ctx context.Context, interval time.Duration) {
	s.logger.Info("scheduler started", "interval", interval, "jobs", len(s.jobs))

	// Run once immediately
	s.RunOnce(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped")
			return
		case <-s.done:
			s.logger.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.RunOnce(ctx)
		}
	}
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	close(s.done)
}
