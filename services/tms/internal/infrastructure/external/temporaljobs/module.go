/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package temporaljobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/domains/shipment"
	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/scheduler"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// Module provides Temporal infrastructure and worker management
var Module = fx.Module(
	"temporal",
	fx.Provide(
		NewClient,
		NewWorker,
		NewRegistry,
		NewService,
		scheduler.NewCronScheduler,
		// Provide activity providers
		shipment.NewActivityProvider,
		// Provide the adapter as the JobService implementation
		fx.Annotate(
			NewTemporalJobServiceAdapter,
			fx.As(new(services.JobService)),
		),
	),
	fx.Invoke(
		RegisterLifecycleHooks,
		RegisterSchedulerLifecycle,
		RegisterAllWorkflowsAndActivities,
		RegisterActivityProviders,
	),
)

// LifecycleParams defines dependencies for lifecycle management
type LifecycleParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Worker    *Worker
}

// RegisterLifecycleHooks registers startup and shutdown hooks for the Temporal worker
func RegisterLifecycleHooks(p LifecycleParams) {
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			// Start worker in a non-blocking way
			if err := p.Worker.Start(); err != nil {
				log.Error().Err(err).Msg("failed to start temporal worker")
				return err
			}

			log.Info().Msg("temporal worker started")
			return nil
		},
		OnStop: func(context.Context) error {
			// Stop the worker gracefully
			if err := p.Worker.Stop(); err != nil {
				log.Error().Err(err).Msg("failed to stop temporal worker")
				return err
			}

			log.Info().Msg("temporal worker stopped")
			return nil
		},
	})
}

// SchedulerLifecycleParams defines dependencies for scheduler lifecycle
type SchedulerLifecycleParams struct {
	fx.In

	Lifecycle     fx.Lifecycle
	CronScheduler *scheduler.CronScheduler
}

// RegisterSchedulerLifecycle registers startup and shutdown hooks for the scheduler
func RegisterSchedulerLifecycle(p SchedulerLifecycleParams) {
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			// Start scheduler
			if err := p.CronScheduler.Start(); err != nil {
				log.Error().Err(err).Msg("failed to start temporal scheduler")
				return err
			}

			log.Info().Msg("temporal scheduler started")
			return nil
		},
		OnStop: func(context.Context) error {
			// Stop the scheduler
			if err := p.CronScheduler.Stop(); err != nil {
				log.Error().Err(err).Msg("failed to stop temporal scheduler")
				return err
			}

			log.Info().Msg("temporal scheduler stopped")
			return nil
		},
	})
}

// ActivityProviderParams defines dependencies for activity providers
type ActivityProviderParams struct {
	fx.In

	Worker              *Worker
	ShipmentActivityProvider *shipment.ActivityProvider
}

// RegisterActivityProviders registers activity providers with the worker
func RegisterActivityProviders(p ActivityProviderParams) {
	// We need to register the activity provider on the specific task queue worker
	// Since workers are created during Start(), we'll store the provider for later registration
	p.Worker.activityProviders = append(p.Worker.activityProviders, p.ShipmentActivityProvider)
	
	log.Info().Msg("stored activity providers for temporal worker registration")
}

