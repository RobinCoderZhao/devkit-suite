// Package storage provides a database abstraction layer supporting SQLite and PostgreSQL.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// Driver represents a database driver type.
type Driver string

const (
	SQLite   Driver = "sqlite"
	Postgres Driver = "postgres"
)

// Config holds database configuration.
type Config struct {
	Driver Driver `yaml:"driver" json:"driver"`
	DSN    string `yaml:"dsn" json:"dsn"` // Data Source Name
}

// DB wraps a *sql.DB with additional utilities.
type DB struct {
	*sql.DB
	driver Driver
	logger *slog.Logger
}

// Open creates a new database connection.
func Open(cfg Config) (*DB, error) {
	var driverName string
	switch cfg.Driver {
	case SQLite:
		driverName = "sqlite3"
	case Postgres:
		driverName = "postgres"
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db, err := sql.Open(driverName, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{
		DB:     db,
		driver: cfg.Driver,
		logger: slog.Default(),
	}, nil
}

// Driver returns the database driver type.
func (db *DB) DriverType() Driver {
	return db.driver
}

// Migrate runs the given SQL schema on the database.
func (db *DB) Migrate(ctx context.Context, schema string) error {
	_, err := db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	db.logger.Info("database migration completed")
	return nil
}

// Transaction wraps a function in a database transaction.
func (db *DB) Transaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	return tx.Commit()
}
