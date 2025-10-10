package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// Migration represents a database migration
type Migration struct {
	Version     int       `json:"version"`
	Description string    `json:"description"`
	UpSQL       string    `json:"up_sql"`
	DownSQL     string    `json:"down_sql"`
	AppliedAt   *time.Time `json:"applied_at"`
}

// MigrationManager handles database schema migrations
type MigrationManager struct {
	db         *sql.DB
	migrations []Migration
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *sql.DB) *MigrationManager {
	return &MigrationManager{
		db:         db,
		migrations: make([]Migration, 0),
	}
}

// AddMigration adds a migration to the manager
func (mm *MigrationManager) AddMigration(version int, description, upSQL, downSQL string) {
	migration := Migration{
		Version:     version,
		Description: description,
		UpSQL:       upSQL,
		DownSQL:     downSQL,
	}
	mm.migrations = append(mm.migrations, migration)
}

// InitMigrationsTable creates the migrations tracking table
func (mm *MigrationManager) InitMigrationsTable() error {
	schema := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		description TEXT NOT NULL,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := mm.db.Exec(schema)
	return err
}

// GetCurrentVersion returns the current schema version
func (mm *MigrationManager) GetCurrentVersion() (int, error) {
	var version int
	err := mm.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		// If the table doesn't exist, return version 0
		if err.Error() == "no such table: schema_migrations" {
			return 0, nil
		}
		return 0, err
	}
	return version, nil
}

// GetAppliedMigrations returns all applied migrations
func (mm *MigrationManager) GetAppliedMigrations() ([]Migration, error) {
	query := `
		SELECT version, description, applied_at
		FROM schema_migrations
		ORDER BY version ASC
	`
	
	rows, err := mm.db.Query(query)
	if err != nil {
		// If the table doesn't exist, return empty slice
		if err.Error() == "no such table: schema_migrations" {
			return []Migration{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var applied []Migration
	for rows.Next() {
		var migration Migration
		var appliedAt time.Time
		
		err := rows.Scan(&migration.Version, &migration.Description, &appliedAt)
		if err != nil {
			return nil, err
		}
		
		migration.AppliedAt = &appliedAt
		applied = append(applied, migration)
	}

	return applied, nil
}

// Migrate runs all pending migrations up to target version (0 = latest)
func (mm *MigrationManager) Migrate(targetVersion int) error {
	currentVersion, err := mm.GetCurrentVersion()
	if err != nil {
		return err
	}

	// Determine target version
	if targetVersion == 0 {
		for _, m := range mm.migrations {
			if m.Version > targetVersion {
				targetVersion = m.Version
			}
		}
	}

	// Apply migrations
	for _, migration := range mm.migrations {
		if migration.Version > currentVersion && migration.Version <= targetVersion {
			if err := mm.applyMigration(migration); err != nil {
				return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
			}
		}
	}

	return nil
}

// Rollback rolls back migrations to target version
func (mm *MigrationManager) Rollback(targetVersion int) error {
	currentVersion, err := mm.GetCurrentVersion()
	if err != nil {
		return err
	}

	if targetVersion >= currentVersion {
		return fmt.Errorf("target version %d is not less than current version %d", targetVersion, currentVersion)
	}

	// Find migrations to rollback (in reverse order)
	for i := len(mm.migrations) - 1; i >= 0; i-- {
		migration := mm.migrations[i]
		if migration.Version > targetVersion && migration.Version <= currentVersion {
			if err := mm.rollbackMigration(migration); err != nil {
				return fmt.Errorf("failed to rollback migration %d: %w", migration.Version, err)
			}
		}
	}

	return nil
}

// applyMigration applies a single migration
func (mm *MigrationManager) applyMigration(migration Migration) error {
	tx, err := mm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(migration.UpSQL); err != nil {
		return err
	}

	// Record migration
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version, description) VALUES (?, ?)",
		migration.Version, migration.Description,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// rollbackMigration rolls back a single migration
func (mm *MigrationManager) rollbackMigration(migration Migration) error {
	tx, err := mm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute rollback SQL
	if _, err := tx.Exec(migration.DownSQL); err != nil {
		return err
	}

	// Remove migration record
	if _, err := tx.Exec(
		"DELETE FROM schema_migrations WHERE version = ?",
		migration.Version,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// ValidateMigrations checks that migrations are properly ordered and complete
func (mm *MigrationManager) ValidateMigrations() error {
	versions := make(map[int]bool)
	
	for _, migration := range mm.migrations {
		if versions[migration.Version] {
			return fmt.Errorf("duplicate migration version: %d", migration.Version)
		}
		versions[migration.Version] = true
		
		if migration.UpSQL == "" {
			return fmt.Errorf("migration %d missing up SQL", migration.Version)
		}
		if migration.DownSQL == "" {
			return fmt.Errorf("migration %d missing down SQL", migration.Version)
		}
	}
	
	return nil
}