#!/bin/bash
# Quick demo data loader for 100y-saas

set -euo pipefail

DB_PATH="${DB_PATH:-data/app.db}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🎯 Loading demo data into 100y-saas..."
echo "Database: $DB_PATH"

# Check if database exists
if [[ ! -f "$DB_PATH" ]]; then
    echo "❌ Database not found at $DB_PATH"
    echo "   Run the application first to create the database, then try again."
    exit 1
fi

# Load sample data
sqlite3 "$DB_PATH" < "$SCRIPT_DIR/sample_data.sql"

echo "✅ Demo data loaded successfully!"
echo ""
echo "📋 Demo accounts you can use:"
echo "  demo@example.com / hello       (Acme Corporation owner)"
echo "  admin@example.com / admin      (Tech Startup owner)"  
echo "  user@example.com / secret      (Freelancer)"
echo "  test@company.com / (empty)     (Demo Company)"
echo ""
echo "🏢 Demo workspaces:"
echo "  - Acme Corporation (Pro plan, 6 items)"
echo "  - Tech Startup Inc (Starter plan, 4 items)" 
echo "  - Freelancer Workspace (Free plan, 3 items)"
echo "  - Demo Company (Free plan, 2 items)"
echo ""
echo "🚀 Start the application and log in with any demo account to explore!"
