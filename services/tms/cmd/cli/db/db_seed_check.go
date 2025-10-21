package db

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
)

var seedCheckCmd = &cobra.Command{
	Use:   "seed-check",
	Short: "Check for deleted seeds and clean up seed history",
	Long: `Checks for seeds that have been deleted from the filesystem but still exist in the seed_history table.
Optionally removes these orphaned entries from the database.`,
	RunE: runSeedCheck,
}

var seedCleanCmd = &cobra.Command{
	Use:   "seed-clean",
	Short: "Clean up orphaned seed history entries",
	Long:  `Removes seed history entries for seeds that no longer exist in the filesystem.`,
	RunE:  runSeedClean,
}

func init() {
	DbCmd.AddCommand(seedCheckCmd)
	DbCmd.AddCommand(seedCleanCmd)
}

func runSeedCheck(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	existingSeeds, err := getExistingSeeds()
	if err != nil {
		return fmt.Errorf("get existing seeds: %w", err)
	}

	manager, err := seeder.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("create database manager: %w", err)
	}
	defer manager.Close()

	db := manager.GetDB()
	var appliedSeeds []struct {
		Name      string `bun:"name"`
		Version   string `bun:"version"`
		AppliedAt string `bun:"applied_at"`
	}

	err = db.NewSelect().
		Table("seed_history").
		Column("name", "version", "applied_at").
		Where("status = ?", "Active").
		Order("applied_at ASC").
		Scan(ctx, &appliedSeeds)
	if err != nil {
		return fmt.Errorf("get seed history: %w", err)
	}

	var orphaned []string
	for _, applied := range appliedSeeds {
		found := false
		for _, existing := range existingSeeds {
			if applied.Name == existing {
				found = true
				break
			}
		}
		if !found {
			orphaned = append(orphaned, applied.Name)
		}
	}

	if len(orphaned) == 0 {
		color.Green("✓ No orphaned seed history entries found")
		return nil
	}

	color.Yellow("⚠ Found %d orphaned seed history entries:", len(orphaned))
	for _, name := range orphaned {
		fmt.Printf("  - %s\n", name)
	}

	fmt.Println()
	fmt.Println("These seeds exist in the database history but not in the filesystem.")
	fmt.Println("Run 'trenova db seed-clean' to remove these entries.")

	return nil
}

func runSeedClean(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	existingSeeds, err := getExistingSeeds()
	if err != nil {
		return fmt.Errorf("get existing seeds: %w", err)
	}

	manager, err := seeder.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("create database manager: %w", err)
	}
	defer manager.Close()

	db := manager.GetDB()

	var orphaned []string
	err = db.NewSelect().
		Table("seed_history").
		Column("name").
		Where("status = ?", "Active").
		Scan(ctx, &orphaned)
	if err != nil {
		return fmt.Errorf("get seed history: %w", err)
	}

	var toDelete []string
	for _, name := range orphaned {
		found := false
		for _, existing := range existingSeeds {
			if name == existing {
				found = true
				break
			}
		}
		if !found {
			toDelete = append(toDelete, name)
		}
	}

	if len(toDelete) == 0 {
		color.Green("✓ No orphaned seed history entries to clean")
		return nil
	}

	if !force {
		color.Yellow("⚠ About to delete %d orphaned seed history entries:", len(toDelete))
		for _, name := range toDelete {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Print("\nContinue? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted")
			return nil
		}
	}

	result, err := db.NewUpdate().
		Table("seed_history").
		Set("status = ?", "Orphaned").
		Set("notes = ?", "Seed file was deleted from filesystem").
		Where("name IN (?)", bun.In(toDelete)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update seed history: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	color.Green("✓ Marked %d orphaned seed history entries as orphaned", rowsAffected)

	color.Cyan("→ Regenerating seed registry...")
	if err := regenerateRegistry(); err != nil {
		color.Yellow("⚠ Failed to regenerate registry: %v", err)
		fmt.Println(
			"Manually regenerate with: go generate ./internal/infrastructure/database/seeds/...",
		)
	} else {
		color.Green("✓ Registry updated")
	}

	return nil
}

func getExistingSeeds() ([]string, error) {
	var seeds []string

	dirs := []string{
		"./internal/infrastructure/database/seeds/base",
		"./internal/infrastructure/database/seeds/development",
		"./internal/infrastructure/database/seeds/test",
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}

			content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}

			scanner := bufio.NewScanner(strings.NewReader(string(content)))
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, "type ") && strings.Contains(line, "Seed struct") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						seedName := strings.TrimSuffix(parts[1], "Seed")
						seeds = append(seeds, seedName)
						break
					}
				}
			}
		}
	}

	return seeds, nil
}
