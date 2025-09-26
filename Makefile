.PHONY: help run build test lint fmt tidy clean docker docker-up docker-down install-tools dev test-coverage benchmark security-scan demo-data
.DEFAULT_GOAL := help

# Build variables
APP_NAME := 100y-saas
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

help: ## Show this help message
	@echo 'Usage: make <target>'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: ## Start development server with hot reload
	ENVIRONMENT=development DB_PATH=data/app.db APP_SECRET=local go run ./cmd/server

dev: ## Start development server with demo data
	@echo "Starting development server with demo data..."
	@mkdir -p data
	@ENVIRONMENT=development DB_PATH=data/app.db APP_SECRET=local go run ./cmd/server &
	@sleep 2
	@./examples/load_demo_data.sh || echo "Demo data loading failed (server may not be ready yet)"
	@echo "Development server running at http://localhost:8080"
	@echo "Demo accounts: demo@example.com/hello, admin@example.com/admin"

build: ## Build production binary
	@echo "Building $(APP_NAME) v$(VERSION)..."
	@mkdir -p bin
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/app ./cmd/server
	@echo "Binary built: bin/app"

build-all: ## Build binaries for all platforms
	@echo "Building for multiple platforms..."
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(APP_NAME)-linux-amd64 ./cmd/server
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(APP_NAME)-linux-arm64 ./cmd/server
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(APP_NAME)-darwin-amd64 ./cmd/server
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(APP_NAME)-darwin-arm64 ./cmd/server
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(APP_NAME)-windows-amd64.exe ./cmd/server
	@echo "All binaries built in dist/"

test: ## Run tests
	go test -v -race ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

benchmark: ## Run benchmarks
	go test -bench=. -benchmem ./...

lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run --timeout=5m

security-scan: ## Run security scan
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...

fmt: ## Format code
	gofmt -s -w .
	go mod tidy

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy dependencies
	go mod tidy
	go mod verify

clean: ## Clean build artifacts
	rm -rf bin/ dist/ coverage.out coverage.html data/ backups/

docker: ## Build Docker image
	docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

docker-up: ## Start with Docker Compose
	docker-compose up -d
	@echo "Services starting..."
	@sleep 5
	@echo "Application: http://localhost"
	@echo "Health check: http://localhost/healthz"

docker-down: ## Stop Docker Compose
	docker-compose down

demo-data: ## Load demo data (server must be running)
	./examples/load_demo_data.sh

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Tools installed successfully"

deploy-staging: build ## Deploy to staging (requires STAGING_SERVER env var)
	@if [ -z "$(STAGING_SERVER)" ]; then echo "STAGING_SERVER not set"; exit 1; fi
	@echo "Deploying to staging server: $(STAGING_SERVER)"
	scp bin/app $(STAGING_SERVER):/tmp/app-new
	ssh $(STAGING_SERVER) 'sudo systemctl stop 100y-saas || true && sudo cp /tmp/app-new /opt/100y-saas/bin/app && sudo systemctl start 100y-saas'
	@echo "Deployment completed"

backup: ## Create database backup
	@mkdir -p backups
	@cp data/app.db backups/app-$(shell date +%Y%m%d-%H%M%S).db
	@echo "Database backup created in backups/"

restore: ## Restore from backup (use BACKUP_FILE=path/to/backup.db)
	@if [ -z "$(BACKUP_FILE)" ]; then echo "Usage: make restore BACKUP_FILE=path/to/backup.db"; exit 1; fi
	@cp $(BACKUP_FILE) data/app.db
	@echo "Database restored from $(BACKUP_FILE)"

check-config: ## Validate configuration
	@echo "Checking configuration..."
	@ENVIRONMENT=development go run -ldflags "-X main.configCheck=true" ./cmd/server

monitor: ## Show application logs (requires systemd service)
	sudo journalctl -u 100y-saas -f

api-test: ## Test API endpoints (server must be running)
	@echo "Testing API endpoints..."
	curl -s http://localhost:8080/api/ping | jq .
	curl -s http://localhost:8080/healthz | jq .
	@echo "\nAPI tests completed"

# Database operations
db-shell: ## Open SQLite shell
	sqlite3 data/app.db

db-schema: ## Show database schema
	sqlite3 data/app.db '.schema'

db-tables: ## List database tables
	sqlite3 data/app.db '.tables'

db-migrate: ## Run database migrations (automatically done on startup)
	@echo "Migrations are automatically applied on server startup"
	@echo "Current schema version:"
	sqlite3 data/app.db "SELECT value FROM meta WHERE key='schema_version'" || echo "Database not initialized"
