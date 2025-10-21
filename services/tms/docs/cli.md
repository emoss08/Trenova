# Trenova CLI Documentation

## Overview

The Trenova CLI is a unified command-line tool for managing and administering the Trenova Transportation Management System. It combines all system components (API server, worker processes, database management) into a single binary, providing a comprehensive interface for system administration, development, and operations.

## Installation

### Building from Source

```bash
# Build the unified CLI binary
make build-cli

# Or build directly with Go
go build -o trenova ./cmd/cli

# The binary will be available at ./build/trenova-cli
```

### Running the CLI

```bash
# Run directly from source
go run ./cmd/cli [command]

# Or use the built binary
./build/trenova-cli [command]

# Or use make commands (which auto-build if needed)
make run               # Start API server
make run-worker        # Start worker process
make db-status         # Check database status
```

## Global Flags

These flags are available for all commands:

- `--config string` - Specify config file location (default: `config/config.yaml`)
- `--help, -h` - Show help for any command

## Available Commands

### 1. Version Command

Display version and environment information.

```bash
trenova version
```

**Output:**

- Trenova version
- Current environment (development/staging/production)

### 2. Configuration Commands

Manage and validate system configuration.

#### `config validate`

Validate the configuration file for errors.

```bash
trenova config validate
```

**Purpose:** Ensures your configuration file is properly formatted and contains all required fields.

#### `config show`

Display the current configuration file contents.

```bash
trenova config show
```

**Purpose:** Shows the raw configuration file contents, useful for debugging configuration issues.

**Features:**

- Uses `bat` if available for syntax-highlighted YAML output
- Falls back to `cat` for plain text display
- Reads file directly if neither tool is available
- Respects `--config` flag for custom config file paths

### 3. API Commands

Manage and run the API server.

#### `api run`

Start the API server.

```bash
trenova api run
```

**Purpose:** Starts the Trenova API server with all configured services.

**Features:**

- Loads configuration from config file
- Initializes all dependencies via dependency injection
- Starts HTTP server with configured middleware
- Connects to database, cache, and other services
- Handles graceful shutdown

**Environment Variables:**

- Uses configuration from `config/config.yaml` by default
- Respects `--config` flag for custom configuration

### 4. Database Commands

Comprehensive database management utilities located under the `db` command group.

#### `db migrate`

Run pending database migrations.

```bash
trenova db migrate
trenova db migrate --dry-run    # Preview without applying
trenova db migrate --verbose    # Show detailed output
```

**Purpose:** Applies any unapplied database migrations to bring your schema up to date.

**Flags:**

- `--dry-run` - Preview migrations without applying them
- `--verbose` - Show detailed migration output

#### `db rollback`

Rollback database migrations.

```bash
trenova db rollback              # Rollback last migration
trenova db rollback --target 3   # Rollback 3 migrations
trenova db rollback --dry-run    # Preview rollback
```

**Purpose:** Reverses previously applied migrations, useful for testing or recovering from bad migrations.

**Flags:**

- `--target int` - Number of migrations to rollback
- `--dry-run` - Preview rollback without applying
- `--force` - Skip confirmation prompts

#### `db seed`

Apply database seeds for initial/test data.

```bash
trenova db seed
trenova db seed --force      # Re-apply already applied seeds
trenova db seed --verbose    # Show detailed output
trenova db seed --dry-run    # Preview what would be seeded
```

**Purpose:** Populates the database with initial data (admin accounts, permissions, test data based on environment).

**Flags:**

- `--force` - Force re-application of already applied seeds
- `--dry-run` - Preview seeds without applying
- `--verbose` - Show detailed seeding output
- `--interactive, -i` - Interactive mode with confirmations

**Environment-specific behavior:**

- **Production/Staging:** Only base seeds (states, admin, permissions)
- **Development:** Base + development seeds (test organizations, users)
- **Test:** Base + development + test seeds

#### `db reset`

Reset the database (DESTRUCTIVE - removes all data).

```bash
trenova db reset              # Reset with confirmation
trenova db reset --force      # Skip confirmation
trenova db reset --seed       # Also apply seeds after reset
```

**Purpose:** Completely resets the database by dropping all tables, recreating schema, and optionally reseeding.

**Flags:**

- `--force` - Skip confirmation prompt
- `--seed` - Apply seeds after reset

**Safety:** Not available in production environment

#### `db setup`

Complete database setup (migrate + seed).

```bash
trenova db setup
trenova db setup --verbose    # Show detailed output
```

**Purpose:** Runs migrations and seeds in one command, ideal for initial setup.

#### `db status`

Show current migration and seed status.

```bash
trenova db status
```

**Purpose:** Displays:

- Applied migrations with timestamps
- Pending migrations
- Applied seeds with versions and environments
- Orphaned seeds (marked with ✗)

**Output includes:**

- Migration history
- Seed application history
- Summary counts

#### `db create`

Create a new database migration.

```bash
trenova db create add_user_table     # Create SQL migration
trenova db create add_index --tx     # Create transactional migration
```

**Purpose:** Generates a new timestamped migration file in the migrations directory.

**Flags:**

- `--tx` - Create migration with transaction wrapper

#### `db create-seed`

Create a new database seed file.

```bash
trenova db create-seed my_feature         # Base seed (all environments)
trenova db create-seed test_data --dev    # Development seed
trenova db create-seed e2e_data --test    # Test-only seed
```

**Purpose:** Generates a new seed file with boilerplate code in the appropriate directory.

**Flags:**

- `--dev` - Create seed in development directory
- `--test` - Create seed in test directory

**Features:**

- Automatically numbers seeds for execution order
- Generates proper boilerplate with helpers
- Auto-updates seed registry
- Provides environment-specific templates

#### `db seed-sync`

Synchronize the seed registry with filesystem.

```bash
trenova db seed-sync
```

**Purpose:** Regenerates the seed registry to match current seed files. Use when:

- You've manually created seed files
- You've deleted seed files
- Registry is out of sync

**What it does:**

- Scans all seed directories
- Regenerates `seed_registry.go`
- Updates available seeds

#### `db seed-check`

Check for orphaned seeds in the database.

```bash
trenova db seed-check
```

**Purpose:** Identifies seeds that were applied but their files have been deleted from the filesystem.

**Output:**

- Lists orphaned seed entries
- Shows when they were applied
- Suggests running `seed-clean` to remove them

#### `db seed-clean`

Clean up orphaned seed history entries.

```bash
trenova db seed-clean          # Clean with confirmation
trenova db seed-clean --force  # Skip confirmation
```

**Purpose:** Marks orphaned seed entries as "Orphaned" in the database and regenerates the registry.

**Flags:**

- `--force` - Skip confirmation prompt

#### `db seed-watch`

Watch seed directories for changes (development tool).

```bash
trenova db seed-watch
```

**Purpose:** Monitors seed directories and automatically regenerates the registry when files are:

- Created
- Modified
- Deleted
- Renamed

**Features:**

- Real-time monitoring
- Automatic registry updates
- Color-coded change notifications
- Debounced regeneration (2-second delay)

**Use case:** Keep running during seed development for automatic updates

### 5. Search Vector Command

Generate SQL for search vector configurations.

```bash
trenova search-vector             # Interactive mode
trenova search-vector -l          # List all searchable domains
trenova search-vector -a          # Generate SQL for all domains
trenova search-vector -d          # Include optimization hints
```

**Purpose:** Generates PostgreSQL full-text search configurations for domain entities.

**Flags:**

- `--list, -l` - List all searchable domains
- `--all, -a` - Generate SQL for all searchable domains
- `--deep, -d` - Include nested relationships and optimization hints

**Output:**

- SQL ALTER TABLE statements
- Trigger definitions
- Index creation statements
- Performance optimization comments

### 6. Redis Commands

Manage Redis cache operations.

#### `redis flush`

Flush all data from the Redis cache.

```bash
trenova redis flush
```

**Purpose:** Clears all cached data from Redis. Useful for:

- Clearing stale cache after configuration changes
- Resetting cache during debugging
- Forcing fresh data loads

**Warning:** This removes ALL data from Redis, including session data, cached queries, and temporary storage.

### 7. Worker Command

Manage background worker processes.

```bash
trenova worker run               # Start worker process
trenova worker run --verbose     # Run with detailed logging
```

**Purpose:** Starts the background worker that processes:

- Async tasks
- Scheduled jobs
- Queue processing
- Workflow orchestration

**Note:** Worker configuration is read from the main config file.

## Configuration

The CLI reads configuration from `config/config.yaml` by default. You can override this with the `--config` flag.

### Configuration Structure

```yaml
app:
  name: Trenova
  version: 1.0.0
  env: development  # production, staging, development, test

db:
  host: localhost
  port: 5432
  name: trenova_go_db
  username: postgres
  password: postgres
  max_open: 25
  max_idle: 5
  log_queries: true  # Enable query logging in development

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

# Additional service configurations...
```

## Environment Detection

The CLI automatically detects the environment from the configuration and adjusts behavior accordingly:

- **Production:** Restrictive mode with safety checks, no destructive operations
- **Staging:** Similar to production but allows some testing operations
- **Development:** Full features, verbose output, development seeds
- **Test:** Optimized for testing, includes test-specific seeds

## Using with Make

The Makefile provides convenient shortcuts that automatically build the CLI if needed:

```bash
# Application operations
make run                # Start API server
make run-worker         # Start worker process
make dev                # Start development environment with hot reload

# Database operations
make db-migrate          # Run migrations
make db-seed            # Apply seeds
make db-reset           # Reset database
make db-setup           # Setup (migrate + seed)
make db-status          # Show status

# Seed management
make db-create-seed name=my_seed [env=dev|test]
make db-seed-sync       # Sync registry
make db-seed-check      # Check orphaned seeds
make db-seed-clean      # Clean orphaned seeds
make generate-seeds     # Regenerate seed registry

# Redis operations
make redis-flush        # Flush Redis cache

# Build operations
make build              # Build the CLI binary
make build-cli          # Build the unified CLI
make clean              # Clean build artifacts
```

## Architecture

### Directory Structure

```
cmd/cli/
├── main.go              # Entry point and root command
├── api/                 # API server command
│   └── run_api.go       # API server runner
├── db/                  # Database command group
│   ├── database.go      # Main db command and common functions
│   ├── db_create_seed.go
│   ├── db_seed_check.go
│   ├── db_seed_sync.go
│   └── db_seed_watch.go
├── worker/              # Worker process command
│   └── worker.go        # Worker runner
└── [other commands]     # Additional command files
```

### Package Organization

- **Main Package:** Contains root command, version, config, search vector commands
- **API Package:** API server startup and management
- **DB Package:** All database-related commands and utilities
- **Worker Package:** Background worker process management
- **Shared State:** Configuration passed to packages via `SetConfig()`

## Error Handling

The CLI provides detailed error messages with context:

```bash
# Example error output
Error: failed to connect to database: pq: password authentication failed
  Hint: Check your database credentials in config/config.yaml
```

### Common Error Solutions

1. **"Config file not found"**
   - Ensure `config/config.yaml` exists
   - Use `--config` flag to specify location

2. **"Database connection failed"**
   - Check database is running
   - Verify credentials in config
   - Ensure network connectivity

3. **"Migration locked"**
   - Another migration is running
   - Check for stuck migration locks
   - Clear lock table if needed

4. **"Seed already applied"**
   - Seed was previously run
   - Use `--force` to re-apply
   - Check seed history with `db status`

## Best Practices

### Development Workflow

1. **Initial Setup:**

   ```bash
   make quick-start     # Sets up everything
   ```

2. **Daily Development:**

   ```bash
   make db-status       # Check current state
   make db-migrate      # Apply new migrations
   make db-seed         # Apply new seeds
   ```

3. **Creating Seeds:**

   ```bash
   make db-create-seed name=feature_x env=dev
   # Edit the generated file
   make db-seed
   ```

4. **Testing Migrations:**

   ```bash
   make db-migrate --dry-run    # Preview
   make db-migrate               # Apply
   make db-rollback              # If issues
   ```

### Production Deployment

1. **Pre-deployment:**

   ```bash
   trenova config validate       # Validate config
   trenova db migrate --dry-run  # Preview migrations
   ```

2. **Deployment:**

   ```bash
   trenova db migrate            # Apply migrations
   trenova db status             # Verify state
   ```

### Seed Management

1. **Keep seeds idempotent** - Should be safe to run multiple times
2. **Use numeric prefixes** - Controls execution order (00_, 01_, 02_)
3. **Separate by environment** - Use appropriate directories
4. **Document dependencies** - Note what each seed requires

## Extending the CLI

### Adding New Commands

1. Create new file in `cmd/cli/` or `cmd/cli/db/`
2. Define command with `cobra.Command`
3. Add to parent command in init()
4. Follow existing patterns for consistency

### Command Template

```go
var myCmd = &cobra.Command{
    Use:   "my-command",
    Short: "Brief description",
    Long:  `Detailed description...`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Command logic
        return nil
    },
}

func init() {
    parentCmd.AddCommand(myCmd)
    myCmd.Flags().BoolP("flag", "f", false, "Flag description")
}
```

## Troubleshooting

### Debug Mode

Enable debug output:

```bash
export BUNDEBUG=1              # Enable Bun SQL logging
trenova db migrate --verbose   # Verbose output
```

### Checking Logs

The CLI outputs to stdout/stderr. Redirect for analysis:

```bash
trenova db seed 2>&1 | tee seed.log
```

### Performance

For large operations:

- Use `--verbose` to monitor progress
- Run migrations in batches if many pending
- Consider connection pool settings in config

## Security Considerations

- **Credentials:** Never commit config files with production credentials
- **Destructive Operations:** Protected by environment checks
- **Audit Trail:** Seed history tracks who applied what and when
- **Confirmations:** Interactive mode for dangerous operations

## Support

For issues or questions:

- Check this documentation first
- Review error messages and hints
- Enable verbose/debug mode for details
- Report issues with full command output

## Version History

The CLI version matches the main application version. Check with:

```bash
trenova version
```

Updates to the CLI are backward compatible within major versions.
