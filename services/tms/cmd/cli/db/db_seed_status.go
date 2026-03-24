package db

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var dbSeedStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show seed application status",
	Long: `Display the status of all applied seeds including:
- Seed names and versions
- Application timestamps
- Seed environments

Examples:
  trenova db seed status    # Show all applied seeds`,
	RunE: runSeedStatus,
}

func init() {
	dbSeedCmd.AddCommand(dbSeedStatusCmd)
}

func runSeedStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	manager, err := createManager()
	if err != nil {
		return fmt.Errorf("create database manager: %w", err)
	}
	defer manager.Close()

	if err := manager.SeedStatus(ctx); err != nil {
		return fmt.Errorf("get seed status: %w", err)
	}

	return nil
}
