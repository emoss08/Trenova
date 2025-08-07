/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package scheduler

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// CronSchedulerParams defines dependencies for the cron scheduler
type CronSchedulerParams struct {
	fx.In

	Logger     *logger.Logger
	JobService services.JobService
	RedisOpt   asynq.RedisClientOpt
	Config     *config.Manager
}

// CronScheduler manages scheduled recurring jobs
type CronScheduler struct {
	logger     *zerolog.Logger
	scheduler  *asynq.Scheduler
	jobService services.JobService
	cfg        config.CronSchedulerConfig
}

// CronSchedulerInterface defines methods for managing scheduled jobs
type CronSchedulerInterface interface {
	Start() error
	Stop() error
	SchedulePatternAnalysisJobs() error
	ScheduleShipmentJobs() error
	ScheduleSystemJobs() error
	ScheduleComplianceJobs() error
	ScheduleEmailQueueJobs() error
}

// NewCronScheduler creates a new cron scheduler
//
//nolint:gocritic // this is dependency injection
func NewCronScheduler(p CronSchedulerParams) CronSchedulerInterface {
	log := p.Logger.With().
		Str("service", "cron_scheduler").
		Logger()

	scheduler := asynq.NewScheduler(p.RedisOpt, &asynq.SchedulerOpts{
		LogLevel: asynq.DebugLevel, // Reduce noise
		Logger:   &asynqLogger{logger: &log},
	})

	cs := &CronScheduler{
		logger:     &log,
		scheduler:  scheduler,
		jobService: p.JobService,
		cfg:        p.Config.Cfg.CronScheduler,
	}

	return cs
}

// Start begins the cron scheduler
func (cs *CronScheduler) Start() error {
	log := cs.logger.With().
		Str("operation", "Start").
		Logger()

	// Check if scheduler is enabled
	if !cs.cfg.Enabled {
		log.Info().Msg("cron scheduler is disabled via configuration")
		return nil
	}

	log.Info().Msg("starting cron scheduler")

	// Schedule all recurring jobs
	if err := cs.SchedulePatternAnalysisJobs(); err != nil {
		log.Error().Err(err).Msg("failed to schedule pattern analysis jobs")
		return err
	}

	if err := cs.ScheduleShipmentJobs(); err != nil {
		log.Error().Err(err).Msg("failed to schedule shipment jobs")
		return err
	}

	if err := cs.ScheduleSystemJobs(); err != nil {
		log.Error().Err(err).Msg("failed to schedule system jobs")
		return err
	}

	if err := cs.ScheduleEmailQueueJobs(); err != nil {
		log.Error().Err(err).Msg("failed to schedule email queue jobs")
		return err
	}

	if err := cs.ScheduleComplianceJobs(); err != nil {
		log.Error().Err(err).Msg("failed to schedule compliance jobs")
		return err
	}

	// Start the scheduler
	if err := cs.scheduler.Start(); err != nil {
		log.Error().Err(err).Msg("failed to start scheduler")
		return err
	}

	cs.logger.Info().Msg("cron scheduler started successfully")
	return nil
}

// Stop shuts down the cron scheduler
func (cs *CronScheduler) Stop() error {
	cs.logger.Info().Msg("stopping cron scheduler")
	cs.scheduler.Shutdown()
	return nil
}

// ScheduleShipmentJobs schedules all shipment-related jobs
func (cs *CronScheduler) ScheduleShipmentJobs() error {
	log := cs.logger.With().
		Str("operation", "ScheduleShipmentJobs").
		Logger()

	log.Info().Msg("scheduling shipment jobs")

	// Schedule delay shipment job
	if cs.cfg.ShipmentJobs.DelayShipment.Enabled {
		if err := cs.scheduleJob(
			services.JobTypeDelayShipment,
			cs.cfg.ShipmentJobs.DelayShipment,
			cs.createDelayShipmentPayload(),
		); err != nil {
			return fmt.Errorf("failed to schedule delay shipment job: %w", err)
		}
	}

	return nil
}

// SchedulePatternAnalysisJobs schedules recurring pattern analysis jobs
func (cs *CronScheduler) SchedulePatternAnalysisJobs() error {
	log := cs.logger.With().
		Str("operation", "SchedulePatternAnalysisJobs").
		Logger()

	log.Info().Msg("scheduling pattern analysis jobs")

	// Daily pattern analysis
	if cs.cfg.PatternAnalysisJobs.DailyAnalysis.Enabled {
		if err := cs.scheduleJob(
			services.JobTypeAnalyzePatterns,
			cs.cfg.PatternAnalysisJobs.DailyAnalysis,
			cs.createGlobalPatternAnalysisPayload(),
		); err != nil {
			return fmt.Errorf("failed to schedule daily pattern analysis: %w", err)
		}
	}

	// Weekly comprehensive analysis
	if cs.cfg.PatternAnalysisJobs.WeeklyAnalysis.Enabled {
		if err := cs.scheduleJob(
			services.JobTypeAnalyzePatterns,
			cs.cfg.PatternAnalysisJobs.WeeklyAnalysis,
			cs.createGlobalPatternAnalysisPayload(),
		); err != nil {
			return fmt.Errorf("failed to schedule weekly pattern analysis: %w", err)
		}
	}

	// Expire suggestions
	if cs.cfg.PatternAnalysisJobs.ExpireSuggestions.Enabled {
		if err := cs.scheduleJob(
			services.JobTypeExpireOldSuggestions,
			cs.cfg.PatternAnalysisJobs.ExpireSuggestions,
			cs.createExpireSuggestionsPayload(),
		); err != nil {
			return fmt.Errorf("failed to schedule expire suggestions job: %w", err)
		}
	}

	return nil
}

// ScheduleSystemJobs schedules recurring system maintenance jobs
func (cs *CronScheduler) ScheduleSystemJobs() error {
	log := cs.logger.With().
		Str("operation", "ScheduleSystemJobs").
		Logger()

	log.Info().Msg("scheduling system jobs")

	// Cleanup temp files
	if cs.cfg.SystemJobs.CleanupTempFiles.Enabled {
		if err := cs.scheduleJob(
			services.JobTypeCleanupTempFiles,
			cs.cfg.SystemJobs.CleanupTempFiles,
			cs.createSystemJobPayload(),
		); err != nil {
			return fmt.Errorf("failed to schedule cleanup temp files job: %w", err)
		}
	}

	// Generate reports
	if cs.cfg.SystemJobs.GenerateReports.Enabled {
		if err := cs.scheduleJob(
			services.JobTypeGenerateReports,
			cs.cfg.SystemJobs.GenerateReports,
			cs.createSystemJobPayload(),
		); err != nil {
			return fmt.Errorf("failed to schedule generate reports job: %w", err)
		}
	}

	// Data backup
	if cs.cfg.SystemJobs.DataBackup.Enabled {
		if err := cs.scheduleJob(
			services.JobTypeDataBackup,
			cs.cfg.SystemJobs.DataBackup,
			cs.createSystemJobPayload(),
		); err != nil {
			return fmt.Errorf("failed to schedule data backup job: %w", err)
		}
	}

	return nil
}

// createGlobalPatternAnalysisPayload creates a payload for global pattern analysis
func (cs *CronScheduler) createGlobalPatternAnalysisPayload() []byte {
	payload := &services.PatternAnalysisPayload{
		JobBasePayload: services.JobBasePayload{
			JobID:     cs.generateNewJobID(),
			Timestamp: timeutils.NowUnix(),
		},
		MinFrequency:  cs.cfg.PatternAnalysisJobs.MinFrequency,
		TriggerReason: "scheduled",
	}

	data, _ := services.MarshalPayload(payload)
	return data
}

func (cs *CronScheduler) createDelayShipmentPayload() []byte {
	payload := &services.DelayShipmentPayload{
		JobBasePayload: services.JobBasePayload{
			JobID:     cs.generateNewJobID(),
			Timestamp: timeutils.NowUnix(),
		},
	}

	data, _ := services.MarshalPayload(payload)
	return data
}

// createExpireSuggestionsPayload creates a payload for expiring suggestions
func (cs *CronScheduler) createExpireSuggestionsPayload() []byte {
	payload := &services.ExpireSuggestionsPayload{
		JobBasePayload: services.JobBasePayload{
			JobID:     cs.generateNewJobID(),
			Timestamp: timeutils.NowUnix(),
		},
		BatchSize: 100,
	}

	data, _ := services.MarshalPayload(payload)
	return data
}

// ScheduleEmailQueueJobs schedules recurring email queue processing jobs
func (cs *CronScheduler) ScheduleEmailQueueJobs() error {
	log := cs.logger.With().
		Str("operation", "ScheduleEmailQueueJobs").
		Logger()

	log.Info().Msg("scheduling email queue jobs")

	// Process email queue
	if cs.cfg.EmailQueueJobs.ProcessQueue.Enabled {
		if err := cs.scheduleJob(
			services.JobTypeProcessEmailQueue,
			cs.cfg.EmailQueueJobs.ProcessQueue,
			cs.createEmailQueuePayload(),
		); err != nil {
			return fmt.Errorf("failed to schedule process email queue job: %w", err)
		}
	}

	return nil
}

// createEmailQueuePayload creates a payload for email queue processing
func (cs *CronScheduler) createEmailQueuePayload() []byte {
	payload := &services.JobBasePayload{
		JobID:     cs.generateNewJobID(),
		Timestamp: timeutils.NowUnix(),
	}

	data, _ := services.MarshalPayload(payload)
	return data
}

// createSystemJobPayload creates a payload for system jobs
func (cs *CronScheduler) createSystemJobPayload() []byte {
	payload := &services.JobBasePayload{
		JobID:     cs.generateNewJobID(),
		Timestamp: timeutils.NowUnix(),
	}

	data, _ := services.MarshalPayload(payload)
	return data
}

// createComplianceJobPayload creates a payload for compliance jobs
func (cs *CronScheduler) createComplianceJobPayload() []byte {
	payload := &services.JobBasePayload{
		JobID:     cs.generateNewJobID(),
		Timestamp: timeutils.NowUnix(),
	}

	data, _ := services.MarshalPayload(payload)
	return data
}

// ScheduleComplianceJobs schedules recurring compliance jobs
func (cs *CronScheduler) ScheduleComplianceJobs() error {
	log := cs.logger.With().
		Str("operation", "ScheduleComplianceJobs").
		Logger()

	log.Info().Msg("scheduling compliance jobs")

	// Compliance check
	if cs.cfg.ComplianceJobs.ComplianceCheck.Enabled {
		if err := cs.scheduleJob(
			services.JobTypeComplianceCheck,
			cs.cfg.ComplianceJobs.ComplianceCheck,
			cs.createComplianceJobPayload(),
		); err != nil {
			return fmt.Errorf("failed to schedule compliance check job: %w", err)
		}
	}

	// Hazmat expiration check
	if cs.cfg.ComplianceJobs.HazmatExpiration.Enabled {
		if err := cs.scheduleJob(
			services.JobTypeHazmatExpirationCheck,
			cs.cfg.ComplianceJobs.HazmatExpiration,
			cs.createComplianceJobPayload(),
		); err != nil {
			return fmt.Errorf("failed to schedule hazmat expiration job: %w", err)
		}
	}

	return nil
}

// scheduleJob is a helper method to schedule a single job with configuration
func (cs *CronScheduler) scheduleJob(
	jobType services.JobType,
	cfg config.JobConfig,
	payload []byte,
) error {
	// Build task options
	opts := []asynq.Option{
		asynq.Queue(cfg.Queue),
	}

	// Apply retention if specified
	if cfg.Retention > 0 {
		opts = append(opts, asynq.Retention(cfg.Retention))
	}

	// Apply timeout if specified
	if cfg.Timeout > 0 {
		opts = append(opts, asynq.Timeout(cfg.Timeout))
	}

	// Apply unique key if specified
	if cfg.UniqueKey > 0 {
		opts = append(opts, asynq.Unique(cfg.UniqueKey))
	}

	// Apply retry policy
	retryPolicy := cs.cfg.GlobalRetryPolicy
	if cfg.RetryPolicy != nil {
		retryPolicy = *cfg.RetryPolicy
	}

	if retryPolicy.MaxRetries > 0 {
		opts = append(opts, asynq.MaxRetry(retryPolicy.MaxRetries))
	}

	// Create task
	task := asynq.NewTask(string(jobType), payload)

	// Register with scheduler
	entryID, err := cs.scheduler.Register(cfg.Schedule, task, opts...)
	if err != nil {
		return err
	}

	cs.logger.Info().
		Str("job_type", string(jobType)).
		Str("entry_id", entryID).
		Str("schedule", cfg.Schedule).
		Str("queue", cfg.Queue).
		Dur("timeout", cfg.Timeout).
		Dur("retention", cfg.Retention).
		Int("max_retries", retryPolicy.MaxRetries).
		Msg("scheduled job")

	return nil
}

func (cs *CronScheduler) generateNewJobID() string {
	return pulid.MustNew("job_").String()
}

// asynqLogger implements asynq.Logger interface for consistent logging
type asynqLogger struct {
	logger *zerolog.Logger
}

func (l *asynqLogger) Debug(args ...any) {
	l.logger.Debug().Interface("args", args).Msg("asynq debug")
}

func (l *asynqLogger) Info(args ...any) {
	l.logger.Info().Interface("args", args).Msg("asynq info")
}

func (l *asynqLogger) Warn(args ...any) {
	l.logger.Warn().Interface("args", args).Msg("asynq warning")
}

func (l *asynqLogger) Error(args ...any) {
	l.logger.Error().Interface("args", args).Msg("asynq error")
}

func (l *asynqLogger) Fatal(args ...any) {
	l.logger.Fatal().Interface("args", args).Msg("asynq fatal")
}
