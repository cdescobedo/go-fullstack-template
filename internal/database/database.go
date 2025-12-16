// Package database handles PostgreSQL database connections using Bun ORM.
//
// Bun is a lightweight ORM for Go that provides type-safe query building,
// migrations, and efficient scanning into structs. It sits on top of database/sql
// and provides a nicer API while still allowing raw SQL when needed.
//
// Features provided:
//   - Connection pooling (via database/sql)
//   - Query logging in development mode
//   - Graceful connection handling
//
// Usage:
//
//	db, err := database.New(cfg.DatabaseURL, cfg.IsDevelopment())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer database.Close(db)
//
// Query examples with Bun:
//
//	// Select all
//	var books []models.Book
//	db.NewSelect().Model(&books).Scan(ctx)
//
//	// Select with conditions
//	var book models.Book
//	db.NewSelect().Model(&book).Where("id = ?", id).Scan(ctx)
//
//	// Insert
//	book := &models.Book{Title: "...", Author: "..."}
//	db.NewInsert().Model(book).Exec(ctx)
//
//	// Update
//	db.NewUpdate().Model(book).WherePK().Exec(ctx)
//
//	// Delete
//	db.NewDelete().Model((*models.Book)(nil)).Where("id = ?", id).Exec(ctx)
package database

import (
	"context"
	"database/sql"
	"time"

	"replace-me/internal/logger"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// New creates a new database connection with the given DSN.
// If enableQueryLogging is true, all queries will be logged with their execution time.
//
// The DSN format is: postgres://user:password@host:port/dbname?sslmode=disable
//
// Connection pool settings can be adjusted after creation:
//
//	db.SetMaxOpenConns(25)      // Maximum open connections
//	db.SetMaxIdleConns(5)       // Maximum idle connections
//	db.SetConnMaxLifetime(time.Hour) // Maximum connection lifetime
func New(databaseURL string, enableQueryLogging bool) (*bun.DB, error) {
	// Create the underlying sql.DB connection using pgdriver.
	// pgdriver is a pure-Go PostgreSQL driver that doesn't require CGO.
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(databaseURL)))

	// Configure connection pool defaults.
	// These are reasonable defaults for a solo developer's application.
	// Adjust based on your expected load and database server capacity.
	sqldb.SetMaxOpenConns(25)              // Max connections to database
	sqldb.SetMaxIdleConns(5)               // Keep some connections ready
	sqldb.SetConnMaxLifetime(time.Hour)    // Recreate connections periodically

	// Wrap with Bun ORM using PostgreSQL dialect.
	// The dialect handles PostgreSQL-specific SQL syntax and features.
	db := bun.NewDB(sqldb, pgdialect.New())

	// Add query logging hook in development mode.
	// This logs every query with its execution time, which is invaluable
	// for debugging N+1 queries and slow queries during development.
	if enableQueryLogging {
		db.AddQueryHook(&queryLoggingHook{})
	}

	// Verify the connection works by pinging the database.
	// This catches configuration errors early rather than on first query.
	if err := db.Ping(); err != nil {
		return nil, err
	}

	logger.Info("database connected", "url", sanitizeDSN(databaseURL))

	return db, nil
}

// Close gracefully closes the database connection.
// Always defer this after creating the connection to ensure cleanup.
func Close(db *bun.DB) error {
	logger.Info("closing database connection")
	return db.Close()
}

// queryLoggingHook implements bun.QueryHook to log all database queries.
// This is only used in development mode to help debug queries.
type queryLoggingHook struct{}

// BeforeQuery is called before each query executes.
// We use it to record the start time in the query context.
func (h *queryLoggingHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

// AfterQuery is called after each query completes.
// We log the query, its duration, and any errors.
func (h *queryLoggingHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	duration := time.Since(event.StartTime)

	// Build log arguments
	args := []any{
		"query", event.Query,
		"duration", duration.String(),
	}

	// Add operation type for easier filtering
	if event.Model != nil {
		args = append(args, "model", event.Model)
	}

	// Log errors at error level, successful queries at debug level
	if event.Err != nil {
		args = append(args, "error", event.Err.Error())
		logger.Error("database query failed", args...)
	} else {
		// Only log successful queries at debug level to reduce noise
		logger.Debug("database query", args...)
	}
}

// sanitizeDSN removes the password from a database URL for safe logging.
// Example: postgres://user:password@host:5432/db -> postgres://user:***@host:5432/db
func sanitizeDSN(dsn string) string {
	// Simple approach: find :// and @ to locate credentials
	// For a production app, use url.Parse for proper handling
	start := -1
	end := -1
	colonCount := 0

	for i, c := range dsn {
		if c == ':' {
			colonCount++
			if colonCount == 2 {
				start = i + 1
			}
		}
		if c == '@' && start != -1 {
			end = i
			break
		}
	}

	if start != -1 && end != -1 && end > start {
		return dsn[:start] + "***" + dsn[end:]
	}
	return dsn
}

// HealthCheck verifies the database connection is working.
// Use this in health check endpoints to monitor database connectivity.
//
// Usage:
//
//	if err := database.HealthCheck(ctx, db); err != nil {
//	    return c.JSON(500, map[string]string{"status": "unhealthy"})
//	}
func HealthCheck(ctx context.Context, db *bun.DB) error {
	return db.PingContext(ctx)
}
