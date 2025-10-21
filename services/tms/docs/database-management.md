# Database Management System

## Overview

Trenova's database management system provides a comprehensive, enterprise-grade solution for managing database migrations, seeding, and data initialization. The system is designed to work seamlessly in both development and production environments with appropriate safety measures.

## Architecture

```plaintext
internal/infrastructure/database/
├── common/                 # Shared types and interfaces
│   └── types.go           # Common types (Environment, OperationOptions, etc.)
├── migrator/              # Migration engine
│   └── migrator.go        # Handles database migrations using Bun
├── seeder/                # All database management and seeding infrastructure
│   ├── manager.go         # Main database manager that coordinates everything
│   ├── seeder.go          # Core seeding engine with history tracking
│   ├── registry.go        # Registry generator code
│   ├── seed_registry.go   # Generated registry (DO NOT EDIT)
│   └── generate.go        # go:generate directive
└── seeds/                 # ONLY actual seed data files
    ├── base/              # Seeds for all environments
    │   ├── 00_us_states.go
    │   ├── 01_admin_account.go
    │   └── 02_permissions.go
    ├── development/       # Seeds for dev/test only
    │   ├── 00_test_organizations.go
    │   └── 01_test_users.go
    └── test/              # Seeds for test only

pkg/seedhelpers/          # Helper utilities for writing seeds
├── base.go               # BaseSeed struct that all seeds embed
├── context.go            # SeedContext with cached common entities
└── helpers.go            # Helper functions (CreateUser, CreateOrganization, etc.)
```

## How It Works

### 1. Seeder Package (`seeder/`)

Contains all database management infrastructure:

- **manager.go** - Central coordination point, initializes connections, provides unified API
- **seeder.go** - Core seeding engine with history tracking
- **registry.go** - Code generator that discovers and registers seeds
- **seed_registry.go** - Generated file with all seed registrations
- **generate.go** - Contains go:generate directive

Key features:

- Tracks seed history in `seed_history` table
- Ensures idempotent execution
- Handles transactions and rollbacks
- Reports progress
- Uses custom `seed_status_enum` (Active, Inactive, Orphaned)

### 2. Seeds (`seeds/`)

- Actual data initialization code
- Organized by environment (base, development, test)
- Numbered for execution order (00, 01, 02...)
- Auto-registered via code generation

### 3. Code Generation

- `seeder/registry.go` scans the `seeds/` directories
- Generates `seeder/seed_registry.go` with all discovered seeds
- Run with: `go generate ./internal/infrastructure/database/seeder/...`
- Automatically runs when creating new seeds via CLI

### 4. Seed Helpers (`pkg/seedhelpers/`)

- `BaseSeed` - Standard structure all seeds embed
- `SeedContext` - Caches common entities (default org, states, roles)
- Helper functions reduce boilerplate

## CLI Commands

The database management system is integrated into the main Trenova CLI under the `db` command:

### Migration Commands

```bash
# Run pending migrations
trenova db migrate
trenova db migrate --dry-run      # Preview without applying
trenova db migrate --verbose      # Show detailed output

# Rollback migrations
trenova db rollback               # Rollback last migration
trenova db rollback --target 3    # Rollback 3 migrations
trenova db rollback --dry-run     # Preview rollback

# Check migration status
trenova db status                 # Show migration and seed status

# Create new migration
trenova db create add_user_table  # Create SQL migration
trenova db create add_index --tx  # Create transactional migration
```

### Seeding Commands

```bash
# Apply seeds for current environment
trenova db seed
trenova db seed --force           # Force re-apply seeds
trenova db seed --verbose         # Show detailed output

# Create a new seed
trenova db create-seed my_feature          # Base seed (all environments)
trenova db create-seed test_data --dev     # Development seed
trenova db create-seed e2e_data --test      # Test-only seed

# Synchronize seed registry
trenova db seed-sync              # Regenerate registry to match filesystem

# Check for orphaned seeds (applied but deleted)
trenova db seed-check             # Check for deleted seeds
trenova db seed-clean             # Clean orphaned seed history
trenova db seed-clean --force     # Skip confirmation

# Watch for seed changes (development)
trenova db seed-watch             # Auto-update registry on file changes
```

### Maintenance Commands

```bash
# Setup database (migrate + seed)
trenova db setup
trenova db setup --verbose        # Show detailed output

# Reset database (non-production only)
trenova db reset                  # Reset with confirmation
trenova db reset --force          # Skip confirmation
trenova db reset --seed           # Apply seeds after reset
```

## Environment Detection

The system automatically detects the environment and applies appropriate seeds:

- `production/prod` → Only base seeds
- `staging/stage` → Only base seeds
- `development/dev` → Base + development seeds
- `test/testing` → Base + development + test seeds

## Creating New Seeds

### Using the CLI (Recommended)

```bash
# Create a base seed (runs in all environments)
trenova db create-seed my_seed_name

# Create a development seed
trenova db create-seed my_seed_name --dev

# Create a test-only seed
trenova db create-seed my_seed_name --test
```

The CLI will:

1. Create the seed file with proper boilerplate
2. Automatically regenerate the seed registry
3. The seed will be available immediately

### File Naming Convention

Seeds are named with a numeric prefix to control execution order:

- `00_first_seed.go` - Runs first
- `01_second_seed.go` - Runs second
- etc.

### Seed Structure

All seeds should:

1. Embed `seedhelpers.BaseSeed`
2. Implement the `SeedRunner` interface
3. Use the helper functions from `pkg/seedhelpers`

Example:

```go
type MyNewSeed struct {
    seedhelpers.BaseSeed
}

func NewMyNewSeed() *MyNewSeed {
    seed := &MyNewSeed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed(
        "MyNewSeed",
        "1.0.0",
        "Description of what this seed does",
        []common.Environment{common.EnvDevelopment, common.EnvTest},
    )
    return seed
}

func (s *MyNewSeed) Run(ctx context.Context, db *bun.DB) error {
    return seedhelpers.RunInTransaction(ctx, db, s.Name(), func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
        // Your seed logic here
        return nil
    })
}
```

## Automatic Features

### Registry Auto-Update

- When you create a seed using `trenova db create-seed`, the registry is automatically updated
- No need to manually run `go generate`

### Orphan Detection

- `trenova db seed-check` detects seeds that were deleted from filesystem but still in database
- `trenova db seed-clean` marks them as orphaned and updates registry
- The seed_history table tracks status with custom enum (Active, Inactive, Orphaned)

### File Watching (Development)

- `trenova db seed-watch` monitors seed directories
- Automatically regenerates registry when seeds are added, modified, or deleted
- Useful during active development

## Helper Functions

The `pkg/seedhelpers` package provides utilities:

- `SeedContext` - Cached access to common entities
- `CreateUser()` - Create users with defaults
- `CreateOrganization()` - Create orgs with required settings
- `GetDefaultOrganization()` - Get the default org
- `GetState()` - Get US states by abbreviation
- `AssignRoleToUser()` - Assign roles to users
- `RunInTransaction()` - Run seed in a transaction with automatic rollback on error

## Execution Order

1. Base seeds (in numeric order)
2. Development seeds (in numeric order, if applicable)
3. Test seeds (in numeric order, if applicable)

## Best Practices

### Seed Ordering

- Use numeric prefixes (00_, 01_, etc.) to control execution order
- Lower numbers run first
- Leave gaps (00, 05, 10) for future seeds

### Idempotency

- Seeds should be safe to run multiple times
- Check if data exists before creating
- Use upsert operations where appropriate

### Transactions

- Always use `RunInTransaction` helper
- This ensures automatic rollback on errors
- Keeps database consistent

### Caching

- Use `SeedContext` to avoid repeated queries
- Context caches common entities like default org, states, roles
- Improves seed performance

### Dependencies

- Base seeds should not depend on environment-specific seeds
- Use `SetDependencies()` to declare seed dependencies
- Dependencies are informational (not enforced yet)

## Running Seeds

```bash
# Apply all seeds for current environment
trenova db seed

# Force re-run seeds
trenova db seed --force

# See what would be applied
trenova db seed --dry-run

# Verbose output
trenova db seed --verbose
```

## Safety Features

### Migration Safety

- **Lock mechanism** prevents concurrent migrations
- **Dry-run mode** for previewing changes
- **Rollback limits** to prevent excessive rollbacks
- **Transaction support** for atomic migrations

### Seed Safety

- **Idempotent operations** - seeds can be run multiple times safely
- **Version tracking** prevents duplicate application
- **Checksum validation** ensures seed integrity
- **Environment restrictions** prevent production data in dev
- **Orphan detection** identifies deleted seeds
- **Automatic registry updates** keep seeds in sync

### Interactive Mode

- Confirmation prompts for destructive operations
- Clear warnings for production environments
- Detailed progress reporting
- Comprehensive error messages

## Database Tables

### seed_history

Tracks all applied seeds with:

- `id` - Unique identifier
- `name` - Seed name
- `version` - Seed version
- `environment` - Environment where applied
- `checksum` - Integrity check
- `applied_at` - Timestamp
- `applied_by` - User/system
- `status` - seed_status_enum (Active, Inactive, Orphaned)
- `notes` - Additional information
- `error` - Error message if failed
- `details` - JSONB metadata

## Troubleshooting

### Common Issues

1. **"Migration locked" error**
   - Another migration is running or was interrupted
   - Solution: Wait or manually unlock in the database

2. **"Seed already applied" message**
   - The seed has been successfully applied
   - Use `--force` to re-apply if needed

3. **"Not allowed in production" error**
   - Certain operations are restricted in production
   - Verify you're in the correct environment

4. **"Orphaned seed" warnings**
   - Seeds were deleted from filesystem but still in database
   - Run `trenova db seed-clean` to clean up

### Debug Mode

Enable debug output with environment variables:

```bash
export BUNDEBUG=1  # Enable Bun query debugging
trenova db migrate --verbose
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
- name: Run Migrations
  run: |
    trenova db migrate --force

- name: Apply Seeds
  run: |
    trenova db seed

- name: Verify Database
  run: |
    trenova db status
```

### Docker Integration

```dockerfile
# Run migrations on container start
CMD ["trenova", "db", "setup"]
```

## Security Considerations

- Default admin password should be changed immediately
- Database credentials are never logged
- Sensitive operations require explicit confirmation
- Production environment has strict safeguards
- All operations are logged for audit purposes
- Seed history tracked with full audit trail

## Development Workflow

1. **Create a new seed**:

   ```bash
   trenova db create-seed feature_x --dev
   ```

2. **Edit the generated file** in `internal/infrastructure/database/seeds/development/`

3. **Apply the seed**:

   ```bash
   trenova db seed
   ```

4. **If you delete a seed**, clean up:

   ```bash
   trenova db seed-check    # See orphaned seeds
   trenova db seed-clean     # Clean them up
   ```

5. **During active development**, use watch mode:

   ```bash
   trenova db seed-watch    # Auto-updates registry
   ```

## Future Enhancements

- [ ] Seed dependency enforcement (currently informational)
- [ ] Seed rollback functionality
- [ ] Seed version migrations
- [ ] Parallel seed execution for non-dependent seeds
- [ ] Seed performance metrics
- [ ] Automated seed testing framework
