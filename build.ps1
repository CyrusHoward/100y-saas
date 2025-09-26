# PowerShell script to build the 100y-saas binary
# Equivalent to `make build`

Write-Host "Building 100y-saas binary..."

# Create bin directory if it doesn't exist
New-Item -ItemType Directory -Path "bin" -Force | Out-Null

# Build the application
$env:CGO_ENABLED = "0"
go build -o bin/app.exe ./cmd/server

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful! Binary created at bin/app.exe"
    Write-Host ""
    Write-Host "To run in production:"
    Write-Host "  Set-Item env:DB_PATH 'data/app.db'"
    Write-Host "  Set-Item env:APP_SECRET 'your-secret-key'"
    Write-Host "  ./bin/app.exe"
} else {
    Write-Host "Build failed!"
    exit 1
}
