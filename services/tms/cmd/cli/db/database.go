package db

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/spf13/cobra"
)

var (
	dryRun       bool
	force        bool
	verbose      bool
	interactive  bool
	envOverride  string
	ignoreErrors bool
	cfg          *config.Config
)

func SetConfig(c *config.Config) {
	cfg = c
}

func createManager() (*seeder.Manager, error) {
	registry := seeder.NewRegistry()
	seeds.Register(registry)

	return seeder.NewManagerFromConfig(seeder.ManagerConfig{
		Config:   cfg,
		Registry: registry,
	})
}

var DbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database seeding and maintenance commands",
	Long: `Seeding, reconciliation and repairs. Migrations are applied by trenova-db
(services/tms/cmd/db), or trenova-devdb in development.`,
}

var dbSeedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database",
	Long: `Apply seed data to the database based on the current environment.

Examples:
  trenova db seed                       # Apply seeds for current environment
  trenova db seed --env development     # Override environment
  trenova db seed --target USStates     # Run specific seed (with dependencies)
  trenova db seed --force               # Force re-apply already applied seeds
  trenova db seed --ignore-errors       # Continue on seed failure
  trenova db seed --dry-run             # Preview what would be applied`,
	RunE: runSeeds,
}

var dbReconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Reconcile the database with this build after migrating",
	Long: `Do what migrations cannot, after trenova-db up: install the configured
row-level security scope key, grant new permissions to administrator roles,
and re-run the repeatable seeds that are already applied.

Examples:
  trenova db reconcile                  # Reconcile for the configured environment
  trenova db reconcile --env production # Override the environment`,
	RunE: runReconcile,
}

func runReconcile(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	manager, err := createManager()
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	opts := common.OperationOptions{
		DryRun:      dryRun,
		Verbose:     verbose,
		Environment: getEnvironment(),
	}

	if err := manager.Reconcile(ctx, opts); err != nil {
		return fmt.Errorf("reconcile failed: %w", err)
	}

	return nil
}

func runSeeds(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	seedTarget, _ := cmd.Flags().GetString("target")

	manager, err := createManager()
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
		Target:      seedTarget,
	}

	if err := manager.Seed(ctx, opts); err != nil {
		return fmt.Errorf("seeding failed: %w", err)
	}

	return nil
}

func getEnvironment() common.Environment {
	env := cfg.App.Env
	if envOverride != "" {
		env = envOverride
	}

	switch env {
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

	DbCmd.AddCommand(dbReconcileCmd)
	dbReconcileCmd.Flags().
		StringVar(&envOverride, "env", "", "Override environment (development, test, staging, production)")

	DbCmd.AddCommand(dbSeedCmd)
	dbSeedCmd.Flags().
		StringVar(&envOverride, "env", "", "Override environment (development, test, staging, production)")
	dbSeedCmd.Flags().String("target", "", "Run specific seed by name (with its dependencies)")
	dbSeedCmd.Flags().BoolVar(&ignoreErrors, "ignore-errors", false, "Continue on seed failure")
	dbSeedCmd.AddCommand(dbSeedRollbackCmd)

	DbCmd.AddCommand(createSeedCmd)
}
