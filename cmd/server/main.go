// Package main is the entry point for the Go Fullstack Starter web server.
//
// This file sets up and runs the HTTP server with all middleware, routes, and handlers.
// It includes graceful shutdown support to properly close database connections and
// finish in-flight requests when the server is stopped.
//
// To run the server:
//
//	go run cmd/server/main.go
//
// Or using make:
//
//	make dev    # Development with hot reload
//	make build  # Build production binary
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"replace-me/internal/config"
	"replace-me/internal/database"
	"replace-me/internal/handlers"
	"replace-me/internal/logger"
	"replace-me/internal/middleware"

	"github.com/labstack/echo/v4"
)

func main() {
	// Load configuration from environment variables (and .env file in development).
	// See internal/config/config.go for available configuration options.
	cfg := config.Load()

	// Initialize the structured logger based on configuration.
	// In development: human-readable text output
	// In production: JSON output for log aggregation systems
	logger.Init(cfg.LogLevel, cfg.IsDevelopment())

	logger.Info("starting server",
		"port", cfg.Port,
		"environment", cfg.Environment,
	)

	// Connect to the PostgreSQL database.
	// Query logging is enabled in development to help debug queries.
	db, err := database.New(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		logger.Error("failed to connect to database", "error", err.Error())
		os.Exit(1)
	}

	// Create the Echo web server instance.
	// Echo is a high-performance, minimalist web framework for Go.
	e := echo.New()

	// Disable Echo's default banner and use our own logging
	e.HideBanner = true
	e.HidePort = true

	// Configure all middleware (logging, recovery, CORS, timeout, sessions, etc.)
	// See internal/middleware/middleware.go for details on each middleware.
	middleware.Setup(e, cfg)

	// Serve static files (CSS, JS, images) from the static directory.
	// Files are served at /static/* (e.g., /static/css/output.css)
	e.Static("/static", "static")

	// Initialize handlers with database connection.
	// Handlers delegate to services for business logic.
	h := handlers.New(db)

	// =========================================================================
	// Routes
	// =========================================================================
	// Define your application routes here.
	// Group related routes and apply middleware as needed.

	// Home page
	e.GET("/", h.Home)

	// Greeting demo - shows HTMX form handling
	e.POST("/greet", h.Greet)

	// Health check endpoint - useful for load balancers, Kubernetes probes,
	// and monitoring systems to verify the server is running.
	e.GET("/health", h.Health)

	// =========================================================================
	// Graceful Shutdown
	// =========================================================================
	// The server handles SIGINT (Ctrl+C) and SIGTERM (kill) signals gracefully.
	// When a signal is received:
	// 1. Stop accepting new connections
	// 2. Wait for in-flight requests to complete (up to 10 seconds)
	// 3. Close database connections
	// 4. Exit cleanly
	//
	// This prevents data corruption and ensures clients get proper responses.

	// Start server in a goroutine so it doesn't block signal handling
	go func() {
		addr := ":" + cfg.Port
		logger.Info("server listening", "addr", addr)

		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal (Ctrl+C or kill command)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("shutting down server", "signal", sig.String())

	// Create a deadline for shutdown (10 seconds should be enough for most requests)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Gracefully shutdown the Echo server
	// This waits for all in-flight requests to complete
	if err := e.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err.Error())
	}

	// Close database connection after server is shut down
	// This ensures no queries are in progress when we close
	if err := database.Close(db); err != nil {
		logger.Error("database close error", "error", err.Error())
	}

	logger.Info("server stopped")
}
