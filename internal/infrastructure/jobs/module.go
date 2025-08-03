/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package jobs

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/jobs"
	"github.com/emoss08/trenova/internal/pkg/jobs/handlers"
	"github.com/emoss08/trenova/internal/pkg/jobs/scheduler"
	"github.com/emoss08/trenova/internal/pkg/jobs/triggers"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// Module provides job service infrastructure and handlers
var Module = fx.Module(
	"jobs",
	fx.Provide(
		// Infrastructure
		NewRedisConnOpt,
		jobs.NewJobService,
		scheduler.NewCronScheduler,
		triggers.NewShipmentTrigger,

		// Job Handlers
		fx.Annotate(
			handlers.NewPatternAnalysisHandler,
			fx.As(new(jobs.JobHandler)),
			fx.ResultTags(`group:"job_handlers"`),
		),
		fx.Annotate(
			handlers.NewExpireSuggestionsHandler,
			fx.As(new(jobs.JobHandler)),
			fx.ResultTags(`group:"job_handlers"`),
		),
		fx.Annotate(
			handlers.NewDuplicateShipmentHandler,
			fx.As(new(jobs.JobHandler)),
			fx.ResultTags(`group:"job_handlers"`),
		),
		fx.Annotate(
			handlers.NewDelayShipmentHandler,
			fx.As(new(jobs.JobHandler)),
			fx.ResultTags(`group:"job_handlers"`),
		),
		fx.Annotate(
			handlers.NewEmailHandler,
			fx.As(new(jobs.JobHandler)),
			fx.ResultTags(`group:"job_handlers"`),
		),
		fx.Annotate(
			handlers.NewEmailQueueHandler,
			fx.As(new(jobs.JobHandler)),
			fx.ResultTags(`group:"job_handlers"`),
		),
	),
	fx.Invoke(
		RegisterLifecycleHooks,
	),
)

// NewRedisConnOpt creates Redis connection options for Asynq
func NewRedisConnOpt(cfg *config.Manager) asynq.RedisClientOpt {
	redisConfig := cfg.Redis()

	return asynq.RedisClientOpt{
		Addr:     redisConfig.Addr,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	}
}

// LifecycleParams defines dependencies for lifecycle management
type LifecycleParams struct {
	fx.In

	Lifecycle     fx.Lifecycle
	JobService    jobs.JobServiceInterface
	CronScheduler scheduler.CronSchedulerInterface
}

// RegisterLifecycleHooks registers startup and shutdown hooks for the job service and scheduler
func RegisterLifecycleHooks(p LifecycleParams) {
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			// Start job service first
			if err := p.JobService.Start(); err != nil {
				log.Error().Err(err).Msg("failed to start job service")
				return err
			}

			// Then start cron scheduler
			if err := p.CronScheduler.Start(); err != nil {
				log.Error().Err(err).Msg("failed to start cron scheduler")
				return err
			}

			log.Info().Msg("job service and cron scheduler started")

			return nil
		},
		OnStop: func(context.Context) error {
			// Stop scheduler first
			if err := p.CronScheduler.Stop(); err != nil {
				log.Error().Err(err).Msg("failed to stop cron scheduler")
				return err
			}

			// Then stop job service
			if err := p.JobService.Shutdown(); err != nil {
				log.Error().Err(err).Msg("failed to stop job service")
				return err
			}

			log.Info().Msg("job service and cron scheduler stopped")

			return nil
		},
	})
}
