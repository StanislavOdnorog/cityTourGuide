#!/usr/bin/env bash
#
# Database backup script for City Stories Guide
# Creates a gzip-compressed PostgreSQL dump and uploads to S3-compatible storage.
# Manages retention: 7 daily backups + 4 weekly backups.
#
# Usage:
#   ./backup.sh                  # Run backup with defaults
#   ./backup.sh --weekly         # Tag as weekly backup
#   BACKUP_ALERT_WEBHOOK=https://... ./backup.sh  # Enable failure alerts
#
# Required environment variables:
#   POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD
#   S3_ENDPOINT, S3_BUCKET, S3_ACCESS_KEY, S3_SECRET_KEY
#
# Optional:
#   BACKUP_S3_PREFIX       - S3 key prefix (default: backups/)
#   BACKUP_DAILY_RETAIN    - Number of daily backups to keep (default: 7)
#   BACKUP_WEEKLY_RETAIN   - Number of weekly backups to keep (default: 4)
#   BACKUP_ALERT_WEBHOOK   - Webhook URL for failure alerts (POST JSON)
#   BACKUP_ALERT_EMAIL     - Email for failure alerts (requires mail/sendmail)

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
DAILY_RETAIN="${BACKUP_DAILY_RETAIN:-7}"
WEEKLY_RETAIN="${BACKUP_WEEKLY_RETAIN:-4}"
ALERT_WEBHOOK="${BACKUP_ALERT_WEBHOOK:-}"
ALERT_EMAIL="${BACKUP_ALERT_EMAIL:-}"

TIMESTAMP="$(date -u +%Y%m%d_%H%M%S)"
DAY_OF_WEEK="$(date -u +%u)"  # 1=Monday, 7=Sunday

# Determine backup type
if [[ "${1:-}" == "--weekly" ]] || [[ "$DAY_OF_WEEK" == "7" ]]; then
  BACKUP_TYPE="weekly"
else
  BACKUP_TYPE="daily"
fi

DUMP_FILE="/tmp/csg_${POSTGRES_DB}_${TIMESTAMP}.sql.gz"
S3_KEY="${S3_PREFIX}/${BACKUP_TYPE}/csg_${POSTGRES_DB}_${TIMESTAMP}.sql.gz"

# ---------- Logging ----------

log() {
  echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"
}

# ---------- Alert on failure ----------

send_alert() {
  local message="$1"

  if [[ -n "$ALERT_WEBHOOK" ]]; then
    curl -sf -X POST "$ALERT_WEBHOOK" \
      -H "Content-Type: application/json" \
      -d "{\"text\":\"[CSG Backup FAILED] ${message}\",\"username\":\"backup-bot\"}" \
      2>/dev/null || log "WARNING: Failed to send webhook alert"
  fi

  if [[ -n "$ALERT_EMAIL" ]]; then
    echo "[CSG Backup FAILED] ${message}" | mail -s "Backup failure: ${POSTGRES_DB}" "$ALERT_EMAIL" \
      2>/dev/null || log "WARNING: Failed to send email alert"
  fi
}

cleanup() {
  rm -f "$DUMP_FILE"
}
trap cleanup EXIT

handle_error() {
  local msg="Backup failed at $(date -u +%Y-%m-%dT%H:%M:%SZ) — ${1:-unknown error}"
  log "ERROR: $msg"
  send_alert "$msg"
  exit 1
}

# ---------- Step 1: Create database dump ----------

log "Starting ${BACKUP_TYPE} backup of database '${POSTGRES_DB}'..."

export PGPASSWORD="$POSTGRES_PASSWORD"

pg_dump \
  -h "$POSTGRES_HOST" \
  -p "$POSTGRES_PORT" \
  -U "$POSTGRES_USER" \
  -d "$POSTGRES_DB" \
  --format=custom \
  --compress=6 \
  --no-owner \
  --no-privileges \
  -f "$DUMP_FILE" \
  2>/dev/null \
  || handle_error "pg_dump failed"

DUMP_SIZE="$(du -h "$DUMP_FILE" | cut -f1)"
log "Dump created: ${DUMP_FILE} (${DUMP_SIZE})"

# ---------- Step 2: Upload to S3 ----------

log "Uploading to s3://${S3_BUCKET}/${S3_KEY}..."

# Use AWS CLI with custom endpoint for S3-compatible storage
export AWS_ACCESS_KEY_ID="$S3_ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="$S3_SECRET_KEY"

aws s3 cp "$DUMP_FILE" "s3://${S3_BUCKET}/${S3_KEY}" \
  --endpoint-url "$S3_ENDPOINT" \
  --no-progress \
  || handle_error "S3 upload failed"

log "Upload complete: s3://${S3_BUCKET}/${S3_KEY}"

# ---------- Step 3: Enforce retention policy ----------

log "Enforcing retention policy: ${DAILY_RETAIN} daily, ${WEEKLY_RETAIN} weekly..."

enforce_retention() {
  local type="$1"
  local keep="$2"

  # List objects, sort by key (contains timestamp), keep only the ones to delete
  local objects
  objects="$(aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/${type}/" \
    --endpoint-url "$S3_ENDPOINT" 2>/dev/null \
    | awk '{print $NF}' \
    | sort -r || true)"

  local count=0
  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    count=$((count + 1))
    if [[ $count -gt $keep ]]; then
      log "  Deleting old ${type} backup: ${file}"
      aws s3 rm "s3://${S3_BUCKET}/${S3_PREFIX}/${type}/${file}" \
        --endpoint-url "$S3_ENDPOINT" \
        2>/dev/null || log "  WARNING: Failed to delete ${file}"
    fi
  done <<< "$objects"

  local deleted=$((count > keep ? count - keep : 0))
  log "  ${type}: ${count} total, kept ${keep}, deleted ${deleted}"
}

enforce_retention "daily" "$DAILY_RETAIN"
enforce_retention "weekly" "$WEEKLY_RETAIN"

# ---------- Done ----------

log "Backup completed successfully: ${BACKUP_TYPE} (${DUMP_SIZE})"
