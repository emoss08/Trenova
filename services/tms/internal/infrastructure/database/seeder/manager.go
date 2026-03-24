package seeder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/migrator"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/fatih/color"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"go.uber.org/fx"
)

type Manager struct {
	db       *bun.DB
	migrator DatabaseMigrator
	seeder   Seeder
}

type ManagerParams struct {
	fx.In

	DB       *bun.DB
	Migrator DatabaseMigrator
	Seeder   Seeder
}

func NewManager(p ManagerParams) *Manager {
	return &Manager{
		db:       p.DB,
		migrator: p.Migrator,
		seeder:   p.Seeder,
	}
}

type ManagerDeps struct {
	DB       *bun.DB
	Migrator DatabaseMigrator
	Seeder   Seeder
}

func NewManagerWithDeps(deps ManagerDeps) *Manager {
	return &Manager{
		db:       deps.DB,
		migrator: deps.Migrator,
		seeder:   deps.Seeder,
	}
}

type ManagerConfig struct {
	Config   *config.Config
	Registry *Registry
}

func NewManagerFromConfig(cfg ManagerConfig) (*Manager, error) {
	db, err := createDB(cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	dbConfig := createDBConfig(db, cfg.Config)
	mig := migrator.NewMigrator(dbConfig)
	engine := NewEngine(db, cfg.Registry, cfg.Config)

	return &Manager{
		db:       db,
		migrator: mig,
		seeder:   engine,
	}, nil
}

func createDB(cfg *config.Config) (*bun.DB, error) {
	sqldb := sql.OpenDB(
		pgdriver.NewConnector(pgdriver.WithDSN(cfg.GetDSN(cfg.Database.Password))),
	)
	db := bun.NewDB(sqldb, pgdialect.New())

	if cfg.App.Debug {
		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.FromEnv("BUNDEBUG"),
		))
	}

	db.RegisterModel(domainregistry.RegisterEntities()...)

	return db, nil
}

func createDBConfig(db *bun.DB, cfg *config.Config) *common.DatabaseConfig {
	env := common.EnvDevelopment
	switch cfg.App.Env {
	case "production", "prod":
		env = common.EnvProduction
	case "staging", "stage":
		env = common.EnvStaging
	case "test", "testing":
		env = common.EnvTest
	}

	return &common.DatabaseConfig{
		DB:               db,
		Environment:      env,
		MigrationsPath:   "./internal/infrastructure/postgres/migrations",
		MigrationsTable:  "bun_migrations",
		SeedsPath:        "./internal/infrastructure/database/seeds",
		SeedsTable:       "seed_history",
		FixturesPath:     "./test/fixtures",
		BackupPath:       "./backups",
		RequireBackup:    env == common.EnvProduction,
		AllowDestructive: env != common.EnvProduction,
		MaxRollback:      5,
	}
}

func (m *Manager) GetDB() *bun.DB {
	return m.db
}

func (m *Manager) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *Manager) Migrate(ctx context.Context, opts common.OperationOptions) error {
	color.Cyan("🔄 Running database migrations...")

	if err := m.migrator.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}

	result, err := m.migrator.Migrate(ctx, opts)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("migration failed: %s", result.Message)
	}

	return nil
}

func (m *Manager) Rollback(ctx context.Context, opts common.OperationOptions) error {
	color.Cyan("↩ Rolling back database migrations...")

	result, err := m.migrator.Rollback(ctx, opts)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("rollback failed: %s", result.Message)
	}

	return nil
}

func (m *Manager) Seed(ctx context.Context, opts common.OperationOptions) error {
	color.Cyan("🌱 Seeding database...")

	engineOpts := ExecuteOptions{
		Environment:  opts.Environment,
		Target:       opts.Target,
		Force:        opts.Force,
		IgnoreErrors: false,
		DryRun:       opts.DryRun,
		Verbose:      opts.Verbose,
		Interactive:  opts.Interactive,
	}

	report, err := m.seeder.Execute(ctx, engineOpts)
	if err != nil {
		if errors.Is(err, ErrUserCancelled) {
			color.Yellow("→ Seeding cancelled")
			return nil
		}
		return err
	}

	if !report.Success() {
		return fmt.Errorf("seeding completed with %d failures", report.Failed)
	}

	return nil
}

func (m *Manager) RollbackSeed(
	ctx context.Context,
	seedName string,
	env common.Environment,
	dryRun bool,
) error {
	engine, ok := m.seeder.(*Engine)
	if !ok {
		return fmt.Errorf("seeder does not support rollback")
	}

	if dryRun {
		seed, exists := engine.Registry().Get(seedName)
		if !exists {
			return fmt.Errorf("seed %s not found", seedName)
		}

		if !seed.CanRollback() {
			return fmt.Errorf("seed %s does not support rollback", seedName)
		}

		dependents, err := engine.findDependents(seedName)
		if err != nil {
			return err
		}

		if len(dependents) > 0 {
			color.Yellow(
				"→ Seed %s cannot be rolled back (dependent seeds: %v)",
				seedName,
				dependents,
			)
			return nil
		}

		color.Green("→ Seed %s can be rolled back", seedName)
		return nil
	}

	if err := engine.Rollback(ctx, seedName, env); err != nil {
		return err
	}

	return nil
}

func (m *Manager) RollbackAllSeeds(ctx context.Context, env common.Environment, dryRun bool) error {
	engine, ok := m.seeder.(*Engine)
	if !ok {
		return fmt.Errorf("seeder does not support rollback")
	}

	seeds := engine.Registry().All()

	rollbackOrder := make([]Seed, 0, len(seeds))
	for i := len(seeds) - 1; i >= 0; i-- {
		if seeds[i].CanRollback() {
			rollbackOrder = append(rollbackOrder, seeds[i])
		}
	}

	if len(rollbackOrder) == 0 {
		color.Yellow("→ No seeds support rollback")
		return nil
	}

	color.Cyan("→ Rolling back %d seed(s) in reverse order:", len(rollbackOrder))
	for _, seed := range rollbackOrder {
		fmt.Printf("  - %s\n", seed.Name())
	}

	if dryRun {
		return nil
	}

	for _, seed := range rollbackOrder {
		color.Yellow("→ Rolling back %s...", seed.Name())
		if err := engine.Rollback(ctx, seed.Name(), env); err != nil {
			color.Red("✗ Failed to rollback %s: %v", seed.Name(), err)
			return fmt.Errorf("rollback %s: %w", seed.Name(), err)
		}
	}

	return nil
}

func (m *Manager) Reset(ctx context.Context, opts common.OperationOptions) error {
	color.Cyan("🔄 Resetting database...")

	result, err := m.migrator.Reset(ctx, opts)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("reset failed: %s", result.Message)
	}

	return nil
}

func (m *Manager) Setup(ctx context.Context, opts common.OperationOptions) error {
	color.Cyan("🚀 Setting up database...")

	if err := m.Migrate(ctx, opts); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	if err := m.Seed(ctx, opts); err != nil {
		return fmt.Errorf("seeding failed: %w", err)
	}

	color.Green("✅ Database setup completed successfully!")
	return nil
}

func (m *Manager) MigrationStatus(ctx context.Context) error {
	status, err := m.migrator.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	color.Cyan("📊 Migration Status:")
	fmt.Println()

	appliedCount := 0
	pendingCount := 0

	for _, mig := range status {
		if mig.Applied {
			appliedCount++
			color.Green(
				"  ✓ %s (applied at %s)",
				mig.Name,
				mig.MigratedAt.Format("2006-01-02 15:04:05"),
			)
		} else {
			pendingCount++
			color.Yellow("  ○ %s (pending)", mig.Name)
		}
	}

	fmt.Println()
	color.Cyan("Summary: %d applied, %d pending", appliedCount, pendingCount)

	return nil
}

func (m *Manager) SeedStatus(ctx context.Context) error {
	status, err := m.seeder.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to get seed status: %w", err)
	}

	if len(status) == 0 {
		color.Yellow("→ No seeds have been applied")
		return nil
	}

	color.Cyan("📊 Seed Status:")
	fmt.Println()

	for _, seed := range status {
		statusIcon := "✓"
		statusColor := color.GreenString

		if seed.Status != "Active" {
			statusIcon = "✗"
			statusColor = color.RedString
		}

		fmt.Printf("  %s %s (v%s) - %s - Applied at %s\n",
			statusColor(statusIcon),
			seed.Name,
			seed.Version,
			seed.Environment,
			seed.AppliedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func (m *Manager) CreateMigration(ctx context.Context, name string, transactional bool) error {
	files, err := m.migrator.CreateMigration(ctx, name, transactional)
	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	color.Green("✅ Created migration files:")
	for _, file := range files {
		fmt.Printf("  - %s\n", file)
	}

	return nil
}
