package db

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
)

var (
	dryRun  bool
	force   bool
	verbose bool
	cfg     *config.Config
)

func SetConfig(c *config.Config) {
	cfg = c
}

func openDB(ctx context.Context) (*bun.DB, error) {
	return postgres.OpenMaintenanceDB(ctx, cfg)
}

var DbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database maintenance commands",
	Long: `Bootstrap, row-level security and repairs. Migrations and seeds are applied by
trenova-db (services/tms/cmd/db) on every deploy, or trenova-devdb in development.`,
}

func init() {
	DbCmd.PersistentFlags().
		BoolVar(&dryRun, "dry-run", false, "Preview changes without applying them")
	DbCmd.PersistentFlags().BoolVar(&force, "force", false, "Force operation without confirmation")
	DbCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Show detailed output")
}
