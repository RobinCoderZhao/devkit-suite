// Package store provides SQLite-based storage for NewsBot articles and digests.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/RobinCoderZhao/devkit-suite/internal/newsbot/analyzer"
	"github.com/RobinCoderZhao/devkit-suite/internal/newsbot/sources"
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
    date         TEXT NOT NULL,
    language     TEXT NOT NULL DEFAULT 'zh',
    headlines    TEXT NOT NULL,
    summary      TEXT,
    tokens_used  INTEGER DEFAULT 0,
    cost         REAL DEFAULT 0,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(date, language)
);

CREATE TABLE IF NOT EXISTS subscribers (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER DEFAULT 0,
    target_type TEXT NOT NULL,
    target_id   TEXT NOT NULL,
    languages   TEXT NOT NULL DEFAULT 'zh',
    active      INTEGER DEFAULT 1,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(target_type, target_id)
);

CREATE INDEX IF NOT EXISTS idx_articles_source ON articles(source);
CREATE INDEX IF NOT EXISTS idx_articles_fetched ON articles(fetched_at);
CREATE INDEX IF NOT EXISTS idx_digests_date ON digests(date);
`

// Subscriber represents a notification subscriber.
type Subscriber struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Languages  string    `json:"languages"` // comma-separated: "zh,en"
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
}

// LanguageList returns the subscriber's languages as a slice.
func (s Subscriber) LanguageList() []string {
	parts := strings.Split(s.Languages, ",")
	var langs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			langs = append(langs, p)
		}
	}
	return langs
}

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
// Returns the list of newly saved articles (not previously in DB).
func (s *Store) SaveArticles(ctx context.Context, articles []sources.Article) ([]sources.Article, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO articles (title, url, source, author, content, published_at, fetched_at, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var newArticles []sources.Article
	for _, a := range articles {
		tags, _ := json.Marshal(a.Tags)
		result, err := stmt.ExecContext(ctx, a.Title, a.URL, a.Source, a.Author, a.Content, a.PublishedAt, a.FetchedAt, string(tags))
		if err != nil {
			continue
		}
		affected, _ := result.RowsAffected()
		if affected > 0 {
			newArticles = append(newArticles, a)
		}
	}

	return newArticles, tx.Commit()
}

// SaveDigest stores a generated digest for a specific language.
func (s *Store) SaveDigest(ctx context.Context, digest *analyzer.DailyDigest, lang string) error {
	if lang == "" {
		lang = "zh"
	}
	headlines, _ := json.Marshal(digest.Headlines)
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO digests (date, language, headlines, summary, tokens_used, cost)
		VALUES (?, ?, ?, ?, ?, ?)
	`, digest.Date, lang, string(headlines), digest.Summary, digest.TokensUsed, digest.Cost)
	return err
}

// GetLatestDigest retrieves the most recent digest for a given language.
func (s *Store) GetLatestDigest(ctx context.Context, lang string) (*analyzer.DailyDigest, error) {
	if lang == "" {
		lang = "zh"
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT date, headlines, summary, tokens_used, cost, created_at
		FROM digests WHERE language = ? ORDER BY created_at DESC LIMIT 1
	`, lang)

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

// --- Subscriber Management ---

// AddSubscriber adds or updates a subscriber.
func (s *Store) AddSubscriber(ctx context.Context, userID int, targetType, targetID, languages string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO subscribers (user_id, target_type, target_id, languages, active)
		VALUES (?, ?, ?, ?, 1)
		ON CONFLICT(target_type, target_id) DO UPDATE SET user_id = ?, languages = ?, active = 1
	`, userID, targetType, targetID, languages, userID, languages)
	return err
}

// RemoveSubscriber deactivates a subscriber by ID for a specific user.
func (s *Store) RemoveSubscriber(ctx context.Context, id, userID int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE subscribers SET active = 0 WHERE id = ? AND user_id = ?
	`, id, userID)
	return err
}

// RemoveSubscriberByTarget deactivates a subscriber by its target_type and target_id (Admin use).
func (s *Store) RemoveSubscriberByTarget(ctx context.Context, targetType, targetID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE subscribers SET active = 0 WHERE target_type = ? AND target_id = ?
	`, targetType, targetID)
	return err
}

// GetUserSubscribers retrieves all active subscribers for a specific user.
func (s *Store) GetUserSubscribers(ctx context.Context, userID int) ([]Subscriber, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, target_type, target_id, languages, active, created_at 
		FROM subscribers WHERE active = 1 AND user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscriber
	for rows.Next() {
		var sub Subscriber
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.TargetType, &sub.TargetID, &sub.Languages, &sub.Active, &sub.CreatedAt); err != nil {
			continue
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// GetSubscriber returns a specific subscriber by target_type and target_id.
func (s *Store) GetSubscriber(ctx context.Context, targetType, targetID string) (*Subscriber, error) {
	row := s.db.QueryRowContext(ctx, "SELECT id, user_id, target_type, target_id, languages, active, created_at FROM subscribers WHERE target_type = ? AND target_id = ?", targetType, targetID)
	var sub Subscriber
	if err := row.Scan(&sub.ID, &sub.UserID, &sub.TargetType, &sub.TargetID, &sub.Languages, &sub.Active, &sub.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not subscribed or not found
		}
		return nil, err
	}
	return &sub, nil
}

// GetActiveSubscribers returns all active subscribers.
func (s *Store) GetActiveSubscribers(ctx context.Context) ([]Subscriber, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, target_type, target_id, languages, active, created_at FROM subscribers WHERE active = 1
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscriber
	for rows.Next() {
		var sub Subscriber
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.TargetType, &sub.TargetID, &sub.Languages, &sub.Active, &sub.CreatedAt); err != nil {
			continue
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
