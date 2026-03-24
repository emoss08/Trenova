package db

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var dbSeedRollbackCmd = &cobra.Command{
	Use:   "rollback [seed-name]",
	Short: "Rollback a specific seed",
	Long: `Rollback a specific seed by removing all entities it created.

This command will:
1. Verify the seed exists and supports rollback
2. Check for dependent seeds (fails if any exist)
3. Delete all entities created by the seed (in reverse order)
4. Record the rollback in history

Examples:
  trenova db seed rollback FormulaTemplate  # Rollback specific seed
  trenova db seed rollback --dry-run        # Preview rollback
  trenova db seed rollback --all            # Rollback all seeds (reverse order)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSeedRollback,
}

var rollbackAll bool

func init() {
	dbSeedRollbackCmd.Flags().
		BoolVar(&rollbackAll, "all", false, "Rollback all seeds in reverse order")
}

func runSeedRollback(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	manager, err := createManager()
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	env := getEnvironment()

	if rollbackAll {
		return rollbackAllSeeds(ctx, manager, env)
	}

	if len(args) == 0 {
		return fmt.Errorf("seed name is required (use --all to rollback all seeds)")
	}

	seedName := args[0]

	color.Yellow("🔄 Rolling back seed: %s", seedName)

	if dryRun {
		color.Cyan("🔍 Dry run - no changes will be made")
	}

	if err := manager.RollbackSeed(ctx, seedName, env, dryRun); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	if !dryRun {
		color.Green("✓ Seed %s rolled back successfully", seedName)
	}

	return nil
}

func rollbackAllSeeds(ctx context.Context, manager any, env common.Environment) error {
	color.Yellow("🔄 Rolling back all seeds...")

	if dryRun {
		color.Cyan("🔍 Dry run - no changes will be made")
	}

	type rollbackManager interface {
		RollbackAllSeeds(ctx context.Context, env common.Environment, dryRun bool) error
	}

	rm, ok := manager.(rollbackManager)
	if !ok {
		return fmt.Errorf("manager does not support rollback all")
	}

	if err := rm.RollbackAllSeeds(ctx, env, dryRun); err != nil {
		return fmt.Errorf("rollback all failed: %w", err)
	}

	if !dryRun {
		color.Green("✓ All seeds rolled back successfully")
	}

	return nil
}
