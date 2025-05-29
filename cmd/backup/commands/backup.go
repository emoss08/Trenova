package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun/driver/pgdriver"
)

// BackupFileInfo contains information about a backup file
type BackupFileInfo struct {
	Filename  string    `json:"filename"`
	Path      string    `json:"path"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
	Database  string    `json:"database"`
}

var (
	// Flags
	cfgFile             string
	backupRetentionDays int
	cfg                 *config.Config
	manager             *config.Manager
	backupList          bool
	backupRestore       string
	backupCleanup       bool
	verbose             bool
	pgDumpPath          string
	pgRestorePath       string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "backup",
	Short: "Trenova Database Backup Tool",
	Long: `A command-line tool for managing Trenova database backups.

This tool provides commands to create, list, restore, and clean up database
backups without starting the full application server.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		manager, cfg = GetConfigManager()
		return nil
	},
}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new database backup",
	Run: func(cmd *cobra.Command, args []string) {
		path, err := createBackup()
		if err != nil {
			fmt.Printf("Error creating backup: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Backup created successfully: %s\n", path)

		if backupRetentionDays > 0 || cfg.Backup.RetentionDays > 0 {
			days := backupRetentionDays
			if days <= 0 {
				days = cfg.Backup.RetentionDays
			}
			fmt.Printf("Applying retention policy (%d days)...\n", days)
			if err := applyRetentionPolicy(days); err != nil {
				fmt.Printf("Error applying retention policy: %v\n", err)
			}
		}
	},
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	Run: func(cmd *cobra.Command, args []string) {
		backups, err := listBackups()
		if err != nil {
			fmt.Printf("Error listing backups: %v\n", err)
			os.Exit(1)
		}

		if len(backups) == 0 {
			fmt.Println("No backups found.")
			return
		}

		fmt.Println("Available backups:")
		fmt.Println("--------------------------------------------------")
		fmt.Printf("%-30s %-10s %-20s\n", "Filename", "Size", "Created")
		fmt.Println("--------------------------------------------------")

		for _, backup := range backups {
			size := formatSize(backup.SizeBytes)
			created := backup.CreatedAt.Format("2006-01-02 15:04:05")
			fmt.Printf("%-30s %-10s %-20s\n", backup.Filename, size, created)
		}
	},
}

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore [filename]",
	Short: "Restore database from a backup",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var filename string
		if len(args) > 0 {
			filename = args[0]
		} else if backupRestore != "" {
			filename = backupRestore
		} else {
			fmt.Println("Error: Backup filename is required")
			os.Exit(1)
		}

		// Ensure the file exists
		backupDir := cfg.Backup.BackupDir
		if backupDir == "" {
			backupDir = "./backups"
		}

		backupFile := filename
		if !filepath.IsAbs(backupFile) {
			backupFile = filepath.Join(backupDir, backupFile)
		}

		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			fmt.Printf("Error: Backup file %s not found\n", backupFile)
			os.Exit(1)
		}

		fmt.Printf("Restoring database from backup: %s\n", backupFile)
		fmt.Println("WARNING: This will overwrite the existing database!")
		fmt.Print("Are you sure? (y/N): ")

		var response string
		fmt.Scanln(&response)
		if !strings.EqualFold(response, "y") {
			fmt.Println("Restore cancelled.")
			return
		}

		if err := restoreBackup(backupFile); err != nil {
			fmt.Printf("Error restoring backup: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Database restored successfully!")
	},
}

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up old backups according to retention policy",
	Run: func(cmd *cobra.Command, args []string) {
		_, cfg := GetConfigManager()

		days := backupRetentionDays
		if days <= 0 {
			days = cfg.Backup.RetentionDays
		}
		if days <= 0 {
			days = 30 // Default if nothing is specified
		}

		fmt.Printf("Cleaning up backups older than %d days...\n", days)
		if err := applyRetentionPolicy(days); err != nil {
			fmt.Printf("Error cleaning up backups: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Cleanup completed successfully!")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is config/development/config.development.yaml)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose output")

	// Add flags to create command
	createCmd.Flags().IntVar(&backupRetentionDays, "retention-days", 0, "Number of days to keep backups (0 uses configured value)")

	// Add flags to restore command
	restoreCmd.Flags().StringVar(&backupRestore, "file", "", "Backup file to restore (if not provided as argument)")

	// Add flags to cleanup command
	cleanupCmd.Flags().IntVar(&backupRetentionDays, "days", 0, "Number of days to keep backups (0 uses configured value)")

	// Add commands to root
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(cleanupCmd)

	// Legacy command structure for backwards compatibility
	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Legacy backup command",
		Long:  "Legacy command for compatibility with old scripts",
		Run: func(cmd *cobra.Command, args []string) {
			switch {
			case backupList:
				listCmd.Run(cmd, args)
			case backupRestore != "":
				restoreCmd.Run(cmd, args)
			case backupCleanup:
				cleanupCmd.Run(cmd, args)
			default:
				createCmd.Run(cmd, args)
			}
		},
	}

	backupCmd.Flags().BoolVar(&backupList, "list", false, "List available backups")
	backupCmd.Flags().StringVar(&backupRestore, "restore", "", "Restore from the specified backup file")
	backupCmd.Flags().BoolVar(&backupCleanup, "cleanup", false, "Clean up old backups according to retention policy")
	backupCmd.Flags().IntVar(&backupRetentionDays, "retention-days", 0, "Number of days to keep backups (0 uses configured value)")

	rootCmd.AddCommand(backupCmd)

	// Verify PostgreSQL tools
	findPgTools()
}

func findPgTools() {
	var err error

	// Find pg_dump
	pgDumpPath, err = exec.LookPath("pg_dump")
	if err != nil {
		fmt.Println("WARNING: pg_dump not found in PATH. Make sure PostgreSQL client tools are installed.")
		// Try common locations
		locations := []string{
			"/usr/bin/pg_dump",
			"/usr/local/bin/pg_dump",
			"/usr/lib/postgresql/*/bin/pg_dump",
		}
		for _, loc := range locations {
			matches, _ := filepath.Glob(loc)
			if len(matches) > 0 {
				pgDumpPath = matches[0]
				break
			}
		}
	}

	// Find pg_restore
	pgRestorePath, err = exec.LookPath("pg_restore")
	if err != nil {
		fmt.Println("WARNING: pg_restore not found in PATH. Make sure PostgreSQL client tools are installed.")
		// Try common locations
		locations := []string{
			"/usr/bin/pg_restore",
			"/usr/local/bin/pg_restore",
			"/usr/lib/postgresql/*/bin/pg_restore",
		}
		for _, loc := range locations {
			matches, _ := filepath.Glob(loc)
			if len(matches) > 0 {
				pgRestorePath = matches[0]
				break
			}
		}
	}

	// Display found paths if verbose
	if verbose {
		if pgDumpPath != "" {
			fmt.Println("Using pg_dump from:", pgDumpPath)
		}
		if pgRestorePath != "" {
			fmt.Println("Using pg_restore from:", pgRestorePath)
		}
	}
}

func createBackup() (string, error) {
	// Ensure backup directory exists
	backupDir := cfg.Backup.BackupDir
	if backupDir == "" {
		backupDir = "./backups"
	}

	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Verify pg_dump is available
	if pgDumpPath == "" {
		return "", eris.New("pg_dump not found, please install PostgreSQL client tools")
	}

	// Create timestamp for backup file
	timestamp := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.sql.gz", cfg.DB.Database, timestamp)
	fp := filepath.Join(backupDir, filename)

	// Check PostgreSQL version
	serverVersion, err := getServerVersion()
	if err != nil {
		fmt.Printf("Warning: Unable to check PostgreSQL server version: %v\n", err)
		fmt.Println("Make sure your pg_dump version matches or is newer than your PostgreSQL server version!")
	} else {
		clientVersion, cvErr := getPgDumpVersion()
		if cvErr != nil {
			fmt.Printf("Warning: Unable to determine pg_dump version: %v\n", cvErr)
		} else if clientVersion < serverVersion {
			return "", fmt.Errorf("pg_dump version (%d) is older than PostgreSQL server version (%d). Please install pg_dump version %d or newer",
				clientVersion, serverVersion, serverVersion)
		}

		if verbose {
			fmt.Printf("Server PostgreSQL version: %d, Client pg_dump version: %d\n", serverVersion, clientVersion)
		}
	}

	// Build the pg_dump command
	args := []string{
		fmt.Sprintf("--host=%s", cfg.DB.Host),
		fmt.Sprintf("--port=%d", cfg.DB.Port),
		fmt.Sprintf("--username=%s", cfg.DB.Username),
		"--format=custom",
		"--compress=6",
		"--no-owner",
		"--no-privileges",
		fmt.Sprintf("--file=%s", fp),
		cfg.DB.Database,
	}

	cmd := exec.Command(pgDumpPath, args...)

	// Set PGPASSWORD environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.DB.Password))

	// Execute the command
	if verbose {
		fmt.Println("Executing pg_dump command...")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute pg_dump command: %w\n%s", err, string(output))
	}

	// Verify the backup file was created
	if _, err = os.Stat(fp); os.IsNotExist(err) {
		return "", fmt.Errorf("backup file was not created: %w", err)
	}

	return fp, nil
}

func listBackups() ([]BackupFileInfo, error) {
	backupDir := cfg.Backup.BackupDir
	if backupDir == "" {
		backupDir = "./backups"
	}

	// Read directory
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Collect backup files
	backups := make([]BackupFileInfo, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if !strings.HasSuffix(filename, ".sql.gz") {
			continue
		}

		filePath := filepath.Join(backupDir, filename)
		fileInfo, fiErr := file.Info()
		if fiErr != nil {
			// ! If the file is not found, it will be skipped
			continue
		}

		// Extract database name from filename
		dbName := "unknown"
		parts := strings.Split(filename, "-")
		if len(parts) > 0 {
			dbName = parts[0]
		}

		backups = append(backups, BackupFileInfo{
			Filename:  filename,
			Path:      filePath,
			SizeBytes: fileInfo.Size(),
			CreatedAt: fileInfo.ModTime(),
			Database:  dbName,
		})
	}

	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

func restoreBackup(backupFile string) error {
	// Ensure backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupFile)
	}

	// Verify pg_restore is available
	if pgRestorePath == "" {
		return eris.New("pg_restore not found, please install PostgreSQL client tools")
	}

	// Build the pg_restore command
	args := []string{
		fmt.Sprintf("--host=%s", cfg.DB.Host),
		fmt.Sprintf("--port=%d", cfg.DB.Port),
		fmt.Sprintf("--username=%s", cfg.DB.Username),
		"--clean",
		"--if-exists",
		"--no-owner",
		"--no-privileges",
		"--verbose",
		fmt.Sprintf("--dbname=%s", cfg.DB.Database),
		backupFile,
	}

	cmd := exec.Command(pgRestorePath, args...)

	// Set PGPASSWORD environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.DB.Password))

	// Execute the command
	if verbose {
		fmt.Println("Executing pg_restore command...")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute pg_restore command: %w\n%s", err, string(output))
	}

	return nil
}

func applyRetentionPolicy(retentionDays int) error {
	retentionDate := time.Now().AddDate(0, 0, -retentionDays)

	// Get all backups
	backups, err := listBackups()
	if err != nil {
		return err
	}

	// Delete files older than retention date
	deleteCount := 0
	for _, backup := range backups {
		if backup.CreatedAt.Before(retentionDate) {
			if verbose {
				fmt.Printf("Deleting old backup: %s (created: %s)\n",
					backup.Filename, backup.CreatedAt.Format("2006-01-02 15:04:05"))
			}

			if err = os.Remove(backup.Path); err != nil {
				fmt.Printf("Warning: Failed to delete backup %s: %v\n", backup.Path, err)
			} else {
				deleteCount++
			}
		}
	}

	if verbose || deleteCount > 0 {
		fmt.Printf("Deleted %d old backup(s).\n", deleteCount)
	}

	return nil
}

// formatSize formats a size in bytes to a human-readable string
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// getServerVersion attempts to connect to the PostgreSQL server and get its version
func getServerVersion() (int, error) {
	db := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(manager.GetDSN())))
	defer db.Close()

	// Check connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return 0, err
	}

	// Query server version
	var version int
	err := db.QueryRowContext(ctx, "SHOW server_version_num").Scan(&version)
	if err != nil {
		return 0, err
	}

	// Return major version (first two digits)
	return version / 10000, nil
}

// getPgDumpVersion gets the pg_dump version
func getPgDumpVersion() (int, error) {
	if pgDumpPath == "" {
		return 0, eris.New("pg_dump not found")
	}

	cmd := exec.Command(pgDumpPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	// Parse version string (format: pg_dump (PostgreSQL) 16.2)
	versionStr := string(output)
	parts := strings.Fields(versionStr)
	for _, part := range parts {
		if strings.Contains(part, ".") {
			versionParts := strings.Split(part, ".")
			if major, mErr := parse(versionParts[0]); mErr == nil {
				return major, nil
			}
		}
	}

	return 0, fmt.Errorf("unable to parse pg_dump version from: %s", versionStr)
}

// parse attempts to parse a string to an int
func parse(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func GetConfigManager() (*config.Manager, *config.Config) {
	cm := config.NewManager()

	cmCfg, err := cm.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	return cm, cmCfg
}
