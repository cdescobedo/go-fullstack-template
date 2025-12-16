// Package middleware provides HTTP middleware for the Echo web framework.
//
// Middleware are functions that wrap HTTP handlers to add cross-cutting functionality
// like logging, authentication, error handling, etc. They execute in the order they
// are registered and can modify requests/responses or short-circuit the chain.
//
// This package includes:
//   - Request logging with structured output
//   - Panic recovery with error logging
//   - Request ID generation for tracing
//   - CORS handling for cross-origin requests
//   - Request timeout to prevent hanging requests
//   - Custom error handling with pretty error pages
//   - Session/flash message support
//
// Usage:
//
//	e := echo.New()
//	middleware.Setup(e, cfg)
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"replace-me/internal/config"
	"replace-me/internal/logger"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// sessionStore is the global session store for flash messages and user sessions.
// It uses encrypted cookies to store session data securely on the client side.
var sessionStore *sessions.CookieStore

// SessionName is the name of the session cookie.
// Change this if you want a different cookie name in the browser.
const SessionName = "session"

// Setup configures all middleware for the Echo instance.
// Middleware are applied in order, so the sequence matters:
//  1. RequestID - Adds unique ID to each request for tracing
//  2. Logger - Logs request details (needs request ID to be set first)
//  3. Recover - Catches panics and prevents server crashes
//  4. Timeout - Cancels requests that take too long
//  5. CORS - Handles cross-origin requests
//  6. Session - Makes session available to handlers
//  7. Gzip - Compresses responses (production only)
func Setup(e *echo.Echo, cfg *config.Config) {
	// Initialize the session store with the secret key from config.
	// CookieStore encrypts session data and stores it in a browser cookie.
	// This is simpler than server-side sessions (no Redis/DB needed) but
	// has a 4KB size limit and sends data on every request.
	sessionStore = sessions.NewCookieStore([]byte(cfg.SessionSecret))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,      // Prevents JavaScript access (XSS protection)
		Secure:   cfg.IsProduction(), // HTTPS only in production
		SameSite: http.SameSiteLaxMode, // CSRF protection
	}

	// Request ID middleware generates a unique ID for each request.
	// This ID is added to logs and response headers, making it easy to
	// trace a request through the system and correlate logs.
	e.Use(middleware.RequestID())

	// Custom request logger using our structured logger.
	// Logs method, path, status, latency, and other useful info.
	e.Use(requestLoggerMiddleware())

	// Recover middleware catches panics in handlers and converts them to errors.
	// Without this, a panic would crash the entire server. Instead, we log the
	// panic with stack trace and return a 500 error to the client.
	e.Use(recoverMiddleware())

	// Timeout middleware cancels requests that exceed the configured duration.
	// This prevents slow handlers from consuming resources indefinitely.
	// The handler receives a cancelled context and should check ctx.Done().
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: cfg.RequestTimeout,
		Skipper: func(c echo.Context) bool {
			// Skip timeout for certain paths if needed (e.g., file uploads, SSE)
			// return strings.HasPrefix(c.Path(), "/upload")
			return false
		},
	}))

	// CORS middleware handles Cross-Origin Resource Sharing.
	// This is required when your frontend is served from a different domain
	// than your API (e.g., frontend on localhost:3000, API on localhost:8080).
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: cfg.CORSAllowedOrigins,
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			"X-Request-ID",
			"HX-Request", // HTMX header
			"HX-Current-URL",
			"HX-Target",
			"HX-Trigger",
		},
		AllowCredentials: true, // Allow cookies in cross-origin requests
		MaxAge:           86400, // Cache preflight response for 24 hours
	}))

	// Session middleware makes the session store available to handlers.
	// Handlers can then use GetSession() to read/write session data.
	e.Use(sessionMiddleware())

	// Gzip compression reduces response size by 70-90% for text content.
	// Only enabled in production to avoid slowing down development.
	// The browser automatically decompresses the response.
	if cfg.IsProduction() {
		e.Use(middleware.Gzip())
	}

	// Set custom error handler for pretty error pages
	e.HTTPErrorHandler = customErrorHandler(cfg)
}

// requestLoggerMiddleware returns a middleware that logs HTTP requests using structured logging.
// Each log entry includes: method, path, status, latency, request_id, client_ip, user_agent.
func requestLoggerMiddleware() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:    true,
		LogURI:       true,
		LogStatus:    true,
		LogLatency:   true,
		LogRequestID: true,
		LogRemoteIP:  true,
		LogUserAgent: true,
		LogError:     true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			// Build log entry with request details
			args := []any{
				"method", v.Method,
				"path", v.URI,
				"status", v.Status,
				"latency", v.Latency.String(),
				"request_id", v.RequestID,
				"ip", v.RemoteIP,
			}

			// Add error if present
			if v.Error != nil {
				args = append(args, "error", v.Error.Error())
			}

			// Log at appropriate level based on status code
			switch {
			case v.Status >= 500:
				logger.Error("request failed", args...)
			case v.Status >= 400:
				logger.Warn("request error", args...)
			default:
				logger.Info("request completed", args...)
			}

			return nil
		},
	})
}

// recoverMiddleware returns a middleware that recovers from panics.
// When a panic occurs, it logs the error with stack trace and returns a 500 error.
func recoverMiddleware() echo.MiddlewareFunc {
	return middleware.RecoverWithConfig(middleware.RecoverConfig{
		// LogErrorFunc is called when a panic is recovered
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			logger.Error("panic recovered",
				"error", err.Error(),
				"stack", string(stack),
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
				"path", c.Request().URL.Path,
			)
			return err
		},
	})
}

// sessionMiddleware returns a middleware that initializes the session for each request.
// The session is stored in the Echo context and can be retrieved with GetSession().
func sessionMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get or create session for this request
			session, err := sessionStore.Get(c.Request(), SessionName)
			if err != nil {
				// Session decode error (e.g., invalid signature) - create new session
				logger.Warn("session decode error, creating new session", "error", err.Error())
				session, _ = sessionStore.New(c.Request(), SessionName)
			}

			// Store session in context for handlers to access
			c.Set("session", session)

			// Call the next handler
			err = next(c)

			// Save session after handler completes (even on error)
			// This persists any changes made to the session
			if saveErr := session.Save(c.Request(), c.Response()); saveErr != nil {
				logger.Error("failed to save session", "error", saveErr.Error())
			}

			return err
		}
	}
}

// GetSession retrieves the session from the Echo context.
// Returns nil if the session middleware is not configured.
//
// Usage in handlers:
//
//	session := middleware.GetSession(c)
//	if session != nil {
//	    session.Values["user_id"] = 123
//	}
func GetSession(c echo.Context) *sessions.Session {
	session, ok := c.Get("session").(*sessions.Session)
	if !ok {
		return nil
	}
	return session
}

// Flash message types for styling
const (
	FlashSuccess = "success" // Green - operation succeeded
	FlashError   = "error"   // Red - operation failed
	FlashWarning = "warning" // Yellow - warning/caution
	FlashInfo    = "info"    // Blue - informational
)

// AddFlash adds a flash message to the session.
// Flash messages are shown once and then automatically removed.
// They're useful for showing success/error messages after form submissions.
//
// Parameters:
//   - c: Echo context
//   - flashType: One of FlashSuccess, FlashError, FlashWarning, FlashInfo
//   - message: The message to display to the user
//
// Usage:
//
//	middleware.AddFlash(c, middleware.FlashSuccess, "Book created successfully!")
//	return c.Redirect(http.StatusSeeOther, "/books")
func AddFlash(c echo.Context, flashType, message string) {
	session := GetSession(c)
	if session == nil {
		return
	}
	// Store as a structured flash with type and message
	session.AddFlash(message, flashType)
}

// FlashMessage represents a flash message with its type for styling.
type FlashMessage struct {
	Type    string // "success", "error", "warning", "info"
	Message string // The message content
}

// GetFlashes retrieves and clears all flash messages from the session.
// Call this in your template rendering to display messages to the user.
//
// Usage in handlers:
//
//	flashes := middleware.GetFlashes(c)
//	return pages.Home(books, flashes).Render(ctx, c.Response().Writer)
func GetFlashes(c echo.Context) []FlashMessage {
	session := GetSession(c)
	if session == nil {
		return nil
	}

	var messages []FlashMessage

	// Get flashes for each type
	for _, flashType := range []string{FlashSuccess, FlashError, FlashWarning, FlashInfo} {
		flashes := session.Flashes(flashType)
		for _, flash := range flashes {
			if msg, ok := flash.(string); ok {
				messages = append(messages, FlashMessage{
					Type:    flashType,
					Message: msg,
				})
			}
		}
	}

	return messages
}

// customErrorHandler returns an error handler that renders pretty error pages.
// In development, it shows detailed error information.
// In production, it shows user-friendly messages without technical details.
func customErrorHandler(cfg *config.Config) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		// Don't handle if response already committed
		if c.Response().Committed {
			return
		}

		// Extract HTTP error code and message
		code := http.StatusInternalServerError
		message := "Internal Server Error"

		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			if he.Message != nil {
				message = fmt.Sprintf("%v", he.Message)
			}
		} else if cfg.IsDevelopment() {
			// In development, show the actual error
			message = err.Error()
		}

		// Log the error with context
		requestID := c.Response().Header().Get(echo.HeaderXRequestID)
		if code >= 500 {
			logger.Error("http error",
				"code", code,
				"error", err.Error(),
				"request_id", requestID,
				"path", c.Request().URL.Path,
			)
		}

		// Check if client wants JSON (API request)
		if c.Request().Header.Get("Accept") == "application/json" ||
			c.Request().Header.Get("Content-Type") == "application/json" {
			c.JSON(code, map[string]any{
				"error":      message,
				"code":       code,
				"request_id": requestID,
			})
			return
		}

		// For HTMX requests, return a partial HTML error
		if c.Request().Header.Get("HX-Request") == "true" {
			c.HTML(code, fmt.Sprintf(`
				<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4" role="alert">
					<strong class="font-bold">Error %d</strong>
					<span class="block sm:inline">%s</span>
				</div>
			`, code, message))
			return
		}

		// Render full error page for browser requests
		errorPage := renderErrorPage(code, message, requestID, cfg.IsDevelopment())
		c.HTML(code, errorPage)
	}
}

// renderErrorPage generates an HTML error page.
// The page styling matches the application's design.
func renderErrorPage(code int, message, requestID string, isDevelopment bool) string {
	title := http.StatusText(code)
	if title == "" {
		title = "Error"
	}

	// Additional debug info for development
	debugInfo := ""
	if isDevelopment && requestID != "" {
		debugInfo = fmt.Sprintf(`
			<p class="text-sm text-gray-500 mt-4">Request ID: %s</p>
		`, requestID)
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>%d %s - Go Fullstack Starter</title>
	<link href="/static/css/output.css" rel="stylesheet">
</head>
<body class="min-h-screen bg-gray-100 flex items-center justify-center">
	<div class="text-center p-8">
		<h1 class="text-6xl font-bold text-gray-800 mb-4">%d</h1>
		<h2 class="text-2xl font-semibold text-gray-600 mb-4">%s</h2>
		<p class="text-gray-500 mb-8">%s</p>
		<a href="/" class="inline-block bg-gray-800 text-white px-6 py-3 rounded hover:bg-gray-700 transition-colors">
			Go Home
		</a>
		%s
	</div>
</body>
</html>
`, code, title, code, title, message, debugInfo)
}

// WithTimeout wraps a handler function with a custom timeout.
// Use this for handlers that need longer or shorter timeouts than the default.
//
// Usage:
//
//	e.POST("/upload", middleware.WithTimeout(h.Upload, 5*time.Minute))
func WithTimeout(h echo.HandlerFunc, timeout time.Duration) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), timeout)
		defer cancel()
		c.SetRequest(c.Request().WithContext(ctx))
		return h(c)
	}
}
