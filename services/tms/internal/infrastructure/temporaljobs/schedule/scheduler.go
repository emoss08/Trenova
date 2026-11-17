package schedule

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/emoss08/trenova/internal/infrastructure/config/temporalconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/sourcegraph/conc/panics"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	featureCheckInterval = time.Minute
	retryInitialInterval = time.Second
	retryMaxInterval     = time.Minute
	retryReportEvery     = time.Minute
)

type SchedulerParams struct {
	fx.In

	Client         client.Client
	TemporalConfig *temporalconfig.Config
	Features       platformcatalogports.FeatureGate
	Logger         *zap.Logger
	LC             fx.Lifecycle
	Providers      []Provider `group:"schedule_providers"`
}

type Scheduler struct {
	reconciler *Reconciler
	registry   *Registry
	config     *temporalconfig.Config
	logger     *zap.Logger
	interval   time.Duration
	cancel     context.CancelFunc
	done       chan struct{}
}

func NewScheduler(p SchedulerParams) *Scheduler {
	registry := NewRegistry(p.Logger)

	for _, provider := range p.Providers {
		registry.RegisterProvider(provider)
	}

	reconciler := NewReconciler(p.Client, registry, p.Features, p.Logger)

	s := &Scheduler{
		reconciler: reconciler,
		registry:   registry,
		config:     p.TemporalConfig,
		logger:     p.Logger.Named("scheduler"),
		interval:   featureCheckInterval,
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

func (s *Scheduler) Start(_ context.Context) error {
	if s.registry.ProviderCount() == 0 {
		s.logger.Info("no schedule providers registered, skipping reconciliation")
		return nil
	}

	if err := s.registry.CollectSchedules(); err != nil {
		return fmt.Errorf("collect the declared schedules: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})

	go s.run(ctx)

	return nil
}

func (s *Scheduler) run(ctx context.Context) {
	defer close(s.done)

	applied, ok := s.reconcileUntilDone(ctx)
	if !ok {
		return
	}

	s.runFeatureChecks(ctx, applied)
}

func (s *Scheduler) reconcileUntilDone(ctx context.Context) ([]platformcatalog.FeatureKey, bool) {
	s.logger.Info("starting schedule reconciliation",
		zap.Int("providers", s.registry.ProviderCount()),
	)

	started := time.Now()
	attempts := 0
	failing := rate.Sometimes{Interval: retryReportEvery}

	result, err := backoff.Retry(
		ctx,
		func() (*ReconcileResult, error) {
			attempts++
			return s.reconcileOnce(ctx)
		},
		backoff.WithBackOff(s.retryBackOff()),
		backoff.WithMaxElapsedTime(0),
		backoff.WithNotify(func(err error, next time.Duration) {
			failing.Do(func() {
				s.logger.Warn("schedules are not reconciled yet; retrying until Temporal answers",
					zap.Int("attempts", attempts),
					zap.Duration("retryIn", next.Round(time.Second)),
					zap.Error(err),
				)
			})
		}),
	)
	if err != nil {
		return nil, false
	}

	s.logger.Info("schedule reconciliation complete",
		zap.String("summary", result.Summary()),
		zap.Strings("withheldFeatures", featureStrings(result.Withheld)),
		zap.Int("attempts", attempts),
		zap.Duration("took", time.Since(started).Round(100*time.Millisecond)),
	)

	return result.Withheld, true
}

func (s *Scheduler) reconcileOnce(ctx context.Context) (*ReconcileResult, error) {
	var (
		result *ReconcileResult
		err    error
	)
	recovered := panics.Try(func() { result, err = s.reconciler.Reconcile(ctx) })
	if recovered != nil {
		return nil, fmt.Errorf("schedule reconciliation panicked: %w", recovered.AsError())
	}
	if err != nil {
		return nil, err
	}
	if result.HasErrors() {
		return nil, errors.Join(result.Errors...)
	}
	return result, nil
}

func (s *Scheduler) retryBackOff() backoff.BackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = retryInitialInterval
	b.MaxInterval = retryMaxInterval
	b.Multiplier = 2
	b.RandomizationFactor = 0.2
	return b
}

func (s *Scheduler) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
		select {
		case <-s.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if s.config.Schedule.PersistOnStop {
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

func (s *Scheduler) runFeatureChecks(ctx context.Context, applied []platformcatalog.FeatureKey) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		recovered := panics.Try(func() {
			applied = s.checkFeatures(ctx, applied)
		})
		if recovered != nil {
			s.logger.Error("the plan's feature check panicked", zap.Error(recovered.AsError()))
		}
	}
}

func (s *Scheduler) checkFeatures(
	ctx context.Context,
	applied []platformcatalog.FeatureKey,
) []platformcatalog.FeatureKey {
	withheld, err := s.reconciler.Withheld(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.logger.Warn("could not read the plan's features", zap.Error(err))
		}
		return applied
	}

	if slices.Equal(withheld, applied) {
		return applied
	}

	s.logger.Info("the plan's features changed, reconciling schedules",
		zap.Strings("withheldBefore", featureStrings(applied)),
		zap.Strings("withheldNow", featureStrings(withheld)),
	)

	result, err := s.reconciler.Reconcile(ctx)
	if err != nil {
		s.logger.Error("schedule reconciliation after a plan change failed", zap.Error(err))
		return applied
	}
	if result.HasErrors() {
		s.logger.Error("schedule reconciliation after a plan change had errors",
			zap.String("summary", result.Summary()),
			zap.Error(errors.Join(result.Errors...)),
		)
		return applied
	}

	s.logger.Info("schedule reconciliation after a plan change complete",
		zap.String("summary", result.Summary()),
	)

	return result.Withheld
}
