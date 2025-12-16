package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"replace-me/internal/config"
	"replace-me/internal/database"
	"replace-me/migrations"

	"github.com/uptrace/bun/migrate"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg := config.Load()
	db, err := database.New(cfg.DatabaseURL, false)
	if err != nil {
		fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)

	migrator := migrate.NewMigrator(db, migrations.Migrations)
	ctx := context.Background()

	if err := migrator.Init(ctx); err != nil {
		fatalf("Failed to initialize migrator: %v", err)
	}

	cmd := os.Args[1]
	switch cmd {
	case "up":
		cmdUp(ctx, migrator)
	case "down":
		cmdDown(ctx, migrator)
	case "status":
		cmdStatus(ctx, migrator)
	case "create":
		cmdCreate(ctx, migrator)
	case "delete":
		cmdDelete(ctx, migrator)
	case "redo":
		cmdRedo(ctx, migrator)
	case "lock":
		cmdLock(ctx, migrator)
	case "unlock":
		cmdUnlock(ctx, migrator)
	default:
		fmt.Printf("Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: migrate <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  up       Apply all pending migrations")
	fmt.Println("  down     Rollback the last applied migration")
	fmt.Println("  redo     Rollback and re-apply the last migration")
	fmt.Println("  status   Show migration status")
	fmt.Println("  create   Create a new migration (usage: migrate create <name>)")
	fmt.Println("  delete   Delete an unapplied migration (usage: migrate delete <name>)")
	fmt.Println("  lock     Show migration lock status")
	fmt.Println("  unlock   Force unlock migrations (use with caution)")
}

func cmdUp(ctx context.Context, migrator *migrate.Migrator) {
	group, err := migrator.Migrate(ctx)
	if err != nil {
		fatalf("Migration failed: %v", err)
	}
	if group.IsZero() {
		fmt.Println("No new migrations to apply")
		return
	}
	fmt.Printf("Applied %d migration(s):\n", len(group.Migrations))
	for _, m := range group.Migrations {
		fmt.Printf("  ✓ %s\n", m.Name)
	}
}

func cmdDown(ctx context.Context, migrator *migrate.Migrator) {
	group, err := migrator.Rollback(ctx)
	if err != nil {
		fatalf("Rollback failed: %v", err)
	}
	if group.IsZero() {
		fmt.Println("No migrations to rollback")
		return
	}
	fmt.Printf("Rolled back %d migration(s):\n", len(group.Migrations))
	for _, m := range group.Migrations {
		fmt.Printf("  ↩ %s\n", m.Name)
	}
}

func cmdRedo(ctx context.Context, migrator *migrate.Migrator) {
	group, err := migrator.Rollback(ctx)
	if err != nil {
		fatalf("Rollback failed: %v", err)
	}
	if group.IsZero() {
		fmt.Println("No migrations to redo")
		return
	}
	fmt.Printf("Rolled back: %s\n", group.Migrations[0].Name)

	group, err = migrator.Migrate(ctx)
	if err != nil {
		fatalf("Re-apply failed: %v", err)
	}
	fmt.Printf("Re-applied: %s\n", group.Migrations[0].Name)
}

func cmdStatus(ctx context.Context, migrator *migrate.Migrator) {
	ms, err := migrator.MigrationsWithStatus(ctx)
	if err != nil {
		fatalf("Failed to get migration status: %v", err)
	}
	if len(ms) == 0 {
		fmt.Println("No migrations found")
		return
	}

	applied := 0
	pending := 0
	fmt.Println("Migrations:")
	for _, m := range ms {
		if m.MigratedAt.IsZero() {
			fmt.Printf("  ○ %s (pending)\n", m.Name)
			pending++
		} else {
			fmt.Printf("  ● %s (applied %s)\n", m.Name, m.MigratedAt.Format("2006-01-02 15:04:05"))
			applied++
		}
	}
	fmt.Printf("\nTotal: %d applied, %d pending\n", applied, pending)
}

func cmdCreate(ctx context.Context, migrator *migrate.Migrator) {
	if len(os.Args) < 3 {
		fatalf("Usage: migrate create <name>")
	}
	name := os.Args[2]

	files, err := migrator.CreateSQLMigrations(ctx, name)
	if err != nil {
		fatalf("Failed to create migration: %v", err)
	}
	fmt.Println("Created migration files:")
	for _, f := range files {
		fmt.Printf("  + %s\n", f.Path)
	}
}

func cmdDelete(ctx context.Context, migrator *migrate.Migrator) {
	if len(os.Args) < 3 {
		fatalf("Usage: migrate delete <name>")
	}
	name := os.Args[2]

	ms, err := migrator.MigrationsWithStatus(ctx)
	if err != nil {
		fatalf("Failed to get migration status: %v", err)
	}

	var found *migrate.Migration
	for i := range ms {
		if ms[i].Name == name {
			found = &ms[i]
			break
		}
	}
	if found == nil {
		fatalf("Migration not found: %s", name)
	}
	if !found.MigratedAt.IsZero() {
		fatalf("Cannot delete applied migration: %s\nRun 'migrate down' first to rollback", name)
	}

	// Get the migrations directory path relative to this source file
	migrationsDir := getMigrationsDir()

	upFile := filepath.Join(migrationsDir, name+".up.sql")
	downFile := filepath.Join(migrationsDir, name+".down.sql")

	if err := os.Remove(upFile); err != nil {
		fatalf("Failed to delete %s: %v", upFile, err)
	}
	fmt.Printf("Deleted %s\n", upFile)

	if err := os.Remove(downFile); err != nil {
		fatalf("Failed to delete %s: %v", downFile, err)
	}
	fmt.Printf("Deleted %s\n", downFile)
}

func cmdLock(ctx context.Context, migrator *migrate.Migrator) {
	// Try to acquire lock to check if it's available
	if err := migrator.Lock(ctx); err != nil {
		fmt.Printf("Lock status: LOCKED (another process holds the lock)\n")
		fmt.Println("If this is stuck, use 'migrate unlock' with caution")
		return
	}
	// We got the lock, release it
	migrator.Unlock(ctx)
	fmt.Println("Lock status: AVAILABLE")
}

func cmdUnlock(ctx context.Context, migrator *migrate.Migrator) {
	if err := migrator.Unlock(ctx); err != nil {
		fatalf("Failed to unlock: %v", err)
	}
	fmt.Println("Migration lock released")
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

// getMigrationsDir returns the absolute path to the migrations directory.
// It works regardless of the current working directory.
func getMigrationsDir() string {
	// First, try relative to current working directory (most common case)
	if _, err := os.Stat("migrations"); err == nil {
		abs, _ := filepath.Abs("migrations")
		return abs
	}

	// Fallback: find relative to this source file (for testing)
	_, thisFile, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	// Last resort: assume current directory
	return "migrations"
}
