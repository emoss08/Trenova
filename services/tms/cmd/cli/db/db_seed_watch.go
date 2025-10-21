package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var seedWatchCmd = &cobra.Command{
	Use:   "seed-watch",
	Short: "Watch seed directories and auto-update registry",
	Long: `Watches the seed directories for changes and automatically regenerates
the seed registry when seeds are added, modified, or deleted.`,
	RunE: runSeedWatch,
}

func init() {
	DbCmd.AddCommand(seedWatchCmd)
}

func runSeedWatch(cmd *cobra.Command, args []string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer watcher.Close()

	seedDirs := []string{
		"./internal/infrastructure/database/seeds/base",
		"./internal/infrastructure/database/seeds/development",
		"./internal/infrastructure/database/seeds/test",
	}

	for _, dir := range seedDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}

		if err := watcher.Add(dir); err != nil {
			return fmt.Errorf("watch directory %s: %w", dir, err)
		}
	}

	color.Cyan("👁 Watching seed directories for changes...")
	color.Yellow("Press Ctrl+C to stop")
	fmt.Println()

	var lastRegen time.Time
	regenDebounce := 2 * time.Second

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if !strings.HasSuffix(event.Name, ".go") {
				continue
			}

			if strings.Contains(event.Name, "seed_registry.go") {
				continue
			}

			var action string
			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				action = "created"
				color.Green("✓ Seed created: %s", filepath.Base(event.Name))
			case event.Op&fsnotify.Write == fsnotify.Write:
				action = "modified"
				color.Blue("✓ Seed modified: %s", filepath.Base(event.Name))
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				action = "deleted"
				color.Red("✓ Seed deleted: %s", filepath.Base(event.Name))
			case event.Op&fsnotify.Rename == fsnotify.Rename:
				action = "renamed"
				color.Yellow("✓ Seed renamed: %s", filepath.Base(event.Name))
			default:
				continue
			}

			if time.Since(lastRegen) < regenDebounce {
				continue
			}

			color.Cyan("→ Updating seed registry...")
			if err := regenerateRegistry(); err != nil {
				color.Red("✗ Failed to update registry: %v", err)
			} else {
				color.Green("✓ Registry updated after %s", action)
				lastRegen = time.Now()
			}

			fmt.Println()

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			color.Red("✗ Watcher error: %v", err)
		}
	}
}
