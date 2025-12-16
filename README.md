# Go Fullstack Starter

A production-ready starter template for building fullstack Go web applications. Designed for solo developers who want a solid foundation without the complexity of microservices.

## Features

- **[Echo](https://echo.labstack.com/)** - High-performance web framework
- **[Bun ORM](https://bun.uptrace.dev/)** - Lightweight ORM with type-safe queries
- **[PostgreSQL](https://www.postgresql.org/)** - Reliable relational database
- **[templ](https://templ.guide/)** - Type-safe HTML templates
- **[Tailwind CSS](https://tailwindcss.com/)** - Utility-first CSS framework
- **[HTMX](https://htmx.org/)** - Server-driven interactivity
- **[Alpine.js](https://alpinejs.dev/)** - Client-side reactivity
- **[Air](https://github.com/air-verse/air)** - Live reload for development

### Built-in Functionality

- Graceful shutdown with proper cleanup
- Structured logging (text in dev, JSON in prod)
- Request logging with timing and request IDs
- CORS support for API access
- Request timeout protection
- Session management with flash messages
- Custom error pages (404, 500)
- Health check endpoint
- Database query logging (development)
- Environment-based configuration

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+ (for Tailwind)
- Docker (for PostgreSQL)

### Option 1: GitHub Template (Recommended)

1. Click **"Use this template"** on GitHub
2. Clone your new repository
3. Run the init script:

```bash
cd myapp
./init.sh myapp
# Or with a full module path:
./init.sh github.com/youruser/myapp
# Or with a custom display name:
./init.sh myapp --name "My Cool App"
```

### Option 2: Clone Directly

```bash
git clone https://github.com/yourusername/go-fullstack-starter myapp
cd myapp
./init.sh myapp
```

The init script handles:
- Updating go.mod and all import paths
- Renaming Docker containers and volumes
- Updating package.json
- Reinitializing git history
- Regenerating templ files

### Start Development

```bash
make install    # Install dependencies (air, templ, tailwindcss)
make dbup       # Start PostgreSQL
make migrate-up # Run migrations
make dev        # Start dev server with hot reload
```

Visit http://localhost:8090

### Configuration

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | Server port |
| `DATABASE_URL` | local dev DB | PostgreSQL connection string |
| `ENVIRONMENT` | development | development or production |
| `SESSION_SECRET` | dev key | Cookie encryption (change in prod!) |
| `LOG_LEVEL` | info | debug, info, warn, error |
| `REQUEST_TIMEOUT` | 30s | Max request duration |
| `CORS_ALLOWED_ORIGINS` | * | Allowed origins (comma-separated) |

## Project Structure

```
├── cmd/
│   ├── server/          # Main application entry point
│   └── migrate/         # Database migration CLI
├── internal/
│   ├── config/          # Configuration loading
│   ├── database/        # Database connection
│   ├── handlers/        # HTTP request handlers
│   ├── logger/          # Structured logging
│   ├── middleware/      # HTTP middleware
│   ├── models/          # Database models (Bun)
│   └── services/        # Business logic
├── migrations/          # SQL migration files
├── static/              # Static assets (CSS, JS, images)
├── templates/
│   ├── components/      # Reusable UI components
│   ├── layouts/         # Page layouts
│   └── pages/           # Full page templates
└── Makefile            # Development commands
```

## Development

### Commands

```bash
make help              # Show all available commands
make dev               # Start dev server with hot reload
make build             # Build production binary
make dbup              # Start PostgreSQL
make dbdown            # Stop PostgreSQL

# Migrations
make migrate-up        # Apply pending migrations
make migrate-down      # Rollback last migration
make migrate-status    # Show migration status
make migrate-create name=create_users  # Create new migration

# Testing
go test ./...          # Run all tests
go test ./... -v       # Verbose output
go test ./... -cover   # With coverage
```

### Adding a New Feature

1. **Create the model** (`internal/models/`)
   ```go
   type User struct {
       bun.BaseModel `bun:"table:users,alias:u"`
       ID        int64     `bun:"id,pk,autoincrement"`
       Email     string    `bun:"email,notnull,unique"`
       CreatedAt time.Time `bun:"created_at,notnull,default:now()"`
   }
   ```

2. **Create a migration**
   ```bash
   make migrate-create name=create_users
   # Edit migrations/*.up.sql and migrations/*.down.sql
   make migrate-up
   ```

3. **Create the service** (`internal/services/`)
   ```go
   type UserService struct { db *bun.DB }

   func (s *UserService) Create(ctx context.Context, email string) (*models.User, error) {
       // Validation and business logic
   }
   ```

4. **Create the handler** (`internal/handlers/`)
   ```go
   func (h *Handlers) CreateUser(c echo.Context) error {
       // Parse request, call service, render response
   }
   ```

5. **Add the route** (`cmd/server/main.go`)
   ```go
   e.POST("/users", h.CreateUser)
   ```

6. **Create templates** (`templates/`)

### Flash Messages

Show one-time notifications after actions:

```go
// In handler after successful action
middleware.AddFlash(c, middleware.FlashSuccess, "User created!")
return c.Redirect(http.StatusSeeOther, "/users")

// In handler rendering the page
flashes := middleware.GetFlashes(c)
return pages.Users(users, flashes).Render(ctx, c.Response().Writer)
```

### HTMX Integration

HTMX is included in the base layout. Example (see `/greet` handler):

```html
<!-- Form that updates without page reload -->
<form hx-post="/greet" hx-target="#greeting" hx-swap="outerHTML">
    <input name="name" type="text">
    <button type="submit">Say Hello</button>
</form>
<div id="greeting">Result appears here</div>
```

The handler returns HTML fragments for HTMX requests:

```go
func (h *Handlers) Greet(c echo.Context) error {
    name := c.FormValue("name")

    // Return HTML fragment for HTMX
    if c.Request().Header.Get("HX-Request") == "true" {
        return c.HTML(200, fmt.Sprintf(`<div id="greeting">Hello, %s!</div>`, name))
    }

    // Regular form submission
    middleware.AddFlash(c, middleware.FlashSuccess, "Hello, "+name)
    return c.Redirect(http.StatusSeeOther, "/")
}
```

### Alpine.js Integration

Alpine.js handles client-side interactivity without server round-trips:

```html
<!-- Toggle -->
<div x-data="{ open: false }">
    <button @click="open = !open">Toggle</button>
    <div x-show="open">Content here</div>
</div>

<!-- Dropdown -->
<div x-data="{ open: false }" @click.outside="open = false">
    <button @click="open = !open">Menu</button>
    <div x-show="open" x-transition>
        <a href="#">Option 1</a>
        <a href="#">Option 2</a>
    </div>
</div>

<!-- Form validation -->
<form x-data="{ email: '' }">
    <input x-model="email" type="email">
    <span x-show="email && !email.includes('@')">Invalid email</span>
</form>
```

**When to use which:**
- **HTMX**: Data needs to come from server (forms, search, pagination)
- **Alpine.js**: Pure UI state (dropdowns, tabs, modals, toggles)

## Deployment

### Build for Production

```bash
make build
# Binary at ./bin/server
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
COPY --from=builder /app/server /server
COPY --from=builder /app/static /static
EXPOSE 8080
CMD ["/server"]
```

### Environment Variables for Production

```bash
ENVIRONMENT=production
DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=require
SESSION_SECRET=$(openssl rand -base64 32)
LOG_LEVEL=info
CORS_ALLOWED_ORIGINS=https://myapp.com
```

## Architecture Decisions

**Why Echo?** Lightweight, fast, excellent middleware support. Good balance between features and simplicity.

**Why Bun?** Type-safe query builder that's lighter than GORM but more ergonomic than raw SQL.

**Why templ?** Type-safe templates with Go syntax. Catches errors at compile time. Better IDE support than text/template.

**Why HTMX + Alpine.js?** HTMX handles server interactions (forms, partial updates), Alpine.js handles client-side state (dropdowns, toggles, modals). Together they cover most UI needs without SPA complexity.

**Why Cookie Sessions?** Simpler than server-side sessions (no Redis needed). Good enough for most applications.

## Resources

- [Echo Documentation](https://echo.labstack.com/docs)
- [Bun Documentation](https://bun.uptrace.dev/)
- [templ Guide](https://templ.guide/)
- [HTMX Documentation](https://htmx.org/docs/)
- [Alpine.js Documentation](https://alpinejs.dev/start-here)
- [Tailwind CSS Documentation](https://tailwindcss.com/docs)

## License

MIT
