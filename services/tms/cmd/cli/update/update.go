package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/system"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/spf13/cobra"
)

var cfg *config.Config

var errNoReleaseFound = errors.New("no release found")

func SetConfig(c *config.Config) {
	cfg = c
}

var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Manage Trenova updates",
	Long:  `Commands for checking and applying Trenova updates.`,
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for available updates",
	Long:  `Check GitHub releases for available updates.`,
	RunE:  runCheck,
}

var applyCmd = &cobra.Command{
	Use:   "apply [version]",
	Short: "Apply an update",
	Long: `Apply an update to the specified version.
If no version is specified, updates to the latest version.

This command will:
1. Create a backup of current configuration
2. Pull new Docker images from ghcr.io
3. Run database migrations
4. Restart services`,
	Args: cobra.MaximumNArgs(1),
	RunE: runApply,
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback to previous version",
	Long:  `Rollback to the previously installed version using the last backup.`,
	RunE:  runRollback,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current version and update status",
	RunE:  runStatus,
}

var (
	skipBackup     bool
	skipMigrations bool
	forceUpdate    bool
	composeFile    string
)

func init() {
	UpdateCmd.AddCommand(checkCmd)
	UpdateCmd.AddCommand(applyCmd)
	UpdateCmd.AddCommand(rollbackCmd)
	UpdateCmd.AddCommand(statusCmd)

	applyCmd.Flags().
		BoolVar(&skipBackup, "skip-backup", false, "Skip creating backup before update")
	applyCmd.Flags().BoolVar(&skipMigrations, "skip-migrations", false, "Skip database migrations")
	applyCmd.Flags().
		BoolVar(&forceUpdate, "force", false, "Force update even if already on latest version")
	applyCmd.Flags().
		StringVar(&composeFile, "compose-file", "docker-compose.prod.yml", "Docker compose file to use")
}

func runCheck(_ *cobra.Command, _ []string) error {
	fmt.Println("Checking for updates...")

	release, err := fetchLatestRelease()
	if err != nil {
		if errors.Is(err, errNoReleaseFound) {
			fmt.Println("No releases found.")
			return nil
		}
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	currentVersion := cfg.App.Version
	if currentVersion == "" {
		currentVersion = "unknown"
	}

	fmt.Printf("\nCurrent version: %s\n", currentVersion)
	fmt.Printf("Latest version:  %s\n", release.Version)

	if isNewerVersion(release.Version, currentVersion) {
		fmt.Println("\n✓ Update available!")
		fmt.Printf("\nRelease Notes:\n%s\n", truncateNotes(release.ReleaseNotes, 500))
		fmt.Printf("\nTo update, run: trenova update apply\n")
	} else {
		fmt.Println("\n✓ You are running the latest version.")
	}

	return nil
}

func runApply(_ *cobra.Command, args []string) error {
	var targetVersion string
	if len(args) > 0 {
		targetVersion = args[0]
	}

	currentVersion := cfg.App.Version
	fmt.Printf("Current version: %s\n", currentVersion)

	if targetVersion == "" {
		release, err := fetchLatestRelease()
		if err != nil {
			return fmt.Errorf("failed to fetch latest release: %w", err)
		}
		targetVersion = release.Version
	}

	targetVersion = strings.TrimPrefix(targetVersion, "v")

	fmt.Printf("Target version:  %s\n", targetVersion)

	if !forceUpdate && !isNewerVersion(targetVersion, currentVersion) {
		fmt.Println("\nAlready on the latest version. Use --force to update anyway.")
		return nil
	}

	if !skipBackup {
		fmt.Println("\nCreating backup...")
		if err := createBackup(currentVersion); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Println("✓ Backup created")
	}

	fmt.Println("\nPulling new images from ghcr.io...")
	if err := pullImages(targetVersion); err != nil {
		return fmt.Errorf("failed to pull images: %w", err)
	}
	fmt.Println("✓ Images pulled successfully")

	fmt.Println("\nStopping current services...")
	if err := stopServices(); err != nil {
		fmt.Printf("⚠ Warning: %v\n", err)
	}

	if !skipMigrations {
		fmt.Println("\nRunning database migrations...")
		if err := runMigrations(targetVersion); err != nil {
			fmt.Printf("⚠ Migration warning: %v\n", err)
			fmt.Println("Continuing with update...")
		} else {
			fmt.Println("✓ Migrations complete")
		}
	}

	fmt.Println("\nStarting services with new version...")
	if err := startServices(targetVersion); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}
	fmt.Println("✓ Services started")

	fmt.Println("\nWaiting for health checks...")
	if err := waitForHealthy(); err != nil {
		fmt.Printf("⚠ Health check warning: %v\n", err)
	} else {
		fmt.Println("✓ All services healthy")
	}

	fmt.Printf("\n✓ Update to %s complete!\n", targetVersion)
	return nil
}

func runRollback(_ *cobra.Command, _ []string) error {
	backupDir := getBackupDir()
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("no backups found: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no backups available for rollback")
	}

	var latestBackup string
	var latestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() {
			info, infoErr := entry.Info()
			if infoErr != nil {
				continue
			}
			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestBackup = entry.Name()
			}
		}
	}

	if latestBackup == "" {
		return fmt.Errorf("no valid backup found")
	}

	backupPath := filepath.Join(backupDir, latestBackup)
	fmt.Printf("Rolling back to backup: %s\n", latestBackup)

	parts := strings.Split(latestBackup, "-")
	if len(parts) < 1 {
		return fmt.Errorf("invalid backup name format")
	}
	previousVersion := parts[0]

	envBackup := filepath.Join(backupPath, ".env")
	if _, statErr := os.Stat(envBackup); statErr == nil {
		if copyErr := copyFile(envBackup, ".env"); copyErr != nil {
			return fmt.Errorf("failed to restore .env: %w", copyErr)
		}
		fmt.Println("✓ Restored .env")
	}

	configBackup := filepath.Join(backupPath, "config")
	if _, statErr := os.Stat(configBackup); statErr == nil {
		if copyErr := copyDir(configBackup, "config"); copyErr != nil {
			return fmt.Errorf("failed to restore config: %w", copyErr)
		}
		fmt.Println("✓ Restored config directory")
	}

	fmt.Printf("\nRestarting services with version %s...\n", previousVersion)
	if err = stopServices(); err != nil {
		fmt.Printf("⚠ Warning stopping services: %v\n", err)
	}
	if err = startServices(previousVersion); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	fmt.Println("\n✓ Rollback complete!")
	return nil
}

func runStatus(_ *cobra.Command, _ []string) error {
	currentVersion := cfg.App.Version
	if currentVersion == "" {
		currentVersion = "unknown"
	}

	fmt.Printf("Trenova TMS\n")
	fmt.Printf("===========\n")
	fmt.Printf("Version:     %s\n", currentVersion)
	fmt.Printf("Environment: %s\n", cfg.App.Env)

	release, err := fetchLatestRelease()
	if err != nil {
		if !errors.Is(err, errNoReleaseFound) {
			fmt.Printf("\nUnable to check for updates: %v\n", err)
		}
		return nil
	}

	fmt.Printf("\nLatest available: %s\n", release.Version)
	if isNewerVersion(release.Version, currentVersion) {
		fmt.Println("Status: Update available")
	} else {
		fmt.Println("Status: Up to date")
	}

	backupDir := getBackupDir()
	entries, err := os.ReadDir(backupDir)
	if err == nil && len(entries) > 0 {
		fmt.Printf("\nBackups: %d available\n", len(entries))
	}

	return nil
}

func fetchLatestRelease() (*system.ReleaseInfo, error) {
	owner := "emoss08"
	repo := "trenova"
	if cfg != nil && cfg.Update.GitHubOwner != "" {
		owner = cfg.Update.GitHubOwner
	}
	if cfg != nil && cfg.Update.GitHubRepo != "" {
		repo = cfg.Update.GitHubRepo
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Trenova-CLI")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errNoReleaseFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var ghRelease system.GitHubRelease
	if err := sonic.Unmarshal(body, &ghRelease); err != nil {
		return nil, err
	}

	if ghRelease.Draft {
		return nil, errNoReleaseFound
	}

	publishedAt, _ := time.Parse(time.RFC3339, ghRelease.PublishedAt)

	return &system.ReleaseInfo{
		Version:      strings.TrimPrefix(ghRelease.TagName, "v"),
		TagName:      ghRelease.TagName,
		PublishedAt:  publishedAt.Unix(),
		ReleaseNotes: ghRelease.Body,
		HTMLURL:      ghRelease.HTMLURL,
		IsPrerelease: ghRelease.Prerelease,
	}, nil
}

func isNewerVersion(latest, current string) bool {
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	if latest == current {
		return false
	}

	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		latestNum, err := strconv.Atoi(latestParts[i])
		if err != nil {
			latestNum = 0
		}
		currentNum, err := strconv.Atoi(currentParts[i])
		if err != nil {
			currentNum = 0
		}

		if latestNum > currentNum {
			return true
		}
		if latestNum < currentNum {
			return false
		}
	}

	return len(latestParts) > len(currentParts)
}

func createBackup(version string) error {
	backupDir := getBackupDir()
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s-%s", version, timestamp))
	if err := os.MkdirAll(backupPath, 0o755); err != nil {
		return err
	}

	if _, err := os.Stat(".env"); err == nil {
		if copyErr := copyFile(".env", filepath.Join(backupPath, ".env")); copyErr != nil {
			return fmt.Errorf("failed to backup .env: %w", copyErr)
		}
	}

	if _, err := os.Stat("config"); err == nil {
		if copyErr := copyDir("config", filepath.Join(backupPath, "config")); copyErr != nil {
			return fmt.Errorf("failed to backup config: %w", copyErr)
		}
	}

	return nil
}

func pullImages(version string) error {
	cmd := exec.Command("docker", "compose", "-f", composeFile, "pull")
	cmd.Env = append(os.Environ(), fmt.Sprintf("TRENOVA_VERSION=%s", version))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func stopServices() error {
	cmd := exec.Command("docker", "compose", "-f", composeFile, "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func startServices(version string) error {
	cmd := exec.Command("docker", "compose", "-f", composeFile, "up", "-d")
	cmd.Env = append(os.Environ(), fmt.Sprintf("TRENOVA_VERSION=%s", version))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runMigrations(version string) error {
	cmd := exec.Command("docker", "compose", "-f", composeFile, "run", "--rm", "tms-migrate")
	cmd.Env = append(os.Environ(), fmt.Sprintf("TRENOVA_VERSION=%s", version))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitForHealthy() error {
	maxAttempts := 30
	for range maxAttempts {
		cmd := exec.Command("docker", "compose", "-f", composeFile, "ps", "--format", "json")
		output, err := cmd.Output()
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		if strings.Contains(string(output), "\"Health\":\"healthy\"") ||
			strings.Contains(string(output), "(healthy)") {
			return nil
		}

		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("services did not become healthy within timeout")
}

func getBackupDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".trenova", "backups")
}

func truncateNotes(notes string, maxLen int) string {
	if len(notes) <= maxLen {
		return notes
	}
	return notes[:maxLen] + "..."
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}
