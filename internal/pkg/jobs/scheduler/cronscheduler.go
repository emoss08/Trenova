package scheduler

import (
	"time"

	"github.com/emoss08/trenova/internal/pkg/jobs"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// CronSchedulerParams defines dependencies for the cron scheduler
type CronSchedulerParams struct {
	fx.In

	Logger     *logger.Logger
	JobService jobs.JobServiceInterface
	RedisOpt   asynq.RedisClientOpt
}

// CronScheduler manages scheduled recurring jobs
type CronScheduler struct {
	logger     *zerolog.Logger
	scheduler  *asynq.Scheduler
	jobService jobs.JobServiceInterface
}

// CronSchedulerInterface defines methods for managing scheduled jobs
type CronSchedulerInterface interface {
	Start() error
	Stop() error
	SchedulePatternAnalysisJobs() error
	ScheduleMaintenanceJobs() error
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
		LogLevel: asynq.WarnLevel, // Reduce noise
		Logger:   &asynqLogger{logger: &log},
	})

	cs := &CronScheduler{
		logger:     &log,
		scheduler:  scheduler,
		jobService: p.JobService,
	}

	return cs
}

// Start begins the cron scheduler
func (cs *CronScheduler) Start() error {
	cs.logger.Info().Msg("starting cron scheduler")

	// Schedule all recurring jobs
	if err := cs.SchedulePatternAnalysisJobs(); err != nil {
		cs.logger.Error().Err(err).Msg("failed to schedule pattern analysis jobs")
		return err
	}

	if err := cs.ScheduleDelayShipmentJobs(); err != nil {
		cs.logger.Error().Err(err).Msg("failed to schedule delay shipment jobs")
		return err
	}

	if err := cs.ScheduleMaintenanceJobs(); err != nil {
		cs.logger.Error().Err(err).Msg("failed to schedule maintenance jobs")
		return err
	}

	if err := cs.ScheduleEmailQueueJobs(); err != nil {
		cs.logger.Error().Err(err).Msg("failed to schedule email queue jobs")
		return err
	}

	// Start the scheduler
	if err := cs.scheduler.Start(); err != nil {
		cs.logger.Error().Err(err).Msg("failed to start scheduler")
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

func (cs *CronScheduler) ScheduleDelayShipmentJobs() error {
	cs.logger.Info().Msg("scheduling delay shipment jobs")

	// Delay shipments every 3 minutes
	delayTask := asynq.NewTask(
		string(jobs.JobTypeDelayShipment),
		cs.createDelayShipmentPayload(),
	)

	entryID, err := cs.scheduler.Register("@every 1m", delayTask,
		asynq.Queue(jobs.QueueShipment),
		asynq.MaxRetry(2),
		asynq.Retention(24*time.Hour),
		asynq.Timeout(1*time.Minute), // * Timeout of 1 minute
	)
	if err != nil {
		return err
	}

	cs.logger.Info().
		Str("entry_id", entryID).
		Str("schedule", "@every 3m (every 3 minutes)").
		Msg("scheduled delay shipment job")

	return nil
}

// SchedulePatternAnalysisJobs schedules recurring pattern analysis jobs
func (cs *CronScheduler) SchedulePatternAnalysisJobs() error {
	cs.logger.Info().Msg("scheduling pattern analysis jobs")

	// Daily pattern analysis at 2 AM
	dailyPatternTask := asynq.NewTask(
		string(jobs.JobTypeAnalyzePatterns),
		cs.createGlobalPatternAnalysisPayload(),
		asynq.Retention(24*time.Hour),
	)

	entryID, err := cs.scheduler.Register("@every 24h", dailyPatternTask,
		asynq.Queue(jobs.QueuePattern),
	)
	if err != nil {
		return err
	}

	cs.logger.Info().
		Str("entry_id", entryID).
		Str("schedule", "@every 24h (daily at 2 AM)").
		Msg("scheduled daily pattern analysis")

	// Weekly comprehensive analysis on Sundays at 1 AM
	weeklyPatternTask := asynq.NewTask(
		string(jobs.JobTypeAnalyzePatterns),
		cs.createGlobalPatternAnalysisPayload(),
	)

	entryID, err = cs.scheduler.Register("0 1 * * 0", weeklyPatternTask,
		asynq.Queue(jobs.QueuePattern),
		asynq.MaxRetry(3),
	)
	if err != nil {
		return err
	}

	cs.logger.Info().
		Str("entry_id", entryID).
		Str("schedule", "0 1 * * 0 (weekly on Sunday at 1 AM)").
		Msg("scheduled weekly comprehensive pattern analysis")

	return nil
}

// ScheduleMaintenanceJobs schedules recurring maintenance jobs
func (cs *CronScheduler) ScheduleMaintenanceJobs() error {
	cs.logger.Info().Msg("scheduling maintenance jobs")

	// Expire old suggestions every 6 hours
	expireTask := asynq.NewTask(
		string(jobs.JobTypeExpireOldSuggestions),
		cs.createExpireSuggestionsPayload(),
	)

	entryID, err := cs.scheduler.Register("0 */6 * * *", expireTask,
		asynq.Queue(jobs.QueueSystem),
		asynq.MaxRetry(2),
	)
	if err != nil {
		return err
	}

	cs.logger.Info().
		Str("entry_id", entryID).
		Str("schedule", "0 */6 * * * (every 6 hours)").
		Msg("scheduled suggestion expiration job")

	return nil
}

// createGlobalPatternAnalysisPayload creates a payload for global pattern analysis
func (cs *CronScheduler) createGlobalPatternAnalysisPayload() []byte {
	payload := &jobs.PatternAnalysisPayload{
		BasePayload: jobs.BasePayload{
			JobID:     pulid.MustNew("job_").String(),
			Timestamp: timeutils.NowUnix(),
		},
		MinFrequency:  3, // Conservative frequency for scheduled analysis
		TriggerReason: "scheduled",
	}

	data, _ := jobs.MarshalPayload(payload)
	return data
}

func (cs *CronScheduler) createDelayShipmentPayload() []byte {
	payload := &jobs.DelayShipmentPayload{
		BasePayload: jobs.BasePayload{
			JobID:     pulid.MustNew("job_").String(),
			Timestamp: timeutils.NowUnix(),
		},
	}

	data, _ := jobs.MarshalPayload(payload)
	return data
}

// createExpireSuggestionsPayload creates a payload for expiring suggestions
func (cs *CronScheduler) createExpireSuggestionsPayload() []byte {
	payload := &jobs.ExpireSuggestionsPayload{
		BasePayload: jobs.BasePayload{
			JobID:     pulid.MustNew("job_").String(),
			Timestamp: timeutils.NowUnix(),
		},
		BatchSize: 100,
	}

	data, _ := jobs.MarshalPayload(payload)
	return data
}

// ScheduleEmailQueueJobs schedules recurring email queue processing jobs
func (cs *CronScheduler) ScheduleEmailQueueJobs() error {
	cs.logger.Info().Msg("scheduling email queue processing jobs")

	// Process email queue every 5 minutes
	emailQueueTask := asynq.NewTask(
		string(jobs.JobTypeProcessEmailQueue),
		cs.createEmailQueuePayload(),
		asynq.Retention(12*time.Hour),
	)

	entryID, err := cs.scheduler.Register("@every 5m", emailQueueTask,
		asynq.Queue(jobs.QueueEmail),
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
	)
	if err != nil {
		return err
	}

	cs.logger.Info().
		Str("entry_id", entryID).
		Str("schedule", "@every 5m (every 5 minutes)").
		Msg("scheduled email queue processing job")

	return nil
}

// createEmailQueuePayload creates a payload for email queue processing
func (cs *CronScheduler) createEmailQueuePayload() []byte {
	payload := &jobs.BasePayload{
		JobID:     pulid.MustNew("job_").String(),
		Timestamp: timeutils.NowUnix(),
	}

	data, _ := jobs.MarshalPayload(payload)
	return data
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
