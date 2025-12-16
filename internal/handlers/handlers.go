// Package handlers contains HTTP request handlers for the web application.
//
// Handlers are the entry point for HTTP requests. They:
// 1. Parse and validate request data
// 2. Perform business logic (or delegate to services for complex logic)
// 3. Render responses (HTML templates or JSON)
//
// Architecture:
//
//	HTTP Request → Handler → (optional) Service → Database
//	                  ↓
//	              Response (HTML/JSON)
//
// Handlers should be thin - complex logic belongs in services.
// They handle HTTP concerns: parsing requests, setting headers, rendering responses.
//
// Usage:
//
//	h := handlers.New(db)
//	e.GET("/", h.Home)
//	e.GET("/health", h.Health)
package handlers

import (
	"github.com/uptrace/bun"
)

// Handlers holds dependencies needed by HTTP handlers.
// Using a struct allows for easy dependency injection and testing.
type Handlers struct {
	// db is available for handlers that need database access.
	// For complex applications, inject services instead of using db directly.
	db *bun.DB
}

// New creates a new Handlers instance with the given database connection.
//
// Example:
//
//	db, _ := database.New(cfg.DatabaseURL, true)
//	h := handlers.New(db)
//	e.GET("/", h.Home)
func New(db *bun.DB) *Handlers {
	return &Handlers{
		db: db,
	}
}
