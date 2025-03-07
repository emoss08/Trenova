package dbbackup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/fileutils"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type BackupServiceParams struct {
	fx.In

	Logger *logger.Logger
	DB     db.Connection
	Config *config.Manager
}

type BackupService struct {
	logger           *zerolog.Logger
	db               db.Connection
	cfg              *config.BackupConfig
	backupDir        string
	pgDumpPath       string
	pgRestorePath    string
	compressionLevel int
	retentionDays    int
}

// BackupConfig holds configuration for the backup service.
type BackupConfig struct {
	BackupDir      string `yaml:"backup_dir"`
	Retention      int    `yaml:"retention_days"`
	CompressionOpt string `yaml:"compression"`
}

func NewBackupService(p BackupServiceParams) (*BackupService, error) {
	log := p.Logger.With().Str("component", "dbbackup_service").Logger()

	// * Check if backup is enabled
	cfg := p.Config.Backup()
	if cfg == nil || !cfg.Enabled {
		log.Info().Msg("backup is disabled")
		return nil, nil
	}

	// * Get backup directory from config
	backupDir := cfg.BackupDir
	if backupDir == "" {
		backupDir = "./backups"
	}

	// * Create the backup directory if it doesn't exist
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, eris.Wrapf(err, "failed to create backup directory: %s", backupDir)
	}

	// * Verify PostgreSQL tools are available
	pgDumpPath, err := exec.LookPath("pg_dump")
	if err != nil {
		return nil, eris.Wrap(err, "pg_dump binary not found in PATH, PostgreSQL client tools must be installed")
	}

	pgRestorePath, err := exec.LookPath("pg_restore")
	if err != nil {
		return nil, eris.Wrap(err,
			"pg_restore binary not found in PATH, PostgreSQL client tools must be installed")
	}

	// * Get compression level from config
	compressLvl := cfg.Compression
	if compressLvl < 1 || compressLvl > 9 {
		compressLvl = 6 // * Default to level 6 if invalid
	}

	// * Get retention days from config
	retentionDays := cfg.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 30 // * Default to 30 days if invalid
	}

	log.Info().
		Str("pgDumpPath", pgDumpPath).
		Str("pgRestorePath", pgRestorePath).
		Str("backupDir", backupDir).
		Int("compressionLevel", compressLvl).
		Int("retentionDays", retentionDays).
		Msg("🚀 Backup service initialized successfully")

	return &BackupService{
		logger:           &log,
		db:               p.DB,
		cfg:              cfg,
		backupDir:        backupDir,
		pgDumpPath:       pgDumpPath,
		pgRestorePath:    pgRestorePath,
		compressionLevel: compressLvl,
		retentionDays:    retentionDays,
	}, nil
}

// CreateBackup performs a full database backup using pg_dump
func (s *BackupService) CreateBackup(ctx context.Context) (string, error) {
	log := s.logger.With().
		Str("operation", "CreateBackup").
		Logger()

	// * Get database connection parameters
	dbInfo, err := s.db.ConnectionInfo(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection info")
		return "", eris.Wrap(err, "failed to get database connection info")
	}

	// * Create timestamp for backup file
	timestamp := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.sql.gz", dbInfo.Database, timestamp)
	fp := filepath.Join(s.backupDir, filename)

	// * Build the pg_dump command with full path
	cmd := exec.CommandContext(
		ctx,
		s.pgDumpPath,
		fmt.Sprintf("--host=%s", dbInfo.Host),
		fmt.Sprintf("--port=%d", dbInfo.Port),
		fmt.Sprintf("--username=%s", dbInfo.Username),
		"--format=custom",
		fmt.Sprintf("--compress=%d", s.compressionLevel),
		"--no-owner",
		"--no-privileges",
		fmt.Sprintf("--file=%s", fp),
		dbInfo.Database,
	)

	// * Set the PGPASSWORD environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbInfo.Password))

	// * Execute the pg_dump command
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().
			Err(err).
			Str("output", string(output)).
			Msg("failed to execute pg_dump command")
		return "", eris.Wrapf(err, "failed to execute pg_dump command: %s", string(output))
	}

	// * Verify the backup file was created successfully
	if _, statErr := os.Stat(fp); os.IsNotExist(statErr) {
		return "", eris.Wrap(statErr, "backup file was not created")
	}

	log.Info().
		Str("filename", filename).
		Str("path", fp).
		Int64("sizeBytes", fileutils.GetFileSize(fp)).
		Msg("database backup created successfully")

	return fp, nil
}

// ScheduledBackup performs a backup and handles retention.
// If retentionDays is set to 0, it will use the configured retention period.
func (s *BackupService) ScheduledBackup(ctx context.Context, retentionDays int) error {
	// * Use configured retention days if not explicitly specified
	if retentionDays <= 0 {
		retentionDays = s.retentionDays
	}
	log := s.logger.With().
		Str("operation", "ScheduledBackup").
		Int("retentionDays", retentionDays).
		Logger()

	// * Create backup
	backupPath, err := s.CreateBackup(ctx)
	if err != nil {
		return err
	}

	// * Apply retention policy
	if err := s.ApplyRetentionPolicy(retentionDays); err != nil {
		log.Error().
			Err(err).
			Msg("failed to apply retention policy")
		return eris.Wrap(err, "failed to apply backup retention policy")
	}

	log.Info().
		Str("backupPath", backupPath).
		Msg("scheduled backup completed successfully")

	return nil
}

// ApplyRetentionPolicy deletes backups older than the retention period.
// This can be called directly or used by the scheduler.
func (s *BackupService) ApplyRetentionPolicy(retentionDays int) error {
	log := s.logger.With().
		Str("operation", "applyRetentionPolicy").
		Int("retentionDays", retentionDays).
		Logger()

	// * Calculate retention date
	retentionDate := time.Now().AddDate(0, 0, -retentionDays)

	// * List backup files
	files, err := os.ReadDir(s.backupDir)
	if err != nil {
		return eris.Wrap(err, "failed to read backup directory")
	}

	// * Delete files older than the retention date
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// * Parse the timestamp from the filename
		// * Expected format: database-YYYYMMDD-HHMMSS.sql.gz
		name := file.Name()
		if !strings.HasSuffix(name, ".sql.gz") {
			continue
		}

		parts := strings.Split(name, "-")
		if len(parts) < 3 {
			continue
		}

		dateStr := parts[len(parts)-2]
		timeStr := strings.Split(parts[len(parts)-1], ".")[0]
		timestamp, parseErr := time.Parse("20060102-150405", dateStr+"-"+timeStr)
		if parseErr != nil {
			log.Warn().
				Err(parseErr).
				Str("filename", name).
				Msg("failed to parse timestamp from filename")
			continue
		}

		// * Delete the file if it's older than the retention date
		if timestamp.Before(retentionDate) {
			fp := filepath.Join(s.backupDir, name)
			if removeErr := os.Remove(fp); removeErr != nil {
				log.Error().
					Err(removeErr).
					Str("filepath", fp).
					Msg("failed to delete old backup file")
			} else {
				log.Info().
					Str("filepath", fp).
					Time("fileDate", timestamp).
					Msg("deleted old backup file")
			}
		}
	}

	return nil
}

// RestoreBackup restores a database from a backup file.
func (s *BackupService) RestoreBackup(ctx context.Context, backupFile string) error {
	log := s.logger.With().
		Str("operation", "RestoreBackup").
		Str("backupFile", backupFile).
		Logger()

	// * Validate the backup file
	fileInfo, err := os.Stat(backupFile)
	if os.IsNotExist(err) {
		return eris.New("backup file does not exist")
	}

	// * Check if the file is empty
	if fileInfo.Size() == 0 {
		return eris.New("backup file is empty")
	}

	// * Get database connection parameters
	dbInfo, err := s.db.ConnectionInfo(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection info")
		return eris.Wrap(err, "failed to get database connection info for restore")
	}

	// * Log warning about restore operation
	log.Warn().
		Str("backupFile", backupFile).
		Str("database", dbInfo.Database).
		Msg("starting database restore - this will overwrite existing data")

	// * Build the pg_restore command with full path
	cmd := exec.CommandContext(
		ctx,
		s.pgRestorePath,
		fmt.Sprintf("--host=%s", dbInfo.Host),
		fmt.Sprintf("--port=%d", dbInfo.Port),
		fmt.Sprintf("--username=%s", dbInfo.Username),
		"--clean",
		"--if-exists",
		"--no-owner",
		"--no-privileges",
		"--verbose",
		fmt.Sprintf("--dbname=%s", dbInfo.Database),
		backupFile,
	)

	// * Set the PGPASSWORD environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbInfo.Password))

	// * Execute the pg_restore command
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().
			Err(err).
			Str("output", string(output)).
			Msg("failed to execute pg_restore command")
		return eris.Wrapf(err, "failed to execute pg_restore command: %s", string(output))
	}

	log.Info().
		Str("backupFile", backupFile).
		Str("output", string(output)).
		Msg("database restored successfully")

	return nil
}

// BackupFileInfo contains information about a backup file
type BackupFileInfo struct {
	Filename  string    `json:"filename"`
	Path      string    `json:"path"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
	Database  string    `json:"database"`
}

// ListBackups returns detailed information about available backup files.
func (s *BackupService) ListBackups() ([]BackupFileInfo, error) {
	log := s.logger.With().
		Str("operation", "ListBackups").
		Logger()

	// * List backup files
	files, err := os.ReadDir(s.backupDir)
	if err != nil {
		log.Error().
			Err(err).
			Str("backupDir", s.backupDir).
			Msg("failed to read backup directory")
		return nil, eris.Wrap(err, "failed to read backup directory")
	}

	// * Filter and collect backup file information
	backups := make([]BackupFileInfo, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if !strings.HasSuffix(filename, ".sql.gz") {
			continue
		}

		filePath := filepath.Join(s.backupDir, filename)
		fileInfo, err := file.Info()
		if err != nil {
			log.Warn().
				Err(err).
				Str("filename", filename).
				Msg("failed to get file info")
			continue
		}

		// * Extract database name from filename (format: database-YYYYMMDD-HHMMSS.sql.gz)
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

	// * Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	log.Info().
		Int("count", len(backups)).
		Msg("backup files retrieved successfully")

	return backups, nil
}

// DeleteBackup deletes a backup file.
func (s *BackupService) DeleteBackup(backupPath string) error {
	log := s.logger.With().
		Str("operation", "DeleteBackup").
		Str("backupPath", backupPath).
		Logger()

	// * Verify the file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return eris.New("backup file does not exist")
	}

	// * Delete the file
	if err := os.Remove(backupPath); err != nil {
		log.Error().
			Err(err).
			Msg("failed to delete backup file")
		return eris.Wrap(err, "failed to delete backup file")
	}

	log.Info().
		Msg("backup file deleted successfully")

	return nil
}

// GetRetentionDays returns the configured retention days.
func (s *BackupService) GetRetentionDays() int {
	return s.retentionDays
}

// GetBackupDir returns the backup directory path.
func (s *BackupService) GetBackupDir() string {
	return s.backupDir
}
