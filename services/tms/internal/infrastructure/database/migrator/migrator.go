package migrator

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/migrations"
	"github.com/fatih/color"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type Migrator struct {
	db       *bun.DB
	migrator *migrate.Migrator
	config   *common.DatabaseConfig
	reporter common.ProgressReporter
}

func NewMigrator(config *common.DatabaseConfig) *Migrator {
	migrator := migrate.NewMigrator(config.DB, migrations.Migrations)

	return &Migrator{
		db:       config.DB,
		migrator: migrator,
		config:   config,
		reporter: common.NewConsoleProgressReporter(),
	}
}

func (m *Migrator) SetProgressReporter(reporter common.ProgressReporter) {
	m.reporter = reporter
}

func (m *Migrator) Initialize(ctx context.Context) error {
	if err := m.migrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize migration tables: %w", err)
	}

	color.Green("✓ Migration tables initialized")
	return nil
}

func (m *Migrator) Migrate(
	ctx context.Context,
	opts common.OperationOptions,
) (*common.OperationResult, error) {
	result := &common.OperationResult{
		Type:      common.OpMigrate,
		StartTime: time.Now(),
		Details:   make(map[string]any),
	}

	applied, unapplied, err := m.getMigrationLists(ctx)
	if err != nil {
		result.Error = err
		result.Success = false
		result.Message = fmt.Sprintf("Failed to get migration status: %v", err)
		return result, err
	}

	result.Details["applied_count"] = len(applied)
	result.Details["pending_count"] = len(unapplied)

	if len(unapplied) == 0 {
		result.Success = true
		result.Message = "No pending migrations"
		result.EndTime = time.Now()
		color.Yellow("→ Database is already up to date")
		return result, nil
	}

	color.Cyan("Pending migrations:")
	for _, mig := range unapplied {
		fmt.Printf("  - %s\n", mig.Name)
	}

	if opts.DryRun {
		result.Success = true
		result.Message = fmt.Sprintf("Dry run: Would apply %d migrations", len(unapplied))
		result.EndTime = time.Now()
		color.Blue("→ Dry run completed (no changes made)")
		return result, nil
	}

	if opts.Interactive && !opts.Force {
		fmt.Printf("\n%s Apply %d migration(s)? [y/N]: ",
			color.YellowString("?"),
			len(unapplied))

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			result.Success = false
			result.Message = "Migration cancelled by user"
			result.EndTime = time.Now()
			color.Red("✗ Migration cancelled")
			return result, nil
		}
	}

	if err := m.migrator.Lock(ctx); err != nil {
		result.Error = err
		result.Success = false
		result.Message = fmt.Sprintf("Failed to acquire migration lock: %v", err)
		return result, err
	}
	defer m.migrator.Unlock(ctx)

	m.reporter.Start(len(unapplied), "Running migrations...")

	group, err := m.migrator.Migrate(ctx)
	if err != nil {
		result.Error = err
		result.Success = false
		result.Message = fmt.Sprintf("Migration failed: %v", err)
		result.EndTime = time.Now()
		color.Red("✗ Migration failed: %v", err)
		return result, err
	}

	result.Success = true
	result.Details["migration_group"] = group.ID
	result.Details["migrations_applied"] = len(group.Migrations)
	result.Message = fmt.Sprintf("Successfully applied %d migrations", len(group.Migrations))
	result.EndTime = time.Now()

	m.reporter.Complete("Migrations completed")
	color.Green("✓ Migrated to version %d", group.ID)

	if opts.Verbose {
		for _, mig := range group.Migrations {
			color.Green("  ✓ %s", mig.Name)
		}
	}

	return result, nil
}

func (m *Migrator) Rollback(
	ctx context.Context,
	opts common.OperationOptions,
) (*common.OperationResult, error) {
	result := &common.OperationResult{
		Type:      common.OpRollback,
		StartTime: time.Now(),
		Details:   make(map[string]any),
	}

	if m.config.Environment == common.EnvProduction && !opts.Force {
		result.Error = fmt.Errorf("rollback disabled in production (use --force to override)")
		result.Success = false
		result.Message = "Rollback disabled in production environment"
		color.Red("✗ Rollback is disabled in production. Use --force to override.")
		return result, result.Error
	}

	applied, _, err := m.getMigrationLists(ctx)
	if err != nil {
		result.Error = err
		result.Success = false
		result.Message = fmt.Sprintf("Failed to get migration status: %v", err)
		return result, err
	}

	if len(applied) == 0 {
		result.Success = false
		result.Message = "No migrations to rollback"
		result.EndTime = time.Now()
		color.Yellow("→ No migrations to rollback")
		return result, nil
	}

	rollbackCount := 1
	if opts.Target != "" {
		fmt.Sscanf(opts.Target, "%d", &rollbackCount)
	}

	if rollbackCount > m.config.MaxRollback && m.config.MaxRollback > 0 {
		rollbackCount = m.config.MaxRollback
		color.Yellow("→ Limiting rollback to %d migrations (max configured)", rollbackCount)
	}

	color.Cyan("Migrations to rollback:")
	for i := len(applied) - rollbackCount; i < len(applied) && i >= 0; i++ {
		fmt.Printf("  - %s\n", applied[i].Name)
	}

	if opts.DryRun {
		result.Success = true
		result.Message = fmt.Sprintf("Dry run: Would rollback %d migrations", rollbackCount)
		result.EndTime = time.Now()
		color.Blue("→ Dry run completed (no changes made)")
		return result, nil
	}

	if opts.Interactive && !opts.Force {
		fmt.Printf("\n%s Rollback %d migration(s)? This action is destructive! [y/N]: ",
			color.RedString("⚠"),
			rollbackCount)

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			result.Success = false
			result.Message = "Rollback cancelled by user"
			result.EndTime = time.Now()
			color.Yellow("→ Rollback cancelled")
			return result, nil
		}
	}

	if err := m.migrator.Lock(ctx); err != nil {
		result.Error = err
		result.Success = false
		result.Message = fmt.Sprintf("Failed to acquire migration lock: %v", err)
		return result, err
	}
	defer m.migrator.Unlock(ctx)

	m.reporter.Start(rollbackCount, "Rolling back migrations...")

	group, err := m.migrator.Rollback(ctx)
	if err != nil {
		result.Error = err
		result.Success = false
		result.Message = fmt.Sprintf("Rollback failed: %v", err)
		result.EndTime = time.Now()
		color.Red("✗ Rollback failed: %v", err)
		return result, err
	}

	result.Success = true
	result.Details["rollback_group"] = group.ID
	result.Details["migrations_rolled_back"] = len(group.Migrations)
	result.Message = fmt.Sprintf("Successfully rolled back %d migrations", len(group.Migrations))
	result.EndTime = time.Now()

	m.reporter.Complete("Rollback completed")
	color.Green("✓ Rolled back to version %d", group.ID)

	if opts.Verbose {
		for _, mig := range group.Migrations {
			color.Yellow("  ↩ %s", mig.Name)
		}
	}

	return result, nil
}

func (m *Migrator) Status(ctx context.Context) ([]*common.MigrationStatus, error) {
	applied, unapplied, err := m.getMigrationLists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}

	var status []*common.MigrationStatus

	for _, mig := range applied {
		status = append(status, &common.MigrationStatus{
			ID:         mig.ID,
			Name:       mig.Name,
			Group:      mig.GroupID,
			MigratedAt: mig.MigratedAt,
			Applied:    true,
		})
	}

	for _, mig := range unapplied {
		status = append(status, &common.MigrationStatus{
			Name:    mig.Name,
			Applied: false,
		})
	}

	return status, nil
}

func (m *Migrator) CreateMigration(
	ctx context.Context,
	name string,
	transactional bool,
) ([]string, error) {
	var files []string

	if transactional {
		created, err := m.migrator.CreateTxSQLMigrations(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to create transactional migration: %w", err)
		}

		for _, f := range created {
			files = append(files, f.Path)
			color.Green("✓ Created %s", f.Path)
		}
	} else {
		created, err := m.migrator.CreateSQLMigrations(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to create migration: %w", err)
		}

		for _, f := range created {
			files = append(files, f.Path)
			color.Green("✓ Created %s", f.Path)
		}
	}

	return files, nil
}

func (m *Migrator) Reset(
	ctx context.Context,
	opts common.OperationOptions,
) (*common.OperationResult, error) {
	result := &common.OperationResult{
		Type:      common.OpReset,
		StartTime: time.Now(),
		Details:   make(map[string]any),
	}

	if m.config.Environment == common.EnvProduction {
		result.Error = fmt.Errorf("reset is not allowed in production environment")
		result.Success = false
		result.Message = "Reset operation is not allowed in production"
		color.Red("✗ Reset is not allowed in production environment")
		return result, result.Error
	}

	if opts.Interactive && !opts.Force {
		fmt.Printf("\n%s This will DROP ALL TABLES and recreate them. Are you sure? [y/N]: ",
			color.RedString("⚠ WARNING"))

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			result.Success = false
			result.Message = "Reset cancelled by user"
			result.EndTime = time.Now()
			color.Yellow("→ Reset cancelled")
			return result, nil
		}
	}

	color.Yellow("→ Dropping all tables...")

	if err := m.dropAllTables(ctx); err != nil {
		result.Error = err
		result.Success = false
		result.Message = fmt.Sprintf("Failed to drop tables: %v", err)
		result.EndTime = time.Now()
		color.Red("✗ Failed to drop tables: %v", err)
		return result, err
	}

	color.Green("✓ All tables dropped")

	if err := m.Initialize(ctx); err != nil {
		result.Error = err
		result.Success = false
		result.Message = fmt.Sprintf("Failed to reinitialize: %v", err)
		return result, err
	}

	color.Yellow("→ Running migrations...")
	migrateResult, err := m.Migrate(ctx, common.OperationOptions{Force: true})
	if err != nil {
		result.Error = err
		result.Success = false
		result.Message = fmt.Sprintf("Failed to run migrations after reset: %v", err)
		result.EndTime = time.Now()
		return result, err
	}

	result.Success = true
	result.Message = "Database reset completed successfully"
	result.Details["migrations_applied"] = migrateResult.Details["migrations_applied"]
	result.EndTime = time.Now()

	return result, nil
}

func (m *Migrator) getMigrationLists(
	ctx context.Context,
) (migrate.MigrationSlice, migrate.MigrationSlice, error) {
	applied, err := m.migrator.AppliedMigrations(ctx)
	if err != nil {
		return nil, nil, err
	}

	ms, err := m.migrator.MigrationsWithStatus(ctx)
	if err != nil {
		return nil, nil, err
	}

	appliedMap := make(map[string]bool)
	for _, mig := range applied {
		appliedMap[mig.Name] = true
	}

	var unapplied migrate.MigrationSlice
	for _, mig := range ms {
		if !appliedMap[mig.Name] {
			unapplied = append(unapplied, mig)
		}
	}

	return applied, unapplied, nil
}

func (m *Migrator) dropAllTables(ctx context.Context) error {
	query := `
		DROP SCHEMA public CASCADE;
		CREATE SCHEMA public;
		GRANT ALL ON SCHEMA public TO public;
	`

	_, err := m.db.ExecContext(ctx, query)
	return err
}
