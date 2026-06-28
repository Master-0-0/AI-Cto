package database

import (
	"database/sql"
	"fmt"
)

// Migration represents a single schema migration.
type Migration struct {
	Version int
	Up      string
}

// Migrations is the ordered list of schema migrations.
var Migrations = []Migration{
	{
		Version: 1,
		Up: `CREATE TABLE IF NOT EXISTS operators (
			id              TEXT PRIMARY KEY,
			username        TEXT UNIQUE NOT NULL,
			password_hash   TEXT NOT NULL,
			role            TEXT NOT NULL DEFAULT 'operator',
			created_at      TEXT NOT NULL DEFAULT (datetime('now')),
			last_login      TEXT,
			active          INTEGER NOT NULL DEFAULT 1
		)`,
	},
	{
		Version: 2,
		Up: `CREATE TABLE IF NOT EXISTS refresh_tokens (
			id              TEXT PRIMARY KEY,
			operator_id     TEXT NOT NULL,
			token_hash      TEXT UNIQUE NOT NULL,
			expires_at      TEXT NOT NULL,
			created_at      TEXT NOT NULL DEFAULT (datetime('now')),
			revoked         INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (operator_id) REFERENCES operators(id) ON DELETE CASCADE
		)`,
	},
	{
		Version: 3,
		Up: `CREATE TABLE IF NOT EXISTS listeners (
			id              TEXT PRIMARY KEY,
			name            TEXT NOT NULL,
			protocol        TEXT NOT NULL,
			host            TEXT NOT NULL DEFAULT '0.0.0.0',
			port            INTEGER NOT NULL,
			status          TEXT NOT NULL DEFAULT 'stopped',
			config_json     TEXT,
			created_at      TEXT NOT NULL DEFAULT (datetime('now')),
			started_at      TEXT
		)`,
	},
	{
		Version: 4,
		Up: `CREATE TABLE IF NOT EXISTS agents (
			id              TEXT PRIMARY KEY,
			listener_id     TEXT,
			hostname        TEXT NOT NULL,
			os              TEXT NOT NULL,
			arch            TEXT NOT NULL,
			username        TEXT,
			pid             INTEGER,
			local_ips       TEXT,
			version         TEXT NOT NULL DEFAULT '1.0.0',
			first_seen      TEXT NOT NULL DEFAULT (datetime('now')),
			last_seen       TEXT NOT NULL DEFAULT (datetime('now')),
			heartbeat_interval INTEGER NOT NULL DEFAULT 60,
			status          TEXT NOT NULL DEFAULT 'active',
			session_key     TEXT,
			metadata_json   TEXT,
			FOREIGN KEY (listener_id) REFERENCES listeners(id) ON DELETE SET NULL
		);
		CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
		CREATE INDEX IF NOT EXISTS idx_agents_last_seen ON agents(last_seen)`,
	},
	{
		Version: 5,
		Up: `CREATE TABLE IF NOT EXISTS tasks (
			id              TEXT PRIMARY KEY,
			agent_id        TEXT NOT NULL,
			operator_id     TEXT,
			command         TEXT NOT NULL,
			arguments       TEXT,
			status          TEXT NOT NULL DEFAULT 'pending',
			output          TEXT,
			error_message   TEXT,
			timeout_sec     INTEGER NOT NULL DEFAULT 60,
			created_at      TEXT NOT NULL DEFAULT (datetime('now')),
			sent_at         TEXT,
			completed_at    TEXT,
			duration_ms     INTEGER,
			FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
			FOREIGN KEY (operator_id) REFERENCES operators(id) ON DELETE SET NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tasks_agent_status ON tasks(agent_id, status);
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`,
	},
	{
		Version: 6,
		Up: `CREATE TABLE IF NOT EXISTS audit_logs (
			id              TEXT PRIMARY KEY,
			operator_id     TEXT,
			action          TEXT NOT NULL,
			resource_type   TEXT NOT NULL,
			resource_id     TEXT,
			details_json    TEXT,
			ip_address      TEXT,
			created_at      TEXT NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (operator_id) REFERENCES operators(id) ON DELETE SET NULL
		);
		CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at);
		CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action)`,
	},
	{
		Version: 7,
		Up: `CREATE TABLE IF NOT EXISTS schema_migrations (
			version         INTEGER PRIMARY KEY,
			applied_at      TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
	},
	{
		Version: 8,
		Up: `ALTER TABLE agents ADD COLUMN exe_name TEXT NOT NULL DEFAULT '';
ALTER TABLE agents ADD COLUMN remote_ip TEXT NOT NULL DEFAULT '';`,
	},
}

// ApplyMigrations runs all pending migrations in order.
func ApplyMigrations(db *sql.DB) error {
	// Ensure schema_migrations table exists first (it's migration version 7,
	// but we need it to track all migrations).
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version         INTEGER PRIMARY KEY,
		applied_at      TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	// Get current schema version
	var currentVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	// Apply pending migrations
	for _, m := range Migrations {
		if m.Version <= currentVersion {
			continue
		}

		_, err := db.Exec(m.Up)
		if err != nil {
			return fmt.Errorf("applying migration %d: %w", m.Version, err)
		}

		_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.Version)
		if err != nil {
			return fmt.Errorf("recording migration %d: %w", m.Version, err)
		}
	}

	return nil
}
