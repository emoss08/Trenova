# Trenova Database Backup System

This document provides details on how to configure, use, and maintain the database backup system for Trenova.

## Default State

By default, the backup service is **disabled** in Trenova. This design choice was made for several reasons:

1. **Database Variety**: Self-hosted deployments may use different database configurations
2. **Performance Considerations**: Automatic backups consume resources and should be explicitly enabled
3. **Flexibility**: Users can implement their own backup strategies that fit their infrastructure
4. **Security**: Users should consciously decide where backup files are stored and how they're managed

To enable the backup service, you need to explicitly set `backup.enabled: true` in your configuration file.

## Prerequisites

- PostgreSQL client tools (`pg_dump` and `pg_restore`) must be installed on the system running the backups
  - **IMPORTANT**: The version of these tools must match or be newer than your PostgreSQL server version
  - If you're using PostgreSQL 17, make sure you have PostgreSQL 17 client tools installed
- Sufficient disk space for storing backups
- Appropriate database permissions for the database user

## PostgreSQL Version Compatibility

Version compatibility is critical for successful backups. You cannot use an older version of `pg_dump` to back up a newer PostgreSQL server.

### Checking Your Versions

```bash
# Check PostgreSQL server version
psql -h hostname -U username -d dbname -c "SELECT version();"

# Check pg_dump version
pg_dump --version
```

### Installing Correct PostgreSQL Client Tools

If you encounter a version mismatch error (e.g., "server version mismatch"), install the matching client tools:

#### For Ubuntu/Debian

```bash
# Add PostgreSQL repository
sudo sh -c 'echo "deb https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
sudo apt-get update

# Install PostgreSQL client tools (replace 17 with your server version)
sudo apt-get install -y postgresql-client-17
```

#### For RHEL/CentOS/Fedora

```bash
# Install PostgreSQL repository
sudo dnf install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-8-x86_64/pgdg-redhat-repo-latest.noarch.rpm

# Install PostgreSQL client tools (replace 17 with your server version)
sudo dnf install -y postgresql17
```

#### For Docker Deployments

If you're running Trenova in Docker alongside a PostgreSQL container, ensure your backup scripts run inside the PostgreSQL container or use the same PostgreSQL version for client tools.

## Configuration

The backup system is configured in your application's config file. Here's a sample configuration:

```yaml
backup:
  # Enable or disable the backup service (default: false)
  enabled: true
  
  # Directory where backups will be stored
  backupDir: "./backups"
  
  # Number of days to keep backups before automatic deletion
  retentionDays: 30
  
  # Cron schedule for automated backups (daily at midnight)
  schedule: "0 0 * * *"
  
  # Compression level (1-9, higher = better compression but slower)
  compression: 6
  
  # Maximum number of concurrent backup operations
  maxConcurrentBackups: 1
  
  # Maximum time allowed for a backup operation (in seconds)
  backupTimeout: 1800  # 30 minutes
  
  # Whether to send notifications on backup failures
  notifyOnFailure: true
  
  # Whether to send notifications on backup success
  notifyOnSuccess: false
  
  # Email address for notifications (if enabled)
  notificationEmail: "admin@example.com"
```

## Backup Methods

The system provides several ways to manage backups:

### 1. Automated Backups

When the backup service is enabled, the system will automatically create backups according to the configured schedule. The schedule is specified using a cron expression (e.g., `0 0 * * *` for daily at midnight).

### 2. API Endpoints

The following API endpoints are available for managing backups:

- `GET /api/v1/backups` - List all available backups
- `POST /api/v1/backups` - Create a new backup
- `GET /api/v1/backups/{filename}` - Download a specific backup
- `DELETE /api/v1/backups/{filename}` - Delete a specific backup
- `POST /api/v1/backups/restore/{filename}` - Restore from a specific backup

These endpoints require admin authorization.

### 3. Command Line Interface

Trenova provides a dedicated CLI tool for managing backups. This tool can be used independently of the main application, making it ideal for scheduled jobs and automation.

**For detailed CLI documentation, see: [Trenova Backup CLI Documentation](../docs/CLI_BACKUP.md)**

The CLI tool supports creating, listing, restoring, and cleaning up backups with a simple command structure.

> **Note**: The CLI tool works regardless of whether the backup service is enabled in the configuration. This allows you to perform manual backups even if automatic backups are disabled.

### 4. Cron Jobs

For environments without the scheduler, you can use cron to automate backups:

```bash
# Edit crontab
crontab -e

# Add entry for daily backup at 2 AM
0 2 * * * /path/to/trenova-backup create >> /var/log/trenova-backup.log 2>&1

# Add weekly cleanup (Sundays at 3 AM)
0 3 * * 0 /path/to/trenova-backup cleanup >> /var/log/trenova-cleanup.log 2>&1
```

## Backup Format

Backups are created in PostgreSQL's custom format with compression. This format provides several advantages:

- Smaller file size through compression
- Ability to selectively restore specific parts of the database
- Platform-independent format for restore operations

Backup files are named using the pattern: `{database}-{timestamp}.sql.gz`

## Restoring Backups

**WARNING: Restoring a backup will overwrite your existing database. Make sure you understand the implications before proceeding.**

You can restore a backup using any of the methods described above:

- API: `POST /api/v1/backups/restore/{filename}`
- CLI: `trenova-backup restore filename.sql.gz`
- Manually: `pg_restore --clean --if-exists --no-owner --no-privileges -d your_database backup_file.sql.gz`

## Retention Policy

The system automatically manages backup retention based on the configured `retentionDays` value. Backups older than this number of days will be automatically deleted during:

1. Each new backup operation
2. When explicitly running cleanup operations

## Troubleshooting

If you encounter issues with the backup system, check the following:

### Version Mismatch Errors

If you see an error like `server version mismatch`, your PostgreSQL client tools are older than your server version:

```
Error creating backup: failed to execute pg_dump command: pg_dump: error: aborting because of server version mismatch
pg_dump: detail: server version: 17.4; pg_dump version: 16.8
```

Solution: Install PostgreSQL client tools that match your server version (see [Installing Correct PostgreSQL Client Tools](#installing-correct-postgresql-client-tools)).

### Other Common Issues

1. Ensure PostgreSQL client tools (`pg_dump` and `pg_restore`) are in your PATH
2. Verify the database user has sufficient permissions
3. Check that the backup directory exists and is writable
4. Review the application logs for detailed error messages
5. For restore operations, ensure the backup file exists and is not corrupted

## Best Practices

1. Store backups in a different location from your database server
2. Regularly test backup restoration to ensure backups are valid
3. Consider encrypting sensitive backups
4. Monitor disk space usage to prevent backups from filling up your disk
5. Set up external monitoring to ensure backups are being created successfully
