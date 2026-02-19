// Package watchbot implements the Competitor Monitoring Bot.
package watchbot

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Store provides SQLite-based persistence for WatchBot.
type Store struct {
	db *sql.DB
}

// NewStore creates a new store with the given SQLite database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// InitDB creates all required tables.
func (s *Store) InitDB(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS competitors (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			domain     TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS pages (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			competitor_id   INTEGER NOT NULL REFERENCES competitors(id) ON DELETE CASCADE,
			url             TEXT NOT NULL UNIQUE,
			page_type       TEXT NOT NULL DEFAULT 'general',
			check_interval  INTEGER DEFAULT 86400,
			last_checked    TIMESTAMP,
			status          TEXT DEFAULT 'active',
			UNIQUE(competitor_id, url)
		)`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			page_id     INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
			content     TEXT NOT NULL,
			checksum    TEXT NOT NULL,
			captured_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_snapshots_page_time ON snapshots(page_id, captured_at DESC)`,
		`CREATE TABLE IF NOT EXISTS changes (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			page_id         INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
			old_snapshot_id INTEGER REFERENCES snapshots(id),
			new_snapshot_id INTEGER REFERENCES snapshots(id),
			severity        TEXT DEFAULT 'important',
			analysis        TEXT,
			diff_unified    TEXT,
			diff_additions  INTEGER DEFAULT 0,
			diff_deletions  INTEGER DEFAULT 0,
			detected_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS subscribers (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			email      TEXT NOT NULL UNIQUE,
			active     INTEGER DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			subscriber_id INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
			competitor_id INTEGER NOT NULL REFERENCES competitors(id) ON DELETE CASCADE,
			notify_level  TEXT DEFAULT 'all',
			PRIMARY KEY(subscriber_id, competitor_id)
		)`,
	}
	for _, q := range queries {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("init table: %w", err)
		}
	}
	return nil
}

// --- Competitors ---

// Competitor represents a monitored competitor.
type Competitor struct {
	ID        int
	Name      string
	Domain    string
	CreatedAt time.Time
}

// AddCompetitor inserts or returns existing competitor. Returns the competitor ID.
func (s *Store) AddCompetitor(ctx context.Context, name, domain string) (int, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	// Try insert
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO competitors (name, domain) VALUES (?, ?) ON CONFLICT(domain) DO UPDATE SET name=excluded.name`,
		name, domain)
	if err != nil {
		return 0, fmt.Errorf("add competitor: %w", err)
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		// Already existed, fetch ID
		row := s.db.QueryRowContext(ctx, `SELECT id FROM competitors WHERE domain = ?`, domain)
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
	}
	return int(id), nil
}

// GetCompetitor returns a competitor by name.
func (s *Store) GetCompetitor(ctx context.Context, name string) (*Competitor, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, domain, created_at FROM competitors WHERE name = ? COLLATE NOCASE`, name)
	c := &Competitor{}
	if err := row.Scan(&c.ID, &c.Name, &c.Domain, &c.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return c, nil
}

// ListCompetitors returns all competitors with their page count.
func (s *Store) ListCompetitors(ctx context.Context) ([]Competitor, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, domain, created_at FROM competitors ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Competitor
	for rows.Next() {
		var c Competitor
		if err := rows.Scan(&c.ID, &c.Name, &c.Domain, &c.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

// RemoveCompetitor deletes a competitor and all related data (cascade).
func (s *Store) RemoveCompetitor(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM competitors WHERE name = ? COLLATE NOCASE`, name)
	return err
}

// --- Pages ---

// Page represents a monitored page.
type Page struct {
	ID            int
	CompetitorID  int
	URL           string
	PageType      string
	CheckInterval int
	LastChecked   *time.Time
	Status        string
}

// PageWithMeta includes competitor info for pipeline use.
type PageWithMeta struct {
	Page
	CompetitorName   string
	CompetitorDomain string
}

// AddPage inserts a page for a competitor.
func (s *Store) AddPage(ctx context.Context, competitorID int, url, pageType string) (int, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO pages (competitor_id, url, page_type) VALUES (?, ?, ?) ON CONFLICT(url) DO NOTHING`,
		competitorID, url, pageType)
	if err != nil {
		return 0, fmt.Errorf("add page: %w", err)
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		row := s.db.QueryRowContext(ctx, `SELECT id FROM pages WHERE url = ?`, url)
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
	}
	return int(id), nil
}

// GetAllActivePages returns all active pages with competitor metadata.
func (s *Store) GetAllActivePages(ctx context.Context) ([]PageWithMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.competitor_id, p.url, p.page_type, p.check_interval, p.last_checked, p.status,
		       c.name, c.domain
		FROM pages p
		JOIN competitors c ON c.id = p.competitor_id
		WHERE p.status = 'active'
		ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []PageWithMeta
	for rows.Next() {
		var pm PageWithMeta
		if err := rows.Scan(
			&pm.ID, &pm.CompetitorID, &pm.URL, &pm.PageType,
			&pm.CheckInterval, &pm.LastChecked, &pm.Status,
			&pm.CompetitorName, &pm.CompetitorDomain,
		); err != nil {
			return nil, err
		}
		result = append(result, pm)
	}
	return result, nil
}

// GetPagesByCompetitor returns all pages for a given competitor.
func (s *Store) GetPagesByCompetitor(ctx context.Context, competitorID int) ([]Page, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, competitor_id, url, page_type, check_interval, last_checked, status
		 FROM pages WHERE competitor_id = ? ORDER BY page_type`, competitorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.CompetitorID, &p.URL, &p.PageType, &p.CheckInterval, &p.LastChecked, &p.Status); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

// UpdateLastChecked updates the last_checked timestamp for a page.
func (s *Store) UpdateLastChecked(ctx context.Context, pageID int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE pages SET last_checked = CURRENT_TIMESTAMP WHERE id = ?`, pageID)
	return err
}

// --- Snapshots ---

// SaveSnapshot stores a new content snapshot.
func (s *Store) SaveSnapshot(ctx context.Context, pageID int, content, checksum string) (int, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO snapshots (page_id, content, checksum) VALUES (?, ?, ?)`,
		pageID, content, checksum)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// GetLatestSnapshot returns the most recent snapshot for a page.
func (s *Store) GetLatestSnapshot(ctx context.Context, pageID int) (id int, content, checksum string, err error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, content, checksum FROM snapshots WHERE page_id = ? ORDER BY captured_at DESC LIMIT 1`,
		pageID)
	err = row.Scan(&id, &content, &checksum)
	if err == sql.ErrNoRows {
		return 0, "", "", nil
	}
	return
}

// --- Changes ---

// Change represents a detected change record.
type Change struct {
	ID            int
	PageID        int
	OldSnapshotID int
	NewSnapshotID int
	Severity      string
	Analysis      string
	DiffUnified   string
	Additions     int
	Deletions     int
	DetectedAt    time.Time

	// Populated by join for digest
	CompetitorName string
	PageURL        string
	PageType       string
}

// SaveChange records a detected change.
func (s *Store) SaveChange(ctx context.Context, pageID, oldSnapID, newSnapID int, severity, analysis, diffUnified string, additions, deletions int) (int, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO changes (page_id, old_snapshot_id, new_snapshot_id, severity, analysis, diff_unified, diff_additions, diff_deletions)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		pageID, oldSnapID, newSnapID, severity, analysis, diffUnified, additions, deletions)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// --- Subscribers ---

// Subscriber represents an email subscriber.
type Subscriber struct {
	ID    int
	Email string
}

// AddSubscriber creates or returns existing subscriber.
func (s *Store) AddSubscriber(ctx context.Context, email string) (int, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO subscribers (email) VALUES (?) ON CONFLICT(email) DO UPDATE SET active=1`,
		email)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		row := s.db.QueryRowContext(ctx, `SELECT id FROM subscribers WHERE email = ?`, email)
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
	}
	return int(id), nil
}

// RemoveSubscriber deactivates a subscriber.
func (s *Store) RemoveSubscriber(ctx context.Context, email string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE subscribers SET active = 0 WHERE email = ? COLLATE NOCASE`, email)
	return err
}

// --- Subscriptions ---

// Subscribe links a subscriber to a competitor.
func (s *Store) Subscribe(ctx context.Context, subscriberID, competitorID int) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO subscriptions (subscriber_id, competitor_id) VALUES (?, ?) ON CONFLICT DO NOTHING`,
		subscriberID, competitorID)
	return err
}

// GetActiveSubscribers returns all active subscribers with their subscribed competitor IDs.
func (s *Store) GetActiveSubscribers(ctx context.Context) ([]SubscriberWithCompetitors, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.email, GROUP_CONCAT(sub.competitor_id) as comp_ids,
		       GROUP_CONCAT(c.name) as comp_names
		FROM subscribers s
		JOIN subscriptions sub ON sub.subscriber_id = s.id
		JOIN competitors c ON c.id = sub.competitor_id
		WHERE s.active = 1
		GROUP BY s.id, s.email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []SubscriberWithCompetitors
	for rows.Next() {
		var sw SubscriberWithCompetitors
		var compIDs, compNames string
		if err := rows.Scan(&sw.ID, &sw.Email, &compIDs, &compNames); err != nil {
			return nil, err
		}
		for _, id := range strings.Split(compIDs, ",") {
			var cid int
			fmt.Sscan(id, &cid)
			sw.CompetitorIDs = append(sw.CompetitorIDs, cid)
		}
		sw.CompetitorNames = strings.Split(compNames, ",")
		result = append(result, sw)
	}
	return result, nil
}

// SubscriberWithCompetitors holds subscriber info with their subscriptions.
type SubscriberWithCompetitors struct {
	ID              int
	Email           string
	CompetitorIDs   []int
	CompetitorNames []string
}

// ListSubscribers returns all active subscribers for display.
func (s *Store) ListSubscribers(ctx context.Context) ([]SubscriberWithCompetitors, error) {
	return s.GetActiveSubscribers(ctx)
}
