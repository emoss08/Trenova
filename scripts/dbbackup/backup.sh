#!/bin/bash
# # Copyright 2023-2025 Eric Moss
# # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

#
# Database Backup Script for Trenova
# Usage: backup.sh [--retention-days=DAYS]
#
# This script is designed to be used with cron to create regular PostgreSQL backups.
# It properly handles errors and sends notifications if configured.
#
# Example crontab entry (daily at 2 AM):
# 0 2 * * * /path/to/backup.sh >> /var/log/trenova-backup.log 2>&1
#

# Set up error handling
set -e
trap 'echo "Error occurred at line $LINENO. Command: $BASH_COMMAND"' ERR

# Configuration
APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_BIN="${APP_DIR}/trenova"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
LOG_FILE="${APP_DIR}/logs/backup_${TIMESTAMP}.log"

# Parse command line arguments
RETENTION_DAYS=""
for arg in "$@"; do
	case $arg in
	--retention-days=*)
		RETENTION_DAYS="${arg#*=}"
		shift
		;;
	*)
		# Unknown option
		;;
	esac
done

# Ensure log directory exists
mkdir -p "${APP_DIR}/logs"

# Run the backup
echo "Starting database backup at $(date)" | tee -a "$LOG_FILE"

# Build retention days argument if specified
RETENTION_ARG=""
if [ -n "$RETENTION_DAYS" ]; then
	RETENTION_ARG="--retention-days=$RETENTION_DAYS"
fi

# Execute the backup
if $APP_BIN backup $RETENTION_ARG >>"$LOG_FILE" 2>&1; then
	echo "Backup completed successfully at $(date)" | tee -a "$LOG_FILE"
	exit 0
else
	echo "Backup failed at $(date)" | tee -a "$LOG_FILE"

	# Send notification email if configured
	if command -v mail &>/dev/null && [ -n "$NOTIFICATION_EMAIL" ]; then
		echo "Database backup failed. See attached log for details." | mail -s "Database Backup Failed - $(hostname)" -a "$LOG_FILE" "$NOTIFICATION_EMAIL"
	fi

	exit 1
fi
