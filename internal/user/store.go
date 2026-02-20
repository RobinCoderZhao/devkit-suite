// Package user implements tenant and authentication logic.
package user

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/RobinCoderZhao/devkit-suite/pkg/storage"
)

// Store provides persistence for Users and Organizations.
type Store struct {
	db *storage.DB
}

// NewStore creates a new user store.
func NewStore(db *storage.DB) *Store {
	return &Store{db: db}
}

// User represents a tenant in the system.
type User struct {
	ID           int
	Email        string
	PasswordHash string
	Plan         string
}

// CreateUser inserts a new user.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash, plan string) (int, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if plan == "" {
		plan = "free"
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, plan) VALUES (?, ?, ?)`,
		email, passwordHash, plan)
	if err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// GetUserByEmail finds a user by their email address.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, plan FROM users WHERE email = ?`, email)
	u := &User{}
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Plan); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}
	return u, nil
}
