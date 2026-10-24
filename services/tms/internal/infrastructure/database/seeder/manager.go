package seeder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/fatih/color"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"go.uber.org/fx"
)

type Manager struct {
	db          *bun.DB
	seeder      Seeder
	cfg         *config.Config
	permissions PermissionCacheInvalidator
	redisClient *redis.Client
}

type ManagerParams struct {
	fx.In

	DB     *bun.DB
	Seeder Seeder
}

func NewManager(p ManagerParams) *Manager {
	return &Manager{
		db:     p.DB,
		seeder: p.Seeder,
	}
}

type ManagerDeps struct {
	DB          *bun.DB
	Seeder      Seeder
	Permissions PermissionCacheInvalidator
}

func NewManagerWithDeps(deps ManagerDeps) *Manager {
	return &Manager{
		db:          deps.DB,
		seeder:      deps.Seeder,
		permissions: deps.Permissions,
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

	engine := NewEngine(db, cfg.Registry, cfg.Config)
	redisClient, permissions := newPermissionCache(cfg.Config)

	return &Manager{
		db:          db,
		seeder:      engine,
		cfg:         cfg.Config,
		permissions: permissions,
		redisClient: redisClient,
	}, nil
}

// cliStatementBackstop bounds a single statement from seed and reconcile only
// against a connection that has gone silent. Bulk seed batches legitimately run
// for minutes; a statement blocked on a lock fails fast on lock_timeout
// instead, as on the API's connection.
const cliStatementBackstop = 30 * time.Minute

func createDB(cfg *config.Config) (*bun.DB, error) {
	db := bun.NewDB(
		sql.OpenDB(pgdriver.NewConnector(
			pgdriver.WithDSN(cfg.GetMigrationDSN()),
			pgdriver.WithReadTimeout(cliStatementBackstop),
			pgdriver.WithWriteTimeout(cliStatementBackstop),
			pgdriver.WithConnParams(map[string]any{
				"lock_timeout": fmt.Sprintf(
					"%dms",
					max(cfg.Database.GetLockTimeout().Milliseconds(), 1),
				),
			}),
		)),
		pgdialect.New(),
	)

	if cfg.App.Debug {
		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.FromEnv("BUNDEBUG"),
		))
	}

	db.RegisterModel(domainregistry.RegisterManyToManyEntities()...)
	db.RegisterModel(domainregistry.RegisterEntities()...)

	return db, nil
}

func (m *Manager) GetDB() *bun.DB {
	return m.db
}

func (m *Manager) Close() error {
	var errs []error
	if m.redisClient != nil {
		errs = append(errs, m.redisClient.Close())
	}
	if m.db != nil {
		errs = append(errs, m.db.Close())
	}
	return errors.Join(errs...)
}

func (m *Manager) Reconcile(ctx context.Context, opts common.OperationOptions) error {
	color.Cyan("🔄 Reconciling the database with this build...")

	if err := m.provisionRLS(ctx, opts); err != nil {
		return err
	}

	if err := m.reconcileAdminRoles(ctx, opts); err != nil {
		return err
	}

	return m.reconcileSeeds(ctx, opts)
}

func (m *Manager) reconcileAdminRoles(ctx context.Context, opts common.OperationOptions) error {
	if opts.DryRun || m.db == nil {
		return nil
	}

	result, err := tenantbootstrap.ReconcileAdminRoles(ctx, m.db, permission.NewRegistry())
	if err != nil {
		return fmt.Errorf("grant new permissions to administrator roles: %w", err)
	}

	if result.GrantsAdded > 0 || result.GrantsExtended > 0 {
		color.Yellow(
			"🔑 Granted %d new resource(s) and %d new operation set(s) to %d administrator role(s)",
			result.GrantsAdded,
			result.GrantsExtended,
			result.RolesChecked,
		)
	}

	if err = invalidatePermissionCaches(ctx, m.permissions, result.ChangedOrganizations); err != nil {
		color.Yellow(
			"⚠️  %v; administrators see the new permissions once their cached permissions expire",
			err,
		)
	}

	return nil
}

func (m *Manager) provisionRLS(ctx context.Context, opts common.OperationOptions) error {
	if opts.DryRun || m.cfg == nil {
		return nil
	}

	result, err := postgres.ProvisionRLS(ctx, m.db, m.cfg)
	if err != nil {
		return fmt.Errorf("provision row-level security: %w", err)
	}

	if result.PoliciesAdded > 0 {
		color.Yellow(
			"🔐 Applied tenant isolation policies to %d table(s) that a migration left unprotected",
			result.PoliciesAdded,
		)
	}
	if result.KeyInstalled {
		color.Green("🔐 Row-level security scope key %q is installed", m.cfg.Database.RLS.ScopeKeyID)
	}

	return nil
}

func (m *Manager) reconcileSeeds(ctx context.Context, opts common.OperationOptions) error {
	if opts.DryRun || m.seeder == nil {
		return nil
	}

	report, err := m.seeder.Reconcile(ctx, opts.Environment)
	if err != nil {
		return fmt.Errorf("reconcile seeds: %w", err)
	}
	if report != nil && !report.Success() {
		return fmt.Errorf("reconciling seeds failed %d times", report.Failed)
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
