package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMigrationTestDB(t *testing.T) *sql.DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "migration_test.db")
	
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err, "Failed to create test database")
	
	return db
}

func TestNewMigrationManager(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	mm := NewMigrationManager(db)

	assert.NotNil(t, mm)
	assert.NotNil(t, mm.db)
	assert.Equal(t, 0, len(mm.migrations))
}

func TestMigrationManager_InitMigrationsTable(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	manager := NewMigrationManager(db)

	err := manager.InitMigrationsTable()
	assert.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Test idempotent behavior
	err = manager.InitMigrationsTable()
	assert.NoError(t, err)
}

func TestMigrationManager_AddMigration(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	manager := NewMigrationManager(db)

	manager.AddMigration(1, "Create users table", "CREATE TABLE users (id INTEGER);", "DROP TABLE users;")
	manager.AddMigration(2, "Create posts table", "CREATE TABLE posts (id INTEGER);", "DROP TABLE posts;")

	assert.Len(t, manager.migrations, 2)
	assert.Equal(t, 1, manager.migrations[0].Version)
	assert.Equal(t, "Create users table", manager.migrations[0].Description)
}

func TestMigrationManager_ValidateMigrations(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	manager := NewMigrationManager(db)

	// Test valid migrations
	manager.AddMigration(1, "Migration 1", "SQL", "SQL")
	manager.AddMigration(2, "Migration 2", "SQL", "SQL")
	err := manager.ValidateMigrations()
	assert.NoError(t, err)

	// Test duplicate versions
	manager.AddMigration(1, "Duplicate", "SQL", "SQL")
	err = manager.ValidateMigrations()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate migration version: 1")
}

func TestMigrationManager_GetCurrentVersion(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	manager := NewMigrationManager(db)

	// Test before table exists
	version, err := manager.GetCurrentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 0, version)

	// Initialize table
	err = manager.InitMigrationsTable()
	assert.NoError(t, err)

	// Test with empty table
	version, err = manager.GetCurrentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 0, version)

	// Add migration records
	_, err = db.Exec("INSERT INTO schema_migrations (version, description, applied_at) VALUES (1, 'First', datetime('now'))")
	assert.NoError(t, err)
	_, err = db.Exec("INSERT INTO schema_migrations (version, description, applied_at) VALUES (3, 'Third', datetime('now'))")
	assert.NoError(t, err)

	// Should return highest version
	version, err = manager.GetCurrentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 3, version)
}

func TestMigrationManager_GetAppliedMigrations(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	manager := NewMigrationManager(db)

	// Test before table exists
	migrations, err := manager.GetAppliedMigrations()
	assert.NoError(t, err)
	assert.Empty(t, migrations)

	// Initialize and populate table
	err = manager.InitMigrationsTable()
	assert.NoError(t, err)

	now := time.Now().UTC()
	_, err = db.Exec("INSERT INTO schema_migrations (version, description, applied_at) VALUES (1, 'First', ?)", now.Format(time.RFC3339))
	assert.NoError(t, err)

	// Get applied migrations
	migrations, err = manager.GetAppliedMigrations()
	assert.NoError(t, err)
	assert.Len(t, migrations, 1)
	assert.Equal(t, 1, migrations[0].Version)
	assert.Equal(t, "First", migrations[0].Description)
}

func TestMigrationManager_Migrate(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	manager := NewMigrationManager(db)
	err := manager.InitMigrationsTable()
	assert.NoError(t, err)

	manager.AddMigration(1, "Create users table", 
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);", 
		"DROP TABLE users;")
	
	manager.AddMigration(2, "Create posts table", 
		"CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER);", 
		"DROP TABLE posts;")

	err = manager.Migrate(0)
	assert.NoError(t, err)

	// Verify tables were created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='posts'").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify current version
	version, err := manager.GetCurrentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 2, version)
}

func TestMigrationManager_Rollback(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	manager := NewMigrationManager(db)
	err := manager.InitMigrationsTable()
	assert.NoError(t, err)

	manager.AddMigration(1, "Create users table", 
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);", 
		"DROP TABLE users;")
	
	manager.AddMigration(2, "Create posts table", 
		"CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER);", 
		"DROP TABLE posts;")

	// First migrate to latest
	err = manager.Migrate(0)
	assert.NoError(t, err)

	// Now rollback to version 1
	err = manager.Rollback(1)
	assert.NoError(t, err)

	// Verify posts table was dropped
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='posts'").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	// Verify users table still exists
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify current version
	version, err := manager.GetCurrentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 1, version)
}

func TestMigrationManager_TransactionSafety(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	manager := NewMigrationManager(db)
	err := manager.InitMigrationsTable()
	assert.NoError(t, err)

	manager.AddMigration(1, "Valid migration", 
		"CREATE TABLE users (id INTEGER PRIMARY KEY);", 
		"DROP TABLE users;")
	
	manager.AddMigration(2, "Invalid migration", 
		"INVALID SQL STATEMENT", 
		"DROP TABLE posts;")

	// Try to migrate - should fail on second migration
	err = manager.Migrate(0)
	assert.Error(t, err)

	// Since each migration is its own transaction, the first one should succeed
	currentVersion, err := manager.GetCurrentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 1, currentVersion)

	// Verify users table was created (first migration succeeded)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestMigrationManager_ErrorHandling(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	manager := NewMigrationManager(db)

	// Test migrate with valid migration and invalid SQL
	manager.AddMigration(1, "Invalid SQL", "INVALID SQL STATEMENT", "DROP TABLE test;")
	err := manager.InitMigrationsTable()
	assert.NoError(t, err)
	err = manager.Migrate(1)
	assert.Error(t, err) // Should error due to invalid SQL

	// Test rollback without any applied migrations
	err = manager.Rollback(0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target version 0 is not less than current version")
}

