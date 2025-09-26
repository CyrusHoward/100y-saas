# PowerShell script to run the 100y-saas development server
# Equivalent to `make run`

$env:DB_PATH = "data/app.db"
$env:APP_SECRET = "local"

Write-Host "Starting 100y-saas development server..."
Write-Host "Database: $env:DB_PATH"
Write-Host "Open http://localhost:8080 in your browser"
Write-Host ""

go run ./cmd/server
