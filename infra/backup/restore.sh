#!/usr/bin/env bash
#
# Database restore script for City Stories Guide
# Downloads a backup from S3 and restores it into PostgreSQL.
#
# Usage:
#   ./restore.sh                           # Restore latest daily backup
#   ./restore.sh --list                    # List available backups
#   ./restore.sh --file <s3_filename>      # Restore a specific backup
#   ./restore.sh --type weekly             # Restore latest weekly backup
#   ./restore.sh --local <path>            # Restore from local file
#
# Required environment variables:
#   POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD
#   S3_ENDPOINT, S3_BUCKET, S3_ACCESS_KEY, S3_SECRET_KEY

set -euo pipefail

# ---------- Configuration ----------

POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:?POSTGRES_DB is required}"
POSTGRES_USER="${POSTGRES_USER:?POSTGRES_USER is required}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"

S3_ENDPOINT="${S3_ENDPOINT:?S3_ENDPOINT is required}"
S3_BUCKET="${S3_BUCKET:?S3_BUCKET is required}"
S3_ACCESS_KEY="${S3_ACCESS_KEY:?S3_ACCESS_KEY is required}"
S3_SECRET_KEY="${S3_SECRET_KEY:?S3_SECRET_KEY is required}"

S3_PREFIX="${BACKUP_S3_PREFIX:-backups}"

export AWS_ACCESS_KEY_ID="$S3_ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="$S3_SECRET_KEY"
export PGPASSWORD="$POSTGRES_PASSWORD"

# ---------- Logging ----------

log() {
  echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"
}

# ---------- Parse arguments ----------

ACTION="restore_latest"
BACKUP_TYPE="daily"
SPECIFIC_FILE=""
LOCAL_FILE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --list)
      ACTION="list"
      shift
      ;;
    --file)
      ACTION="restore_specific"
      SPECIFIC_FILE="${2:?--file requires a filename}"
      shift 2
      ;;
    --type)
      BACKUP_TYPE="${2:?--type requires daily or weekly}"
      shift 2
      ;;
    --local)
      ACTION="restore_local"
      LOCAL_FILE="${2:?--local requires a file path}"
      shift 2
      ;;
    *)
      echo "Unknown option: $1" >&2
      echo "Usage: $0 [--list] [--file <name>] [--type daily|weekly] [--local <path>]" >&2
      exit 1
      ;;
  esac
done

# ---------- List backups ----------

list_backups() {
  log "Available backups in s3://${S3_BUCKET}/${S3_PREFIX}/:"
  echo ""
  echo "=== Daily backups ==="
  aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/daily/" \
    --endpoint-url "$S3_ENDPOINT" 2>/dev/null \
    | sort -r \
    || echo "  (none)"
  echo ""
  echo "=== Weekly backups ==="
  aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/weekly/" \
    --endpoint-url "$S3_ENDPOINT" 2>/dev/null \
    | sort -r \
    || echo "  (none)"
}

# ---------- Download backup ----------

download_backup() {
  local s3_key="$1"
  local dest="$2"

  log "Downloading s3://${S3_BUCKET}/${s3_key}..."
  aws s3 cp "s3://${S3_BUCKET}/${s3_key}" "$dest" \
    --endpoint-url "$S3_ENDPOINT" \
    --no-progress \
    || { log "ERROR: Failed to download backup"; exit 1; }

  log "Downloaded: $(du -h "$dest" | cut -f1)"
}

get_latest_file() {
  local type="$1"
  aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/${type}/" \
    --endpoint-url "$S3_ENDPOINT" 2>/dev/null \
    | sort -r \
    | head -1 \
    | awk '{print $NF}'
}

# ---------- Restore database ----------

do_restore() {
  local dump_file="$1"

  log "WARNING: This will overwrite database '${POSTGRES_DB}' on ${POSTGRES_HOST}:${POSTGRES_PORT}"
  echo ""

  # Skip prompt if running non-interactively (e.g., in CI/cron)
  if [[ -t 0 ]]; then
    read -r -p "Are you sure you want to continue? (yes/no): " confirm
    if [[ "$confirm" != "yes" ]]; then
      log "Restore cancelled."
      exit 0
    fi
  fi

  log "Restoring database '${POSTGRES_DB}'..."

  pg_restore \
    -h "$POSTGRES_HOST" \
    -p "$POSTGRES_PORT" \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" \
    --clean \
    --if-exists \
    --no-owner \
    --no-privileges \
    --single-transaction \
    "$dump_file" \
    2>/dev/null \
    || { log "ERROR: pg_restore failed"; exit 1; }

  log "Restore completed successfully."
}

# ---------- Main ----------

TEMP_FILE="/tmp/csg_restore_$$.dump"
trap 'rm -f "$TEMP_FILE"' EXIT

case "$ACTION" in
  list)
    list_backups
    ;;

  restore_latest)
    LATEST="$(get_latest_file "$BACKUP_TYPE")"
    if [[ -z "$LATEST" ]]; then
      log "ERROR: No ${BACKUP_TYPE} backups found"
      exit 1
    fi
    log "Latest ${BACKUP_TYPE} backup: ${LATEST}"
    download_backup "${S3_PREFIX}/${BACKUP_TYPE}/${LATEST}" "$TEMP_FILE"
    do_restore "$TEMP_FILE"
    ;;

  restore_specific)
    # Try to find the file in daily or weekly
    if aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/daily/${SPECIFIC_FILE}" --endpoint-url "$S3_ENDPOINT" &>/dev/null; then
      download_backup "${S3_PREFIX}/daily/${SPECIFIC_FILE}" "$TEMP_FILE"
    elif aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/weekly/${SPECIFIC_FILE}" --endpoint-url "$S3_ENDPOINT" &>/dev/null; then
      download_backup "${S3_PREFIX}/weekly/${SPECIFIC_FILE}" "$TEMP_FILE"
    else
      log "ERROR: Backup file '${SPECIFIC_FILE}' not found in daily or weekly"
      exit 1
    fi
    do_restore "$TEMP_FILE"
    ;;

  restore_local)
    if [[ ! -f "$LOCAL_FILE" ]]; then
      log "ERROR: Local file '${LOCAL_FILE}' not found"
      exit 1
    fi
    log "Restoring from local file: ${LOCAL_FILE}"
    do_restore "$LOCAL_FILE"
    ;;
esac
