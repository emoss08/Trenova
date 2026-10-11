package schedule

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SchedulerParams struct {
	fx.In

	Client    client.Client
	Config    *config.Config
	Logger    *zap.Logger
	LC        fx.Lifecycle
	Providers []Provider `group:"schedule_providers"`
}

type Scheduler struct {
	reconciler *Reconciler
	registry   *Registry
	config     *config.Config
	logger     *zap.Logger
}

func NewScheduler(p SchedulerParams) *Scheduler {
	registry := NewRegistry(p.Logger)

	for _, provider := range p.Providers {
		registry.RegisterProvider(provider)
	}

	reconciler := NewReconciler(p.Client, registry, p.Logger)

	s := &Scheduler{
		reconciler: reconciler,
		registry:   registry,
		config:     p.Config,
		logger:     p.Logger.Named("scheduler"),
	}

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return s.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return s.Stop(ctx)
		},
	})

	return s
}

func (s *Scheduler) Start(ctx context.Context) error {
	if s.registry.ProviderCount() == 0 {
		s.logger.Info("no schedule providers registered, skipping reconciliation")
		return nil
	}

	s.logger.Info("starting schedule reconciliation",
		zap.Int("providers", s.registry.ProviderCount()),
	)

	result, err := s.reconciler.ReconcileWithRetry(ctx, 3)
	if err != nil {
		s.logger.Error("schedule reconciliation failed", zap.Error(err))
		return err
	}

	s.logger.Info("schedule reconciliation complete",
		zap.String("summary", result.Summary()),
	)

	return nil
}

func (s *Scheduler) Stop(_ context.Context) error {
	if s.config.Temporal.Schedule.PersistOnStop {
		s.logger.Info("schedules will persist (persistOnStop=true)")
		return nil
	}

	s.logger.Info("scheduler stopped (schedules persist in Temporal by default)")
	return nil
}

func (s *Scheduler) GetRegistry() *Registry {
	return s.registry
}

func (s *Scheduler) GetReconciler() *Reconciler {
	return s.reconciler
}

func (s *Scheduler) ForceReconcile(ctx context.Context) (*ReconcileResult, error) {
	return s.reconciler.Reconcile(ctx)
}
