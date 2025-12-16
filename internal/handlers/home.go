package handlers

import (
	"fmt"
	"net/http"
	"time"

	"replace-me/internal/database"
	"replace-me/internal/middleware"
	"replace-me/templates/pages"

	"github.com/labstack/echo/v4"
)

// Home renders the home page.
// This demonstrates the basic request flow: handler → template rendering.
//
// Route: GET /
func (h *Handlers) Home(c echo.Context) error {
	// Get any flash messages to display
	flashes := middleware.GetFlashes(c)

	// Render the home page template
	return pages.Home(flashes).Render(c.Request().Context(), c.Response().Writer)
}

// Greet handles the greeting form submission.
// This demonstrates:
//   - Form data parsing
//   - Flash messages
//   - HTMX partial responses
//
// Route: POST /greet
func (h *Handlers) Greet(c echo.Context) error {
	name := c.FormValue("name")
	if name == "" {
		name = "World"
	}

	// For HTMX requests, return just the greeting HTML fragment
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, fmt.Sprintf(
			"Hello, %s! 👋",
			name,
		))
	}

	// For regular form submissions, use flash message and redirect
	middleware.AddFlash(c, middleware.FlashSuccess, fmt.Sprintf("Hello, %s!", name))
	return c.Redirect(http.StatusSeeOther, "/")
}

// Health handles health check requests.
// It verifies the server is running and can connect to the database.
//
// Route: GET /health
//
// Returns JSON:
//
//	{"status": "healthy", "database": "connected", "timestamp": "..."}
//
// Status codes:
//   - 200: Server is healthy
//   - 503: Server is unhealthy (database connection failed)
//
// Use this endpoint for:
//   - Kubernetes liveness/readiness probes
//   - Load balancer health checks
//   - Monitoring systems
func (h *Handlers) Health(c echo.Context) error {
	ctx := c.Request().Context()

	// Check database connectivity
	dbStatus := "connected"
	if err := database.HealthCheck(ctx, h.db); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status":    "unhealthy",
			"database":  "disconnected",
			"error":     err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":    "healthy",
		"database":  dbStatus,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
