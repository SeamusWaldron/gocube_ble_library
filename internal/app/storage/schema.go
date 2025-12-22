package storage

import (
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed migrations/001_initial.sql
var migration001 string

//go:embed migrations/002_new_phases.sql
var migration002 string

//go:embed migrations/003_add_scramble.sql
var migration003 string

//go:embed migrations/004_orientations.sql
var migration004 string

// migrations is an ordered list of migration SQL statements.
var migrations = []struct {
	version int
	sql     string
}{
	{1, migration001},
	{2, migration002},
	{3, migration003},
	{4, migration004},
}

// applyMigrations applies all pending migrations.
func applyMigrations(db *sql.DB) error {
	// Get current version
	currentVersion := 0

	// Check if schema_version table exists
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name='schema_version'
	`).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check schema version table: %w", err)
	}

	if count > 0 {
		err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
		if err != nil {
			return fmt.Errorf("failed to get current version: %w", err)
		}
	}

	// Apply pending migrations
	for _, m := range migrations {
		if m.version <= currentVersion {
			continue
		}

		_, err := db.Exec(m.sql)
		if err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", m.version, err)
		}
	}

	return nil
}

// InitSchema initializes the database schema (alias for MigrateUp).
func InitSchema(db *sql.DB) error {
	return applyMigrations(db)
}
