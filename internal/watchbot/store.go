// Package watchbot implements the Competitor Monitoring Bot.
package watchbot

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/RobinCoderZhao/devkit-suite/pkg/storage"
)

// Store provides persistence for WatchBot using the common storage layer.
type Store struct {
	db *storage.DB
}

// NewStore creates a new store with the given storage database.
func NewStore(db *storage.DB) *Store {
	return &Store{db: db}
}

// --- Competitors ---

// Competitor represents a monitored competitor for a specific user.
type Competitor struct {
	ID        int
	UserID    int
	Name      string
	Domain    string
	CreatedAt time.Time
}

// AddCompetitor inserts or returns an existing competitor for a user.
func (s *Store) AddCompetitor(ctx context.Context, userID int, name, domain string) (int, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))

	// PostgreSQL and SQLite have different upsert syntaxes if returning ID.
	// For simplicity, handle insert and then fallback to select.
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO competitors (user_id, name, domain) VALUES (?, ?, ?) 
		 ON CONFLICT(user_id, domain) DO UPDATE SET name=excluded.name`,
		userID, name, domain)
	if err != nil {
		return 0, fmt.Errorf("add competitor: %w", err)
	}

	id, _ := res.LastInsertId()
	if id == 0 {
		row := s.db.QueryRowContext(ctx, `SELECT id FROM competitors WHERE user_id = ? AND domain = ?`, userID, domain)
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
	}
	return int(id), nil
}

// GetCompetitor returns a competitor for a user by name.
func (s *Store) GetCompetitor(ctx context.Context, userID int, name string) (*Competitor, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, domain, created_at FROM competitors WHERE user_id = ? AND name = ? COLLATE NOCASE`, userID, name)
	c := &Competitor{}
	if err := row.Scan(&c.ID, &c.UserID, &c.Name, &c.Domain, &c.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return c, nil
}

// ListCompetitorsByUser returns all competitors for a specific user.
func (s *Store) ListCompetitorsByUser(ctx context.Context, userID int) ([]Competitor, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, name, domain, created_at FROM competitors WHERE user_id = ? ORDER BY name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Competitor
	for rows.Next() {
		var c Competitor
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Domain, &c.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

// --- Pages ---

// Page represents a monitored page.
type Page struct {
	ID            int
	CompetitorID  int
	URL           string
	PageType      string
	LastCheckedAt *time.Time
	CreatedAt     time.Time
}

// PageWithMeta includes competitor/user info for pipeline use.
type PageWithMeta struct {
	Page
	CompetitorName   string
	CompetitorDomain string
	UserID           int
	UserEmail        string
}

// AddPage inserts a page for a competitor.
func (s *Store) AddPage(ctx context.Context, competitorID int, url, pageType string) (int, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO pages (competitor_id, url, page_type) VALUES (?, ?, ?)`,
		competitorID, url, pageType)
	if err != nil {
		// Ignore constraint failures (URL already exists for this competitor)
		// SQLite: UNIQUE constraint failed, Postgres: unique_violation
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			row := s.db.QueryRowContext(ctx, `SELECT id FROM pages WHERE url = ? AND competitor_id = ?`, url, competitorID)
			var id int64
			if err := row.Scan(&id); err != nil {
				return 0, err
			}
			return int(id), nil
		}
		return 0, fmt.Errorf("add page: %w", err)
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// GetPagesByCompetitor retrieves all pages tracked for a specific competitor.
func (s *Store) GetPagesByCompetitor(ctx context.Context, competitorID int) ([]Page, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, competitor_id, url, page_type, last_checked_at, created_at FROM pages WHERE competitor_id = ?`,
		competitorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.CompetitorID, &p.URL, &p.PageType, &p.LastCheckedAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

// GetAllActivePages retrieves all monitored pages (Phase 1 simplistic pipeline logic).
// This is used by the global pipeline to fetch all URLs that need checking.
func (s *Store) GetAllActivePages(ctx context.Context) ([]PageWithMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.competitor_id, p.url, p.page_type, p.last_checked_at, p.created_at,
		       c.name, c.domain, c.user_id,
		       u.email
		FROM pages p
		JOIN competitors c ON c.id = p.competitor_id
		JOIN users u ON u.id = c.user_id
		ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []PageWithMeta
	for rows.Next() {
		var pm PageWithMeta
		if err := rows.Scan(
			&pm.ID, &pm.CompetitorID, &pm.URL, &pm.PageType, &pm.LastCheckedAt, &pm.CreatedAt,
			&pm.CompetitorName, &pm.CompetitorDomain, &pm.UserID,
			&pm.UserEmail,
		); err != nil {
			return nil, err
		}
		result = append(result, pm)
	}
	return result, nil
}

// UpdateLastChecked updates the last_checked_at timestamp for a page.
func (s *Store) UpdateLastChecked(ctx context.Context, pageID int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE pages SET last_checked_at = CURRENT_TIMESTAMP WHERE id = ?`, pageID)
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

// --- Analyses (formerly Changes) ---

// Change represents a detected change record (mapped to analyses table).
type Change struct {
	ID            int
	PageID        int
	OldSnapshotID sql.NullInt64 // Can be null for first snapshot
	NewSnapshotID int
	Severity      string
	Analysis      string
	DiffUnified   string
	CreatedAt     time.Time

	// Synthesized fields for diff stats (extract from DiffUnified or add to schema later)
	Additions int
	Deletions int

	// Populated by join for digest
	CompetitorID   int
	CompetitorName string
	PageURL        string
	PageType       string
	UserID         int
}

// SaveChange records a detected change.
func (s *Store) SaveChange(ctx context.Context, pageID, oldSnapID, newSnapID int, severity, analysis, diffUnified string, additions, deletions int) (int, error) {
	var oldSnap interface{}
	if oldSnapID > 0 {
		oldSnap = oldSnapID
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO analyses (page_id, old_snapshot_id, new_snapshot_id, severity, summary, raw_diff)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		pageID, oldSnap, newSnapID, severity, analysis, diffUnified)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// GetLatestChange fetches the most recent analysis change for a page.
func (s *Store) GetLatestChange(ctx context.Context, pageID int) (*Change, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, page_id, old_snapshot_id, new_snapshot_id, severity, summary, raw_diff, created_at 
		 FROM analyses 
		 WHERE page_id = ? 
		 ORDER BY created_at DESC LIMIT 1`, pageID)

	var c Change
	var summary, diffUnified sql.NullString
	err := row.Scan(&c.ID, &c.PageID, &c.OldSnapshotID, &c.NewSnapshotID, &c.Severity, &summary, &diffUnified, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.Analysis = summary.String
	c.DiffUnified = diffUnified.String
	return &c, nil
}

// GetTimelineByCompetitor returns all historical changes for a specific competitor's pages.
func (s *Store) GetTimelineByCompetitor(ctx context.Context, competitorID int) ([]Change, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT a.id, a.page_id, a.old_snapshot_id, a.new_snapshot_id, a.severity, a.summary, a.raw_diff, a.created_at, p.url 
		 FROM analyses a
		 JOIN pages p ON a.page_id = p.id
		 WHERE p.competitor_id = ?
		 ORDER BY a.created_at DESC`, competitorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Change
	for rows.Next() {
		var c Change
		var summary, diffUnified sql.NullString
		if err := rows.Scan(&c.ID, &c.PageID, &c.OldSnapshotID, &c.NewSnapshotID, &c.Severity, &summary, &diffUnified, &c.CreatedAt, &c.PageURL); err != nil {
			return nil, err
		}
		c.Analysis = summary.String
		c.DiffUnified = diffUnified.String
		result = append(result, c)
	}
	return result, nil
}

// --- Users (formerly Subscribers) ---

// User represents a tenant.
type User struct {
	ID    int
	Email string
	Plan  string
}

// ensureUser is a helper for testing/CLI to make sure a user exists.
func (s *Store) ensureUser(ctx context.Context, email string) (int, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, plan) VALUES (?, 'dummy_hash', 'free') 
         ON CONFLICT(email) DO NOTHING`, email)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		row := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ?`, email)
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
	}
	return int(id), nil
}

// UserWithCompetitors holds user info with their competitors.
type UserWithCompetitors struct {
	ID              int
	Email           string
	CompetitorIDs   []int
	CompetitorNames []string
}

// GetUsersWithCompetitors returns all users along with their monitored competitors.
// This replaces the old GetActiveSubscribers logic.
func (s *Store) GetUsersWithCompetitors(ctx context.Context) ([]UserWithCompetitors, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.email, GROUP_CONCAT(c.id) as comp_ids,
		       GROUP_CONCAT(c.name) as comp_names
		FROM users u
		JOIN competitors c ON c.user_id = u.id
		GROUP BY u.id, u.email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []UserWithCompetitors
	for rows.Next() {
		var uw UserWithCompetitors
		var compIDs, compNames string
		if err := rows.Scan(&uw.ID, &uw.Email, &compIDs, &compNames); err != nil {
			return nil, err
		}
		for _, idStr := range strings.Split(compIDs, ",") {
			if idStr == "" {
				continue
			}
			var cid int
			fmt.Sscan(idStr, &cid)
			uw.CompetitorIDs = append(uw.CompetitorIDs, cid)
		}
		if compNames != "" {
			uw.CompetitorNames = strings.Split(compNames, ",")
		}
		result = append(result, uw)
	}
	return result, nil
}

func (s *Store) InitMetadata(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS metadata (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`)
	return err
}

func (s *Store) GetMeta(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM metadata WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *Store) SetMeta(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO metadata (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	return err
}

func (s *Store) GetLastChangeTime(ctx context.Context) (time.Time, error) {
	var t time.Time
	err := s.db.QueryRowContext(ctx,
		`SELECT created_at FROM analyses ORDER BY created_at DESC LIMIT 1`).Scan(&t)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	return t, err
}
