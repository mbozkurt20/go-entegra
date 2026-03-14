#!/bin/bash
set -e

PROJECT_DIR="/home/mbozkurt/www/me/go-entegra"
BIN_DIR="$PROJECT_DIR/bin"
BINARY="$BIN_DIR/go-entegra"
SERVICE="go-entegra"
GO_BIN="/home/mbozkurt/go/bin/go"

echo "=== Go Entegra Deploy ==="
echo "Branch: $(git -C $PROJECT_DIR rev-parse --abbrev-ref HEAD)"
echo "Commit: $(git -C $PROJECT_DIR rev-parse --short HEAD)"

# Build
echo "[1/3] Building..."
mkdir -p $BIN_DIR
cd $PROJECT_DIR
$GO_BIN build -o $BINARY ./cmd/main.go
echo "Build OK: $BINARY"

# Restart service
echo "[2/3] Restarting service..."
sudo systemctl restart $SERVICE
sleep 2

# Health check
echo "[3/3] Health check..."
if systemctl is-active --quiet $SERVICE; then
    echo "Service running OK"
else
    echo "Service FAILED to start"
    sudo journalctl -u $SERVICE -n 30 --no-pager
    exit 1
fi

echo "=== Deploy complete ==="
