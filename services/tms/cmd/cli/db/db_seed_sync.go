package db

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var seedSyncCmd = &cobra.Command{
	Use:   "seed-sync",
	Short: "Synchronize seed registry with actual seed files",
	Long: `Regenerates the seed registry based on the current seed files in the filesystem.
Use this command when you:
- Create new seed files manually
- Delete seed files that haven't been applied
- Want to ensure the registry is in sync with actual files`,
	RunE: runSeedSync,
}

func init() {
	DbCmd.AddCommand(seedSyncCmd)
}

func runSeedSync(cmd *cobra.Command, args []string) error {
	color.Cyan("→ Synchronizing seed registry...")

	genCmd := exec.Command("go", "generate", "./internal/infrastructure/database/seeder/...")
	var stderr bytes.Buffer
	genCmd.Stderr = &stderr

	if err := genCmd.Run(); err != nil {
		color.Red("✗ Failed to synchronize registry: %v", err)
		if stderr.Len() > 0 {
			fmt.Printf("Error details: %s\n", stderr.String())
		}
		return fmt.Errorf("registry synchronization failed: %w", err)
	}

	color.Green("✓ Seed registry synchronized successfully")

	if verbose {
		fmt.Println("\nThe registry has been updated to match the current seed files.")
		fmt.Println(
			"Any seeds that were deleted from the filesystem have been removed from the registry.",
		)
		fmt.Println("Any new seeds that were added to the filesystem have been registered.")
	}

	return nil
}
