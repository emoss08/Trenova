package db

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/migrations"
	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var dbEnableVectorCmd = &cobra.Command{
	Use:   "enable-vector",
	Short: "Install pgvector and the semantic retrieval vector tables",
	Long: `Create the pgvector extension (updating it when the installed version is older
than 0.8) and the tables, indexes and triggers semantic retrieval stores its
embeddings in. Run it after installing pgvector on a server whose migrations
already ran without it; until then semantic retrieval stays on keyword search.

It runs the same SQL as the ai_retrieval_vector migration and is safe to run
more than once. The database role needs permission to create the extension.

Examples:
  trenova db enable-vector              # Install and report the result
  trenova db enable-vector --dry-run    # Report what is installed now`,
	RunE: runEnableVector,
}

func runEnableVector(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	manager, err := createManager()
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	db := manager.GetDB()

	if dryRun {
		support, probeErr := dbdialect.ProbeVector(ctx, db)
		if probeErr != nil {
			return fmt.Errorf("failed to probe pgvector: %w", probeErr)
		}
		reportVectorSupport("Current state", support)
		return nil
	}

	result, err := migrations.EnableVector(ctx, db)
	if err != nil {
		if result.Before.State != "" {
			reportVectorSupport("Before", result.Before)
		}
		return fmt.Errorf("failed to enable vector search: %w", err)
	}

	reportVectorSupport("Before", result.Before)
	reportVectorSupport("After", result.After)
	color.Green(
		"✓ Semantic retrieval vector storage is ready. Running services pick it up within five minutes.",
	)

	return nil
}

func reportVectorSupport(label string, support dbdialect.VectorSupport) {
	version := support.ExtensionVersion
	if version == "" {
		version = "not installed"
	}

	color.Cyan("→ %s: %s (pgvector %s)", label, support.State, version)
}

func init() {
	DbCmd.AddCommand(dbEnableVectorCmd)
}
