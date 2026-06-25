package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// Open opens a SQLite database with recommended settings.
func Open(dbPath string) (*sql.DB, error) {
	// Use WAL mode for better concurrent read performance
	dsn := dbPath + "?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	db.SetMaxIdleConns(1)

	// Enable WAL mode explicitly
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return db, nil
}

// Close closes the database connection cleanly.
func Close(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}
