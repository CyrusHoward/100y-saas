#!/usr/bin/env bash
set -euo pipefail
SRC=${DB_PATH:-data/app.db}
DST_DIR=${BACKUP_DIR:-backups}
mkdir -p "$DST_DIR"
ts=$(date -u +%Y%m%d-%H%M%S)
cp "$SRC" "$DST_DIR/app-$ts.db"
