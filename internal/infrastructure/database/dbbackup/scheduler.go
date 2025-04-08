package dbbackup

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
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
	BackupService services.BackupService
	Config        *config.Manager
}

type backupScheduler struct {
	logger    *zerolog.Logger
	bs        services.BackupService
	cfg       *config.BackupConfig
	scheduler *gocron.Scheduler
}

// NewBackupScheduler creates a new backup scheduler.
func NewBackupScheduler(p BackupSchedulerParams) (services.BackupScheduler, error) {
	log := p.Logger.With().
		Str("component", "backup_scheduler").
		Logger()

	// Check if backups are enabled in config
	backupCfg := p.Config.Backup()
	if backupCfg == nil || !backupCfg.Enabled {
		log.Info().Msg("backup scheduler is disabled because backup service is disabled")
		return nil, eris.New("backup scheduler is disabled because backup service is disabled")
	}

	// Check if backup service was successfully initialized
	if p.BackupService == nil {
		log.Error().Msg("backup scheduler cannot start because backup service is not initialized")
		return nil, eris.New("backup service not initialized")
	}

	scheduler := gocron.NewScheduler(time.UTC)

	bs := &backupScheduler{
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
func (bs *backupScheduler) Start(_ context.Context) error {
	if bs.cfg == nil || !bs.cfg.Enabled || bs.bs == nil {
		return nil // Skip if backups are disabled
	}

	// Get backup schedule from config
	schedule := bs.cfg.Schedule
	if schedule == "" {
		schedule = "0 0 * * *" // Default: daily at midnight
	}

	bs.logger.Info().
		Str("schedule", schedule).
		Int("retentionDays", bs.cfg.RetentionDays).
		Msg("starting backup scheduler")

	// Schedule backup job with cron expression
	_, err := bs.scheduler.Cron(schedule).Do(func() {
		// Create a new context with timeout for the backup operation
		timeout := bs.cfg.BackupTimeout
		if timeout <= 0 {
			timeout = 30 * time.Minute
		}

		c, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		bs.logger.Info().Msg("running scheduled backup")
		if err := bs.bs.ScheduledBackup(c, bs.cfg.RetentionDays); err != nil {
			bs.logger.Error().Err(err).Msg("scheduled backup failed")
			// * TODO(Wolfred): If notifications are enabled, this would be a good place to send an alert
		}
	})
	if err != nil {
		bs.logger.Error().Err(err).Msg("failed to schedule backup job")
		return err
	}

	// Start the scheduler in a background goroutine
	bs.scheduler.StartAsync()

	return nil
}

// Stop stops the backup scheduler.
func (bs *backupScheduler) Stop(_ context.Context) error {
	if bs.scheduler != nil {
		bs.logger.Info().Msg("stopping backup scheduler")
		bs.scheduler.Stop()
	}
	return nil
}

// RunNow triggers an immediate backup.
func (bs *backupScheduler) RunNow(ctx context.Context) error {
	if bs.bs == nil {
		return eris.New("backup service not initialized")
	}

	bs.logger.Info().Msg("running manual backup")
	retentionDays := 30
	if bs.cfg != nil {
		retentionDays = bs.cfg.RetentionDays
	}

	return bs.bs.ScheduledBackup(ctx, retentionDays)
}
