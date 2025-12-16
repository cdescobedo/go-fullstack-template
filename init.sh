#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_success() { echo -e "${GREEN}✓${NC} $1"; }
print_warning() { echo -e "${YELLOW}!${NC} $1"; }
print_error() { echo -e "${RED}✗${NC} $1"; }

# Usage
usage() {
    cat << EOF
Usage: ./init.sh <module-name> [options]

Initialize a new project from this template.

Arguments:
  module-name    Go module path (e.g., github.com/user/myapp or just myapp)

Options:
  --name         Display name for the app (default: derived from module name)
  --keep-git     Keep existing git history (default: reinitialize)
  -h, --help     Show this help message

Examples:
  ./init.sh myapp
  ./init.sh github.com/carlos/myapp
  ./init.sh myapp --name "My Cool App"
  ./init.sh myapp --keep-git

EOF
    exit 1
}

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -f "Makefile" ]; then
    print_error "Please run this script from the project root directory"
    exit 1
fi

# Check if already initialized
if ! grep -q "replace-me" go.mod 2>/dev/null; then
    print_error "Project appears to already be initialized (no 'replace-me' found in go.mod)"
    exit 1
fi

# Parse arguments
MODULE_NAME=""
APP_NAME=""
KEEP_GIT=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            ;;
        --name)
            APP_NAME="$2"
            shift 2
            ;;
        --keep-git)
            KEEP_GIT=true
            shift
            ;;
        -*)
            print_error "Unknown option: $1"
            usage
            ;;
        *)
            if [ -z "$MODULE_NAME" ]; then
                MODULE_NAME="$1"
            else
                print_error "Unexpected argument: $1"
                usage
            fi
            shift
            ;;
    esac
done

# Validate module name
if [ -z "$MODULE_NAME" ]; then
    print_error "Module name is required"
    usage
fi

# Derive short name from module (last segment of path)
SHORT_NAME=$(basename "$MODULE_NAME")

# Default app name if not provided
if [ -z "$APP_NAME" ]; then
    # Convert hyphenated name to title case
    APP_NAME=$(echo "$SHORT_NAME" | sed 's/-/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) tolower(substr($i,2))}1')
fi

echo ""
echo "Initializing new project:"
echo "  Module:       $MODULE_NAME"
echo "  Short name:   $SHORT_NAME"
echo "  Display name: $APP_NAME"
echo ""

# Detect OS for sed compatibility
if [[ "$OSTYPE" == "darwin"* ]]; then
    SED_INPLACE="sed -i ''"
else
    SED_INPLACE="sed -i"
fi

# Function to replace in files
replace_in_file() {
    local file="$1"
    local from="$2"
    local to="$3"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s|$from|$to|g" "$file"
    else
        sed -i "s|$from|$to|g" "$file"
    fi
}

# Replace module name in Go files
echo "Updating Go module and imports..."
find . -type f -name "*.go" -not -path "./vendor/*" | while read -r file; do
    replace_in_file "$file" "replace-me" "$MODULE_NAME"
done
print_success "Updated Go files"

# Replace in templ files
find . -type f -name "*.templ" | while read -r file; do
    replace_in_file "$file" "replace-me" "$MODULE_NAME"
done
print_success "Updated templ files"

# Update go.mod
replace_in_file "go.mod" "replace-me" "$MODULE_NAME"
print_success "Updated go.mod"

# Update compose.yaml
replace_in_file "compose.yaml" "replace-me" "$SHORT_NAME"
print_success "Updated compose.yaml"

# Update package.json
replace_in_file "package.json" "replace-me" "$SHORT_NAME"
print_success "Updated package.json"

# Update app name in error pages
replace_in_file "internal/middleware/middleware.go" "Go Fullstack Starter" "$APP_NAME"
print_success "Updated app display name"

# Clean up template files
echo ""
echo "Cleaning up template files..."

# Remove this init script
rm -f init.sh
print_success "Removed init.sh"

# Update README - replace template instructions with project-specific content
cat > README.md << EOF
# $APP_NAME

Built with Go, Echo, PostgreSQL, templ, Tailwind CSS, and HTMX.

## Quick Start

\`\`\`bash
# Install dependencies
make install

# Start database
make dbup

# Run migrations
make migrate-up

# Start development server
make dev
\`\`\`

Visit http://localhost:8090

## Commands

\`\`\`bash
make dev               # Start dev server with hot reload
make build             # Build production binary
make dbup              # Start PostgreSQL
make dbdown            # Stop PostgreSQL
make migrate-up        # Apply pending migrations
make migrate-down      # Rollback last migration
make migrate-status    # Show migration status
make migrate-create name=create_users  # Create new migration
\`\`\`

## Project Structure

\`\`\`
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
├── static/              # Static assets
└── templates/           # templ HTML templates
\`\`\`

## Configuration

Copy \`.env.example\` to \`.env\` and customize:

| Variable | Default | Description |
|----------|---------|-------------|
| \`PORT\` | 8080 | Server port |
| \`DATABASE_URL\` | local dev DB | PostgreSQL connection |
| \`ENVIRONMENT\` | development | development/production |
| \`SESSION_SECRET\` | dev key | Cookie encryption |
| \`LOG_LEVEL\` | info | debug/info/warn/error |
EOF
print_success "Updated README.md"

# Handle git
echo ""
if [ "$KEEP_GIT" = true ]; then
    print_warning "Keeping existing git history (--keep-git)"
else
    echo "Reinitializing git repository..."
    rm -rf .git
    git init -q
    git add -A
    git commit -q -m "Initial commit from go-fullstack-starter template"
    print_success "Initialized fresh git repository"
fi

# Regenerate templ files if templ is installed
echo ""
if command -v templ &> /dev/null; then
    echo "Regenerating templ files..."
    templ generate > /dev/null 2>&1
    print_success "Regenerated templ files"
else
    print_warning "templ not installed - run 'make install' then 'make templ-generate'"
fi

echo ""
echo -e "${GREEN}Done!${NC} Your project '$APP_NAME' is ready."
echo ""
echo "Next steps:"
echo "  1. make install    # Install dependencies"
echo "  2. make dbup       # Start PostgreSQL"
echo "  3. make migrate-up # Run migrations"
echo "  4. make dev        # Start development server"
echo ""
