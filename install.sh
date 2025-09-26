#!/bin/bash
set -euo pipefail

# 100y-saas One-Click Install Script
# Installs Go, builds the application, and sets up systemd service

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="100y-saas"
SERVICE_USER="100y-saas"
INSTALL_DIR="/opt/100y-saas"
GO_VERSION="1.22.5"
DOMAIN="${DOMAIN:-}"
APP_SECRET="${APP_SECRET:-$(openssl rand -hex 32)}"

echo -e "${BLUE}🚀 100y-saas One-Click Install Script${NC}"
echo "======================================"

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}❌ This script must be run as root (use sudo)${NC}"
   exit 1
fi

# Detect OS
if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    OS=$ID
    OS_VERSION=$VERSION_ID
else
    echo -e "${RED}❌ Cannot detect OS version${NC}"
    exit 1
fi

echo -e "${BLUE}📋 Detected OS: $OS $OS_VERSION${NC}"

# Function to install Go
install_go() {
    echo -e "${YELLOW}📦 Installing Go $GO_VERSION...${NC}"
    
    # Check if Go is already installed with correct version
    if command -v go &> /dev/null; then
        CURRENT_GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        if [[ "$CURRENT_GO_VERSION" == "$GO_VERSION" ]]; then
            echo -e "${GREEN}✅ Go $GO_VERSION is already installed${NC}"
            return
        fi
    fi
    
    # Determine architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        armv6l) ARCH="armv6l" ;;
        armv8) ARCH="arm64" ;;
        aarch64) ARCH="arm64" ;;
        *) echo -e "${RED}❌ Unsupported architecture: $ARCH${NC}"; exit 1 ;;
    esac
    
    # Download and install Go
    GO_TAR="go${GO_VERSION}.linux-${ARCH}.tar.gz"
    cd /tmp
    wget -q "https://golang.org/dl/${GO_TAR}"
    
    # Remove existing Go installation
    rm -rf /usr/local/go
    
    # Extract new version
    tar -C /usr/local -xzf "$GO_TAR"
    
    # Add to PATH
    echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
    chmod +x /etc/profile.d/go.sh
    
    # Make available immediately
    export PATH=$PATH:/usr/local/go/bin
    
    echo -e "${GREEN}✅ Go $GO_VERSION installed successfully${NC}"
}

# Function to install system dependencies
install_dependencies() {
    echo -e "${YELLOW}📦 Installing system dependencies...${NC}"
    
    case $OS in
        ubuntu|debian)
            apt-get update -qq
            apt-get install -y wget curl git build-essential sqlite3
            ;;
        centos|rhel|fedora)
            if command -v dnf &> /dev/null; then
                dnf install -y wget curl git gcc sqlite
            else
                yum install -y wget curl git gcc sqlite
            fi
            ;;
        *)
            echo -e "${YELLOW}⚠️  Unknown OS, please install: wget, curl, git, build-essential, sqlite3${NC}"
            ;;
    esac
    
    echo -e "${GREEN}✅ System dependencies installed${NC}"
}

# Function to create service user
create_user() {
    echo -e "${YELLOW}👤 Creating service user...${NC}"
    
    if id "$SERVICE_USER" &>/dev/null; then
        echo -e "${GREEN}✅ User $SERVICE_USER already exists${NC}"
    else
        useradd --system --no-create-home --shell /bin/false "$SERVICE_USER"
        echo -e "${GREEN}✅ Created user $SERVICE_USER${NC}"
    fi
}

# Function to build application
build_app() {
    echo -e "${YELLOW}🔨 Building application...${NC}"
    
    # Create install directory
    mkdir -p "$INSTALL_DIR"
    cp -r . "$INSTALL_DIR/"
    cd "$INSTALL_DIR"
    
    # Build the application
    export PATH=$PATH:/usr/local/go/bin
    go mod download
    CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o bin/app ./cmd/server
    
    # Make executable
    chmod +x bin/app
    
    # Create data directories
    mkdir -p data backups logs
    chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"
    
    echo -e "${GREEN}✅ Application built successfully${NC}"
}

# Function to create systemd service
create_systemd_service() {
    echo -e "${YELLOW}🔧 Creating systemd service...${NC}"
    
    cat > /etc/systemd/system/${APP_NAME}.service << EOF
[Unit]
Description=100y-saas - Maintenance-free SaaS Application
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$INSTALL_DIR
Environment=DB_PATH=$INSTALL_DIR/data/app.db
Environment=APP_SECRET=$APP_SECRET
ExecStart=$INSTALL_DIR/bin/app
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$APP_NAME

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$INSTALL_DIR/data $INSTALL_DIR/backups $INSTALL_DIR/logs
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "$APP_NAME"
    
    echo -e "${GREEN}✅ Systemd service created${NC}"
}

# Function to setup Caddy (optional)
setup_caddy() {
    if [[ -z "$DOMAIN" ]]; then
        echo -e "${YELLOW}⚠️  No DOMAIN specified, skipping Caddy setup${NC}"
        return
    fi
    
    echo -e "${YELLOW}🌐 Setting up Caddy reverse proxy...${NC}"
    
    # Install Caddy
    case $OS in
        ubuntu|debian)
            apt install -y debian-keyring debian-archive-keyring apt-transport-https
            curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
            echo "deb [signed-by=/usr/share/keyrings/caddy-stable-archive-keyring.gpg] https://dl.cloudsmith.io/public/caddy/stable/deb/debian any-version main" | tee /etc/apt/sources.list.d/caddy-stable.list
            apt update
            apt install -y caddy
            ;;
        centos|rhel|fedora)
            dnf copr enable @caddy/caddy
            dnf install -y caddy
            ;;
    esac
    
    # Create Caddyfile
    cat > /etc/caddy/Caddyfile << EOF
$DOMAIN {
    reverse_proxy localhost:8080
    encode gzip
    
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        X-Content-Type-Options nosniff
        X-Frame-Options DENY
        X-XSS-Protection "1; mode=block"
    }
}
EOF
    
    systemctl enable caddy
    systemctl restart caddy
    
    echo -e "${GREEN}✅ Caddy configured for $DOMAIN${NC}"
}

# Function to create backup script
setup_backups() {
    echo -e "${YELLOW}💾 Setting up automated backups...${NC}"
    
    cat > "$INSTALL_DIR/backup.sh" << 'EOF'
#!/bin/bash
set -euo pipefail

SRC=${DB_PATH:-/opt/100y-saas/data/app.db}
DST_DIR=${BACKUP_DIR:-/opt/100y-saas/backups}
RETENTION_DAYS=${RETENTION_DAYS:-30}

mkdir -p "$DST_DIR"
ts=$(date -u +%Y%m%d-%H%M%S)

# Create backup
cp "$SRC" "$DST_DIR/app-$ts.db"
gzip "$DST_DIR/app-$ts.db"

# Cleanup old backups
find "$DST_DIR" -name "app-*.db.gz" -type f -mtime +$RETENTION_DAYS -delete

echo "Backup completed: app-$ts.db.gz"
EOF
    
    chmod +x "$INSTALL_DIR/backup.sh"
    chown "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR/backup.sh"
    
    # Add to crontab for daily backups at 2:15 AM
    (crontab -u $SERVICE_USER -l 2>/dev/null; echo "15 2 * * * $INSTALL_DIR/backup.sh") | crontab -u $SERVICE_USER -
    
    echo -e "${GREEN}✅ Daily backup scheduled at 2:15 AM${NC}"
}

# Function to start services
start_services() {
    echo -e "${YELLOW}🚀 Starting services...${NC}"
    
    systemctl start "$APP_NAME"
    
    # Wait a moment for startup
    sleep 3
    
    # Check status
    if systemctl is-active --quiet "$APP_NAME"; then
        echo -e "${GREEN}✅ $APP_NAME service is running${NC}"
    else
        echo -e "${RED}❌ $APP_NAME service failed to start${NC}"
        systemctl status "$APP_NAME" --no-pager
        exit 1
    fi
}

# Function to display completion info
show_completion_info() {
    echo
    echo -e "${GREEN}🎉 Installation completed successfully!${NC}"
    echo "=================================="
    echo -e "📍 Application installed at: ${BLUE}$INSTALL_DIR${NC}"
    echo -e "👤 Service user: ${BLUE}$SERVICE_USER${NC}"
    echo -e "🔑 App secret: ${YELLOW}$APP_SECRET${NC} (save this!)"
    echo
    echo -e "${YELLOW}📋 Management Commands:${NC}"
    echo "  Start:   sudo systemctl start $APP_NAME"
    echo "  Stop:    sudo systemctl stop $APP_NAME" 
    echo "  Status:  sudo systemctl status $APP_NAME"
    echo "  Logs:    sudo journalctl -u $APP_NAME -f"
    echo
    echo -e "${YELLOW}🌐 Access Information:${NC}"
    if [[ -n "$DOMAIN" ]]; then
        echo "  URL: https://$DOMAIN"
    else
        echo "  URL: http://localhost:8080"
        echo "  Health: http://localhost:8080/healthz"
    fi
    echo
    echo -e "${YELLOW}💾 Backups:${NC}"
    echo "  Location: $INSTALL_DIR/backups/"
    echo "  Schedule: Daily at 2:15 AM"
    echo "  Manual:   sudo -u $SERVICE_USER $INSTALL_DIR/backup.sh"
    echo
    echo -e "${YELLOW}⚙️  Configuration:${NC}"
    echo "  Edit: $INSTALL_DIR/.env (create if needed)"
    echo "  Restart after config changes: sudo systemctl restart $APP_NAME"
    echo
    echo -e "${GREEN}🚀 Your 100y-saas application is ready!${NC}"
}

# Main installation flow
main() {
    echo -e "${BLUE}Starting installation...${NC}"
    
    install_dependencies
    install_go
    create_user
    build_app
    create_systemd_service
    setup_caddy
    setup_backups
    start_services
    show_completion_info
}

# Handle interruption
trap 'echo -e "\n${RED}Installation interrupted!${NC}"; exit 1' INT TERM

# Run main installation
main "$@"
