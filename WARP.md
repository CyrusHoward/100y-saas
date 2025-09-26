# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

100-Year SaaS is a minimalist, boring-on-purpose Go application designed for extreme durability and simplicity. The project follows the principles of protocols over frameworks, using HTTP, HTML, and SQL with a single-binary approach that can run for decades with minimal maintenance.

**Status**: This is a complete, production-ready SaaS platform with authentication, multi-tenancy, analytics, subscriptions, background jobs, rate limiting, and email notifications. All features are maintenance-free and use only SQLite + standard library.

## Getting Started

1. **Install Go**: Download and install Go 1.22 or later from https://golang.org/dl/
2. **Install dependencies**: `go mod tidy`
3. **Run the application**: 
   - On Windows: `.\run.ps1`
   - On Unix/Linux/macOS: `make run`
4. **Open your browser**: Navigate to http://localhost:8080
5. **Start adding items**: Use the simple web interface to create and manage items

## Architecture

This is a full-stack web application with a simple 3-tier architecture:

```
[Static HTML/CSS/JS] → [Caddy Proxy] → [Go HTTP Server] → [SQLite Database]
```

**Key Components:**
- **Go Server** (`cmd/server/main.go`): Single binary HTTP server with embedded static files and database migrations
- **SQLite Database** (`internal/db/schema.sql`): Single-file ACID database with idempotent migrations, includes all SaaS tables
- **Static Frontend** (`web/`): Vanilla HTML/CSS/JS with no build tools or frameworks
- **Caddy Proxy** (`Caddyfile`): Auto-HTTPS reverse proxy for production deployment
- **Docker Support**: Multi-stage Dockerfile using distroless base for minimal attack surface

**SaaS Features (Zero Maintenance):**
- **Authentication** (`internal/auth/`): User registration, login, session management with auto-cleanup
- **Multi-tenancy** (`internal/saas/`): Complete tenant isolation, subscription limits, role-based access
- **Analytics** (`internal/analytics/`): Usage tracking, reporting, real-time stats with automatic data rotation
- **Background Jobs** (`internal/jobs/`): SQLite-based job queue with retries and scheduled tasks
- **Rate Limiting** (`internal/http/`): In-memory token bucket algorithm with auto-cleanup
- **Email Service** (`internal/email/`): SMTP-based notifications with template system

## Essential Commands

### Development
```bash
# Run the application locally (Unix/Linux/macOS)
make run
# Alternative: DB_PATH=data/app.db APP_SECRET=local go run ./cmd/server

# Run on Windows PowerShell
.\run.ps1
# Alternative: $env:DB_PATH="data/app.db"; $env:APP_SECRET="local"; go run ./cmd/server

# Build production binary (Unix/Linux/macOS)
make build
# Alternative: CGO_ENABLED=0 go build -o bin/app ./cmd/server

# Build on Windows PowerShell
.\build.ps1
# Alternative: $env:CGO_ENABLED="0"; go build -o bin/app.exe ./cmd/server

# Format Go code
make fmt
# Alternative: gofmt -s -w .

# Clean up dependencies
make tidy
# Alternative: go mod tidy
```

### Docker Deployment
```bash
# Build Docker image
docker build -t 100y-saas .

# Run with persistent data volume
docker run -p 8080:8080 -v $(pwd)/data:/data 100y-saas
```

### Production with Caddy
```bash
# Run Caddy for auto-HTTPS (after building the Go binary)
caddy run --config Caddyfile
```

### Database Backups
```bash
# Manual backup
./backup.sh

# Or with custom paths
DB_PATH=data/app.db BACKUP_DIR=backups ./backup.sh
```

## Key Design Patterns

**Embedded Assets Pattern**: Static files and SQL migrations are embedded in the Go binary using `//go:embed`, eliminating deployment complexity.

**Idempotent Migrations**: Database schema changes use `CREATE TABLE IF NOT EXISTS` and `INSERT OR IGNORE` patterns to safely run on every startup.

**Zero-Dependency Frontend**: No build tools, bundlers, or frameworks - just vanilla HTML/CSS/JS that works everywhere.

**Single-Binary Deployment**: Everything needed to run the application is contained in one executable file.

**Environment-Based Configuration**: Uses environment variables with sensible defaults (DB_PATH, APP_SECRET).

## Development Guidelines

### Code Structure
- Keep all HTTP handlers in the main.go file for simplicity
- Use standard library types (`http.Handler`, `sql.DB`) over abstractions
- Embed all assets and migrations directly in the binary
- Prefer explicit SQL queries over ORMs

### Security Considerations
- HMAC-signed cookies for session management
- Security headers set by default (`X-Content-Type-Options`, `X-Frame-Options`, etc.)
- Content Security Policy restricts to same-origin resources
- Non-root user execution in Docker

### Data Export
- All data is exportable via `/export` endpoint (returns CSV)
- SQLite database is a single file that can be copied/backed up easily
- No vendor lock-in - uses standard SQL and file formats

## Environment Variables

- `DB_PATH`: Path to SQLite database file (default: `data/app.db`)
- `APP_SECRET`: Secret key for cookie signing (default: `change-me` - must change in production)

## Common Tasks

### Adding New API Endpoints
1. Add handler function following the `itemsHandler` pattern
2. Register route in main function: `mux.HandleFunc("/api/newroute", app.newHandler)`
3. Update database schema in `internal/db/schema.sql` if needed

### Adding New Database Tables
1. Add `CREATE TABLE IF NOT EXISTS` statement to `schema.sql`
2. Update the `meta` table schema version if needed
3. Add corresponding Go structs and handlers

### Production Deployment
1. Build binary: `make build`
2. Copy `bin/app`, `Caddyfile`, and create `data/` directory to server
3. Set `APP_SECRET` environment variable to a random string
4. Run Caddy and the Go binary as separate processes or use Docker

### Backup Strategy
- Use the provided `backup.sh` script
- Schedule via cron: `15 2 * * * /path/to/backup.sh`
- Store backups in separate location from primary database
- SQLite supports online backups without stopping the application

## SaaS Features (Maintenance-Free)

### Authentication & Multi-tenancy
- **User Management**: Registration, login, session-based authentication
- **Tenant Isolation**: Complete data separation between organizations
- **Subscription Limits**: Automatic enforcement of item/user limits per plan
- **Role-based Access**: Owner/member roles within tenants

### Analytics & Usage Tracking
- **Event Tracking**: All user actions automatically tracked
- **Usage Reports**: Daily/monthly summaries, top users, event timelines
- **Real-time Dashboard**: Live stats without external services
- **Data Retention**: Automatic cleanup after 90 days

### Background Job System
- **SQLite-based Queue**: No Redis or external queue required
- **Automatic Retries**: Exponential backoff for failed jobs
- **Built-in Tasks**: Session cleanup, analytics rotation, email sending
- **Custom Jobs**: Easy to add new background processing

### Rate Limiting & Security
- **In-memory Limiting**: Token bucket algorithm with auto-cleanup
- **Flexible Keys**: IP, user, or tenant-based rate limiting
- **No External Store**: Uses application memory, scales with instances

### Email Notifications
- **Standard SMTP**: Uses Go's built-in email capabilities
- **Template System**: Welcome emails, password resets, limit warnings
- **Development Mode**: Logs instead of sending during development
- **Zero Dependencies**: No external email services required

### Key Benefits
- **Zero Maintenance**: All features self-manage and auto-cleanup
- **Single Database**: Everything stored in one SQLite file
- **No External Services**: Completely self-contained
- **Complete Data Ownership**: No vendor lock-in or data sharing
- **Predictable Costs**: No per-request or usage-based pricing

> 📖 **See [SAAS_FEATURES_GUIDE.md](SAAS_FEATURES_GUIDE.md) for detailed implementation guide and examples**
