package dbbackup

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/go-co-op/gocron"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"

	"go.uber.org/fx"
)

type BackupSchedulerParams struct {
	fx.In

	LC            fx.Lifecycle
	Logger        *logger.Logger
	BackupService *BackupService
	Config        *config.Manager
}

type BackupScheduler struct {
	logger    *zerolog.Logger
	bs        *BackupService
	cfg       *config.BackupConfig
	scheduler *gocron.Scheduler
}

// NewBackupScheduler creates a new backup scheduler.
func NewBackupScheduler(p BackupSchedulerParams) (*BackupScheduler, error) {
	log := p.Logger.With().
		Str("component", "backup_scheduler").
		Logger()

	// Check if backups are enabled in config
	backupCfg := p.Config.Backup()
	if backupCfg == nil || !backupCfg.Enabled {
		log.Info().Msg("backup scheduler is disabled because backup service is disabled")
		return nil, nil
	}

	// Check if backup service was successfully initialized
	if p.BackupService == nil {
		log.Error().Msg("backup scheduler cannot start because backup service is not initialized")
		return nil, eris.New("backup service not initialized")
	}

	scheduler := gocron.NewScheduler(time.UTC)

	bs := &BackupScheduler{
		logger:    &log,
		bs:        p.BackupService,
		cfg:       backupCfg,
		scheduler: scheduler,
	}

	// Register lifecycle hooks
	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return bs.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return bs.Stop(ctx)
		},
	})

	return bs, nil
}

// Start starts the backup scheduler.
func (s *BackupScheduler) Start(ctx context.Context) error {
	if s.cfg == nil || !s.cfg.Enabled || s.bs == nil {
		return nil // Skip if backups are disabled
	}

	// Get backup schedule from config
	schedule := s.cfg.Schedule
	if schedule == "" {
		schedule = "0 0 * * *" // Default: daily at midnight
	}

	s.logger.Info().
		Str("schedule", schedule).
		Int("retentionDays", s.cfg.RetentionDays).
		Msg("starting backup scheduler")

	// Schedule backup job with cron expression
	_, err := s.scheduler.Cron(schedule).Do(func() {
		// Create a new context with timeout for the backup operation
		timeout := s.cfg.BackupTimeout
		if timeout <= 0 {
			timeout = 30 * time.Minute
		}

		c, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		s.logger.Info().Msg("running scheduled backup")
		if err := s.bs.ScheduledBackup(c, s.cfg.RetentionDays); err != nil {
			s.logger.Error().Err(err).Msg("scheduled backup failed")
			// * TODO(Wolfred): If notifications are enabled, this would be a good place to send an alert
		}
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to schedule backup job")
		return err
	}

	// Start the scheduler in a background goroutine
	s.scheduler.StartAsync()

	return nil
}

// Stop stops the backup scheduler.
func (s *BackupScheduler) Stop(ctx context.Context) error {
	if s.scheduler != nil {
		s.logger.Info().Msg("stopping backup scheduler")
		s.scheduler.Stop()
	}
	return nil
}

// RunNow triggers an immediate backup.
func (s *BackupScheduler) RunNow(ctx context.Context) error {
	if s.bs == nil {
		return eris.New("backup service not initialized")
	}

	s.logger.Info().Msg("running manual backup")
	retentionDays := 30
	if s.cfg != nil {
		retentionDays = s.cfg.RetentionDays
	}

	return s.bs.ScheduledBackup(ctx, retentionDays)
}
