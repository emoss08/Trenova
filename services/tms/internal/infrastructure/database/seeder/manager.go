package seeder

import (
	"context"
	"database/sql"
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
)

type Manager struct {
	config   *config.Config
	db       *bun.DB
	migrator *migrator.Migrator
	seeder   *Seeder
}

func NewManager(cfg *config.Config) (*Manager, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.GetDSN(cfg.Database.Password))))
	db := bun.NewDB(sqldb, pgdialect.New())

	if cfg.App.Debug {
		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
			bundebug.FromEnv("BUNDEBUG"),
		))
	}

	db.RegisterModel(domainregistry.RegisterEntities()...)

	env := common.EnvDevelopment
	switch cfg.App.Env {
	case "production", "prod":
		env = common.EnvProduction
	case "staging", "stage":
		env = common.EnvStaging
	case "test", "testing":
		env = common.EnvTest
	}

	dbConfig := &common.DatabaseConfig{
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

	m := &Manager{
		config:   cfg,
		db:       db,
		migrator: migrator.NewMigrator(dbConfig),
		seeder:   NewSeeder(dbConfig),
	}

	m.registerSeedRunners()

	return m, nil
}

func (m *Manager) GetDB() *bun.DB {
	return m.db
}

func (m *Manager) Close() error {
	return m.db.Close()
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

	result, err := m.seeder.Seed(ctx, opts)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("seeding failed: %s", result.Message)
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

func (m *Manager) registerSeedRunners() {
	env := common.EnvDevelopment // Default
	switch m.config.App.Env {
	case "production", "prod":
		env = common.EnvProduction
	case "staging", "stage":
		env = common.EnvStaging
	case "test", "testing":
		env = common.EnvTest
	}

	RegisterAll(m.seeder, env)
}
