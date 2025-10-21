package db

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	dryRun      bool
	force       bool
	verbose     bool
	interactive bool
	target      string
	cfg         *config.Config
)

func SetConfig(c *config.Config) {
	cfg = c
}

var DbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
	Long:  `Comprehensive database management for migrations, seeding, and maintenance`,
}

var dbMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run pending database migrations",
	Long: `Apply all pending database migrations to bring the database schema up to date.

Examples:
  trenova db migrate                    # Run migrations
  trenova db migrate --dry-run          # Preview migrations without applying
  trenova db migrate --verbose          # Show detailed output`,
	RunE: runMigrations,
}

var dbRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback database migrations",
	Long: `Rollback database migrations to a previous state.

Examples:
  trenova db rollback                   # Rollback last migration
  trenova db rollback --target 3        # Rollback 3 migrations
  trenova db rollback --dry-run         # Preview rollback without applying`,
	RunE: runRollback,
}

var dbStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration and seed status",
	Long:  `Display the current status of database migrations and applied seeds`,
	RunE:  showStatus,
}

var dbCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new migration",
	Long: `Create a new database migration file.

Examples:
  trenova db create add_user_table      # Create SQL migration
  trenova db create add_index --tx      # Create transactional migration`,
	Args: cobra.ExactArgs(1),
	RunE: createMigration,
}

var dbSeedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database",
	Long: `Apply seed data to the database based on the current environment.

Examples:
  trenova db seed                       # Apply seeds for current environment
  trenova db seed --force               # Force re-apply already applied seeds
  trenova db seed --verbose             # Show detailed output`,
	RunE: runSeeds,
}

var dbResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the database",
	Long: `Drop all tables and recreate from migrations (non-production only).

WARNING: This is a destructive operation that will delete all data!

Examples:
  trenova db reset                      # Reset with confirmation
  trenova db reset --force              # Skip confirmation`,
	RunE: resetDatabase,
}

var dbSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup database (migrate + seed)",
	Long: `Perform initial database setup by running migrations and applying base seeds.

This is typically used for initial deployment or setting up new environments.

Examples:
  trenova db setup                      # Setup with defaults
  trenova db setup --verbose            # Show detailed output`,
	RunE: setupDatabase,
}

func runMigrations(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	manager, err := seeder.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	opts := common.OperationOptions{
		DryRun:      dryRun,
		Force:       force,
		Verbose:     verbose,
		Interactive: interactive,
		Environment: getEnvironment(),
	}

	if err := manager.Migrate(ctx, opts); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

func runRollback(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	manager, err := seeder.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	opts := common.OperationOptions{
		DryRun:      dryRun,
		Force:       force,
		Verbose:     verbose,
		Interactive: interactive,
		Target:      target,
		Environment: getEnvironment(),
	}

	if err := manager.Rollback(ctx, opts); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	return nil
}

func showStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	manager, err := seeder.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	if err := manager.MigrationStatus(ctx); err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	fmt.Println()

	if err := manager.SeedStatus(ctx); err != nil {
		return fmt.Errorf("failed to get seed status: %w", err)
	}

	return nil
}

func createMigration(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	manager, err := seeder.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	transactional, _ := cmd.Flags().GetBool("tx")

	if err := manager.CreateMigration(ctx, args[0], transactional); err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	return nil
}

func runSeeds(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	manager, err := seeder.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	opts := common.OperationOptions{
		DryRun:      dryRun,
		Force:       force,
		Verbose:     verbose,
		Interactive: interactive,
		Environment: getEnvironment(),
	}

	if err := manager.Seed(ctx, opts); err != nil {
		return fmt.Errorf("seeding failed: %w", err)
	}

	return nil
}

func resetDatabase(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if cfg.IsProduction() {
		return fmt.Errorf("database reset is not allowed in production environment")
	}

	manager, err := seeder.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	opts := common.OperationOptions{
		Force:       force,
		Verbose:     verbose,
		Interactive: !force,
		Environment: getEnvironment(),
	}

	if err := manager.Reset(ctx, opts); err != nil {
		return fmt.Errorf("reset failed: %w", err)
	}

	if seedAfter, _ := cmd.Flags().GetBool("seed"); seedAfter {
		color.Cyan("→ Applying seeds after reset...")
		if err := manager.Seed(ctx, opts); err != nil {
			return fmt.Errorf("seeding after reset failed: %w", err)
		}
	}

	return nil
}

func setupDatabase(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	manager, err := seeder.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	opts := common.OperationOptions{
		Force:       force,
		Verbose:     verbose,
		Interactive: interactive,
		Environment: getEnvironment(),
	}

	if err := manager.Setup(ctx, opts); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	return nil
}

func getEnvironment() common.Environment {
	switch cfg.App.Env {
	case "production", "prod":
		return common.EnvProduction
	case "staging", "stage":
		return common.EnvStaging
	case "test", "testing":
		return common.EnvTest
	default:
		return common.EnvDevelopment
	}
}

func init() {
	DbCmd.PersistentFlags().
		BoolVar(&dryRun, "dry-run", false, "Preview changes without applying them")
	DbCmd.PersistentFlags().BoolVar(&force, "force", false, "Force operation without confirmation")
	DbCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Show detailed output")
	DbCmd.PersistentFlags().
		BoolVarP(&interactive, "interactive", "i", false, "Interactive mode with confirmations")

	DbCmd.AddCommand(dbMigrateCmd)
	DbCmd.AddCommand(dbRollbackCmd)
	dbRollbackCmd.Flags().StringVar(&target, "target", "", "Number of migrations to rollback")

	DbCmd.AddCommand(dbStatusCmd)
	DbCmd.AddCommand(dbCreateCmd)
	dbCreateCmd.Flags().Bool("tx", false, "Create transactional migration")

	DbCmd.AddCommand(dbSeedCmd)

	DbCmd.AddCommand(dbResetCmd)
	dbResetCmd.Flags().Bool("seed", false, "Apply seeds after reset")

	DbCmd.AddCommand(dbSetupCmd)
	DbCmd.AddCommand(createSeedCmd)
}
