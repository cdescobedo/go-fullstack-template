// Package migrations handles database schema migrations using Bun's migrate package.
//
// Migrations are SQL files stored in this directory with the naming convention:
//   - {timestamp}_{name}.up.sql   - Applied when migrating up
//   - {timestamp}_{name}.down.sql - Applied when rolling back
//
// Creating migrations:
//
//	make migrate-create name=create_users
//
// This creates two files:
//   - migrations/20240101120000_create_users.up.sql
//   - migrations/20240101120000_create_users.down.sql
//
// Running migrations:
//
//	make migrate-up      # Apply all pending migrations
//	make migrate-down    # Rollback the last migration
//	make migrate-status  # Show which migrations have been applied
//
// Example migration (up):
//
//	CREATE TABLE users (
//	    id BIGSERIAL PRIMARY KEY,
//	    email VARCHAR(255) NOT NULL UNIQUE,
//	    name VARCHAR(255) NOT NULL,
//	    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
//	    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
//	);
//
// Example migration (down):
//
//	DROP TABLE IF EXISTS users;
package migrations

import (
	"embed"
	"io/fs"

	"github.com/uptrace/bun/migrate"
)

// sqlMigrations embeds SQL files in this directory.
// Note: go:embed requires at least one matching file, so we keep a .gitkeep
// or the first real migration. The Discover call filters to only .sql files.
//
//go:embed *.sql
var sqlMigrations embed.FS

// Migrations is the migration collection used by the migrate command.
// It discovers all embedded SQL migrations on initialization.
var Migrations = migrate.NewMigrations()

func init() {
	// Check if any SQL files exist before discovering.
	// This handles the edge case where only .gitkeep exists.
	entries, err := fs.Glob(sqlMigrations, "*.sql")
	if err != nil {
		panic(err)
	}
	if len(entries) == 0 {
		return
	}

	if err := Migrations.Discover(sqlMigrations); err != nil {
		panic(err)
	}
}
