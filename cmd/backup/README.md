# Trenova Backup CLI

A standalone command-line tool for managing Trenova database backups without starting the full application server.

## Features

- Create database backups
- List available backups
- Restore from backups
- Clean up old backups based on retention policy
- Version compatibility checks between pg_dump and PostgreSQL server

## Installation

```bash
# Clone the repository
git clone https://github.com/emoss08/trenova.git

# Build the backup CLI
cd trenova
go build -o trenova-backup ./cmd/backup
```

## Usage

### Basic Commands

```bash
# Create a backup
./trenova-backup create

# List available backups
./trenova-backup list

# Restore from a backup
./trenova-backup restore filename.sql.gz

# Clean up old backups
./trenova-backup cleanup
```

### Options

```bash
# Specify a config file
./trenova-backup --config=/path/to/config.yaml create

# Enable verbose output
./trenova-backup --verbose list

# Specify retention days for cleanup
./trenova-backup cleanup --days=60

# Restore with confirmation prompt
./trenova-backup restore mybackup.sql.gz
```

### Legacy Command Structure

For backward compatibility with existing scripts:

```bash
# Create a backup (default operation)
./trenova-backup backup

# List backups
./trenova-backup backup --list

# Restore from backup
./trenova-backup backup --restore=mybackup.sql.gz

# Clean up old backups
./trenova-backup backup --cleanup
```

## Configuration

The tool reads configuration from:

1. Command-line arguments
2. Environment variables with `TRENOVA_` prefix
3. Configuration file (default: `config/development/config.development.yaml`)

### Sample Configuration

```yaml
db:
  host: localhost
  port: 5432
  database: trenova
  username: postgres
  password: secretpassword
  sslMode: disable

backup:
  backupDir: ./backups
  retentionDays: 30
```

### Environment Variables

```bash
# Set database connection details
export TRENOVA_DB_HOST=localhost
export TRENOVA_DB_PORT=5432
export TRENOVA_DB_DATABASE=trenova
export TRENOVA_DB_USERNAME=postgres
export TRENOVA_DB_PASSWORD=secretpassword

# Set backup options
export TRENOVA_BACKUP_BACKUPDIR=./backups
export TRENOVA_BACKUP_RETENTIONDAYS=30
```

## PostgreSQL Version Compatibility

The tool automatically checks for version compatibility between your PostgreSQL server and the pg_dump client tool. If your pg_dump version is older than your PostgreSQL server version, you'll receive an error message.

For example:

```
Error creating backup: pg_dump version (16) is older than PostgreSQL server version (17). Please install pg_dump version 17 or newer
```

### Installing Compatible PostgreSQL Client Tools

#### Ubuntu/Debian

```bash
# Add PostgreSQL repository
sudo sh -c 'echo "deb https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
sudo apt-get update

# Install PostgreSQL 17 client tools
sudo apt-get install -y postgresql-client-17
```

#### RHEL/CentOS/Fedora

```bash
# Install PostgreSQL repository
sudo dnf install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-8-x86_64/pgdg-redhat-repo-latest.noarch.rpm

# Install PostgreSQL 17 client tools
sudo dnf install -y postgresql17
```

## Setting Up Cron Jobs

For automated backups, add a cron job:

```bash
# Edit crontab
crontab -e

# Add entry for daily backup at 2 AM
0 2 * * * /path/to/trenova-backup create > /var/log/trenova-backup.log 2>&1

# Add weekly cleanup (Sundays at 3 AM)
0 3 * * 0 /path/to/trenova-backup cleanup > /var/log/trenova-cleanup.log 2>&1
```
