// Package logger provides structured logging using Go's standard library slog.
//
// This package wraps slog to provide a consistent logging interface across the application.
// It supports different log levels and formats based on the environment.
//
// Features:
//   - Structured logging with key-value pairs
//   - JSON output for production (machine-readable)
//   - Text output for development (human-readable)
//   - Configurable log levels (debug, info, warn, error)
//   - Request context integration
//
// Usage:
//
//	// Initialize once at startup
//	logger.Init("info", true) // level, isDevelopment
//
//	// Use throughout the application
//	logger.Info("user logged in", "user_id", 123, "ip", "192.168.1.1")
//	logger.Error("database error", "err", err, "query", "SELECT * FROM users")
//	logger.Debug("request details", "headers", headers) // Only shown if level is debug
//
// With request context:
//
//	logger.InfoContext(ctx, "processing request", "path", "/api/users")
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// logger is the global logger instance.
// It's initialized with defaults and can be reconfigured with Init().
var logger *slog.Logger

// init sets up a default logger that writes to stderr.
// This ensures logging works even if Init() is not called.
func init() {
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Init configures the global logger based on environment and log level.
//
// Parameters:
//   - level: Log level string ("debug", "info", "warn", "error")
//   - isDevelopment: If true, uses human-readable text format; if false, uses JSON
//
// In development mode:
//   - Uses colorized text output for easy reading in terminals
//   - Shows source file and line numbers for debug level
//
// In production mode:
//   - Uses JSON format for easy parsing by log aggregators (e.g., ELK, Datadog)
//   - Omits debug-level source information to reduce log size
func Init(level string, isDevelopment bool) {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
		// AddSource adds file:line to log entries - useful for debugging
		// but adds overhead, so only enable for debug level in development
		AddSource: isDevelopment && logLevel == slog.LevelDebug,
	}

	var handler slog.Handler
	if isDevelopment {
		// Text handler is easier to read in development terminals
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		// JSON handler is better for production log aggregation systems
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger = slog.New(handler)

	// Also set as the default logger for any code using slog directly
	slog.SetDefault(logger)
}

// Debug logs a message at debug level.
// Debug logs are typically only enabled during development or troubleshooting.
// Use for detailed information like variable values, function entry/exit, etc.
//
// Example:
//
//	logger.Debug("processing item", "item_id", 42, "status", "pending")
func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// Info logs a message at info level.
// Info logs record normal application events that are useful for monitoring.
// Use for significant events like server startup, user actions, job completions.
//
// Example:
//
//	logger.Info("server started", "port", 8080, "env", "production")
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// Warn logs a message at warning level.
// Warning logs indicate potential issues that don't prevent operation but need attention.
// Use for deprecated features, approaching limits, recoverable errors.
//
// Example:
//
//	logger.Warn("rate limit approaching", "current", 950, "limit", 1000)
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Error logs a message at error level.
// Error logs indicate failures that need immediate attention.
// Use for exceptions, failed operations, or conditions that prevent normal operation.
//
// Example:
//
//	logger.Error("database connection failed", "err", err, "host", "db.example.com")
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

// DebugContext logs a debug message with request context.
// The context can carry request-specific values like request ID, user ID, etc.
func DebugContext(ctx context.Context, msg string, args ...any) {
	logger.DebugContext(ctx, msg, args...)
}

// InfoContext logs an info message with request context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	logger.InfoContext(ctx, msg, args...)
}

// WarnContext logs a warning message with request context.
func WarnContext(ctx context.Context, msg string, args ...any) {
	logger.WarnContext(ctx, msg, args...)
}

// ErrorContext logs an error message with request context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	logger.ErrorContext(ctx, msg, args...)
}

// With returns a new logger with the given attributes added to every log entry.
// This is useful for adding common context to all logs in a request handler.
//
// Example:
//
//	reqLogger := logger.With("request_id", requestID, "user_id", userID)
//	reqLogger.Info("processing started")
//	reqLogger.Info("processing completed") // Both logs have request_id and user_id
func With(args ...any) *slog.Logger {
	return logger.With(args...)
}

// GetLogger returns the underlying slog.Logger for advanced use cases.
// Prefer using the package-level functions (Info, Error, etc.) when possible.
func GetLogger() *slog.Logger {
	return logger
}
