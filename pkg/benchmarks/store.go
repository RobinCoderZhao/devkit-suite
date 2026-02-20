package benchmarks

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages benchmark data in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a benchmark store using the given database connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.initTables(); err != nil {
		return nil, fmt.Errorf("init benchmark tables: %w", err)
	}
	return s, nil
}

func (s *Store) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS benchmark_scores (
			id              INTEGER PRIMARY KEY,
			benchmark_id    TEXT NOT NULL,
			model_name      TEXT NOT NULL,
			model_provider  TEXT NOT NULL,
			variant         TEXT DEFAULT '',
			score           REAL NOT NULL,
			source_url      TEXT DEFAULT '',
			scraped_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(benchmark_id, model_name, variant)
		)`,
		`CREATE TABLE IF NOT EXISTS benchmark_models (
			name          TEXT PRIMARY KEY,
			provider      TEXT NOT NULL,
			thinking      TEXT DEFAULT '',
			gen           TEXT DEFAULT 'latest',
			display_order INTEGER DEFAULT 0
		)`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// UpsertScore inserts or updates a benchmark score.
func (s *Store) UpsertScore(ctx context.Context, score BenchmarkScore) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO benchmark_scores (benchmark_id, model_name, model_provider, variant, score, source_url, scraped_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(benchmark_id, model_name, variant) DO UPDATE SET
			score = excluded.score,
			source_url = excluded.source_url,
			scraped_at = excluded.scraped_at,
			model_provider = excluded.model_provider
	`, score.BenchmarkID, score.ModelName, score.ModelProvider,
		score.Variant, score.Score, score.SourceURL, time.Now())
	return err
}

// BulkUpsert inserts multiple scores efficiently.
func (s *Store) BulkUpsert(ctx context.Context, scores []BenchmarkScore) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO benchmark_scores (benchmark_id, model_name, model_provider, variant, score, source_url, scraped_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(benchmark_id, model_name, variant) DO UPDATE SET
			score = excluded.score,
			source_url = excluded.source_url,
			scraped_at = excluded.scraped_at,
			model_provider = excluded.model_provider
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, sc := range scores {
		if _, err := stmt.ExecContext(ctx, sc.BenchmarkID, sc.ModelName, sc.ModelProvider,
			sc.Variant, sc.Score, sc.SourceURL, now); err != nil {
			return fmt.Errorf("upsert %s/%s: %w", sc.BenchmarkID, sc.ModelName, err)
		}
	}
	return tx.Commit()
}

// GetScoresForReport loads all scores for the given models into a BenchmarkReport.
func (s *Store) GetScoresForReport(ctx context.Context, models []ModelConfig, date string) (*BenchmarkReport, error) {
	report := NewReport(models, date)

	// Build model name list for IN clause
	modelNames := make([]interface{}, len(models))
	placeholders := ""
	for i, m := range models {
		modelNames[i] = m.Name
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
	}

	query := fmt.Sprintf(`
		SELECT benchmark_id, model_name, variant, score
		FROM benchmark_scores
		WHERE model_name IN (%s)
		ORDER BY benchmark_id, variant, model_name
	`, placeholders)

	rows, err := s.db.QueryContext(ctx, query, modelNames...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var benchID, modelName, variant string
		var score float64
		if err := rows.Scan(&benchID, &modelName, &variant, &score); err != nil {
			return nil, err
		}
		report.SetScore(benchID, variant, modelName, score)
	}
	return report, rows.Err()
}

// GetAllScores returns all stored scores.
func (s *Store) GetAllScores(ctx context.Context) ([]BenchmarkScore, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, benchmark_id, model_name, model_provider, variant, score, source_url, scraped_at
		FROM benchmark_scores ORDER BY benchmark_id, model_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []BenchmarkScore
	for rows.Next() {
		var sc BenchmarkScore
		if err := rows.Scan(&sc.ID, &sc.BenchmarkID, &sc.ModelName, &sc.ModelProvider,
			&sc.Variant, &sc.Score, &sc.SourceURL, &sc.ScrapedAt); err != nil {
			return nil, err
		}
		scores = append(scores, sc)
	}
	return scores, rows.Err()
}

// ScoreCount returns the total number of stored scores.
func (s *Store) ScoreCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM benchmark_scores`).Scan(&count)
	return count, err
}

// SaveModels persists the model configuration.
func (s *Store) SaveModels(ctx context.Context, models []ModelConfig) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear and re-insert
	if _, err := tx.ExecContext(ctx, `DELETE FROM benchmark_models`); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO benchmark_models (name, provider, thinking, gen, display_order)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range models {
		if _, err := stmt.ExecContext(ctx, m.Name, m.Provider, m.Thinking, m.Gen, m.DisplayOrder); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// LoadModels loads the model configuration from DB, falls back to defaults.
func (s *Store) LoadModels(ctx context.Context) ([]ModelConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, provider, thinking, gen, display_order
		FROM benchmark_models ORDER BY display_order
	`)
	if err != nil {
		return DefaultModels, nil
	}
	defer rows.Close()

	var models []ModelConfig
	for rows.Next() {
		var m ModelConfig
		if err := rows.Scan(&m.Name, &m.Provider, &m.Thinking, &m.Gen, &m.DisplayOrder); err != nil {
			return DefaultModels, nil
		}
		models = append(models, m)
	}
	if len(models) == 0 {
		return DefaultModels, nil
	}
	return models, nil
}
