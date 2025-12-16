// Package config handles application configuration through environment variables.
//
// Configuration is loaded from environment variables with sensible defaults for local development.
// In development, variables can be set in a .env file which is automatically loaded.
//
// Environment Variables:
//   - PORT: HTTP server port (default: "8080")
//   - DATABASE_URL: PostgreSQL connection string (default: local dev database)
//   - ENVIRONMENT: "development" or "production" (default: "development")
//   - SESSION_SECRET: Secret key for session encryption (default: insecure dev key)
//   - CORS_ALLOWED_ORIGINS: Comma-separated list of allowed origins (default: "*")
//   - REQUEST_TIMEOUT: Request timeout duration (default: "30s")
//   - LOG_LEVEL: Logging level - debug, info, warn, error (default: "info")
//
// Usage:
//
//	cfg := config.Load()
//	fmt.Println(cfg.Port)           // "8080"
//	fmt.Println(cfg.IsDevelopment()) // true
package config

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration values.
// Values are populated from environment variables when Load() is called.
type Config struct {
	// Port is the HTTP server port (e.g., "8080")
	Port string

	// DatabaseURL is the PostgreSQL connection string.
	// Format: postgres://user:password@host:port/dbname?sslmode=disable
	DatabaseURL string

	// Environment is either "development" or "production".
	// Affects logging verbosity, error details, and middleware behavior.
	Environment string

	// SessionSecret is used to encrypt session cookies.
	// IMPORTANT: In production, set this to a random 32+ character string.
	// You can generate one with: openssl rand -base64 32
	SessionSecret string

	// CORSAllowedOrigins is a list of origins allowed to make cross-origin requests.
	// Use ["*"] to allow all origins (not recommended for production with credentials).
	CORSAllowedOrigins []string

	// RequestTimeout is the maximum duration for processing a request.
	// Requests exceeding this duration will be cancelled.
	RequestTimeout time.Duration

	// LogLevel controls the verbosity of logging.
	// Valid values: "debug", "info", "warn", "error"
	LogLevel string
}

// Load reads configuration from environment variables.
// In development, it first loads variables from a .env file if present.
// Missing variables use sensible defaults suitable for local development.
//
// The .env file is optional and is typically NOT committed to version control.
// See .env.example for a template of available variables.
func Load() *Config {
	// Load .env file if it exists. This is a no-op in production where
	// environment variables are set directly (e.g., via Docker, Kubernetes).
	// The error is intentionally ignored - missing .env is fine in production.
	if err := godotenv.Load(); err != nil {
		// Only log in development to avoid noise in production
		if os.Getenv("ENVIRONMENT") == "" || os.Getenv("ENVIRONMENT") == "development" {
			log.Println("No .env file found, using environment variables and defaults")
		}
	}

	// Parse request timeout with fallback
	timeout, err := time.ParseDuration(getEnv("REQUEST_TIMEOUT", "30s"))
	if err != nil {
		timeout = 30 * time.Second
	}

	// Parse CORS origins - split comma-separated string into slice
	corsOrigins := strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "*"), ",")
	for i := range corsOrigins {
		corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
	}

	return &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/testdb?sslmode=disable"),
		Environment:        getEnv("ENVIRONMENT", "development"),
		SessionSecret:      getEnv("SESSION_SECRET", "dev-secret-key-change-in-production-123"),
		CORSAllowedOrigins: corsOrigins,
		RequestTimeout:     timeout,
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}
}

// IsDevelopment returns true if running in development mode.
// Use this to conditionally enable development-only features like
// detailed error messages, query logging, or debug endpoints.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode.
// Use this to conditionally enable production-only features like
// response compression, stricter security headers, or error aggregation.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// getEnv retrieves an environment variable or returns a fallback value.
// This is a helper function to provide defaults for missing variables.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
