// Package store provides SQLite-based storage for NewsBot articles and digests.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/analyzer"
	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/sources"
	_ "modernc.org/sqlite"
)

// Schema is the SQLite schema for NewsBot.
const Schema = `
CREATE TABLE IF NOT EXISTS articles (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT NOT NULL,
    url         TEXT NOT NULL UNIQUE,
    source      TEXT NOT NULL,
    author      TEXT,
    content     TEXT,
    published_at TIMESTAMP,
    fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tags        TEXT
);

CREATE TABLE IF NOT EXISTS digests (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    date         TEXT NOT NULL UNIQUE,
    headlines    TEXT NOT NULL,
    summary      TEXT,
    tokens_used  INTEGER DEFAULT 0,
    cost         REAL DEFAULT 0,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_articles_source ON articles(source);
CREATE INDEX IF NOT EXISTS idx_articles_fetched ON articles(fetched_at);
CREATE INDEX IF NOT EXISTS idx_digests_date ON digests(date);
`

// Store provides NewsBot data persistence.
type Store struct {
	db *sql.DB
}

// New creates a new Store and initializes the schema.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if _, err := db.Exec(Schema); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Store{db: db}, nil
}

// SaveArticles stores fetched articles (skipping duplicates by URL).
func (s *Store) SaveArticles(ctx context.Context, articles []sources.Article) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO articles (title, url, source, author, content, published_at, fetched_at, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	saved := 0
	for _, a := range articles {
		tags, _ := json.Marshal(a.Tags)
		result, err := stmt.ExecContext(ctx, a.Title, a.URL, a.Source, a.Author, a.Content, a.PublishedAt, a.FetchedAt, string(tags))
		if err != nil {
			continue
		}
		affected, _ := result.RowsAffected()
		saved += int(affected)
	}

	return saved, tx.Commit()
}

// SaveDigest stores a generated digest.
func (s *Store) SaveDigest(ctx context.Context, digest *analyzer.DailyDigest) error {
	headlines, _ := json.Marshal(digest.Headlines)
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO digests (date, headlines, summary, tokens_used, cost)
		VALUES (?, ?, ?, ?, ?)
	`, digest.Date, string(headlines), digest.Summary, digest.TokensUsed, digest.Cost)
	return err
}

// GetLatestDigest retrieves the most recent digest.
func (s *Store) GetLatestDigest(ctx context.Context) (*analyzer.DailyDigest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT date, headlines, summary, tokens_used, cost, created_at
		FROM digests ORDER BY created_at DESC LIMIT 1
	`)

	var digest analyzer.DailyDigest
	var headlinesJSON string
	var createdAt time.Time
	if err := row.Scan(&digest.Date, &headlinesJSON, &digest.Summary, &digest.TokensUsed, &digest.Cost, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	digest.GeneratedAt = createdAt
	json.Unmarshal([]byte(headlinesJSON), &digest.Headlines)
	return &digest, nil
}

// GetArticleCount returns the total number of stored articles.
func (s *Store) GetArticleCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM articles").Scan(&count)
	return count, err
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
