package temporaljobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/temporaljobs/health"
	"github.com/emoss08/trenova/internal/core/temporaljobs/interceptors"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type WorkerManagerParams struct {
	fx.In

	Client      client.Client
	Logger      *zap.Logger
	LC          fx.Lifecycle
	Config      *config.Config
	Metrics     *metrics.Registry         `optional:"true"`
	Registries  []registry.WorkerRegistry `group:"worker_registries"`
	QueueFilter *registry.QueueFilter     `optional:"true"`
}

func NewWorkerManager(p WorkerManagerParams) (*registry.WorkerManager, error) {
	manager := registry.NewWorkerManager(p.Client, p.Logger)

	var temporalMetrics *metrics.Temporal
	if p.Metrics != nil {
		temporalMetrics = p.Metrics.Temporal
	}
	manager.SetInterceptors(interceptors.BuildWorkerInterceptorChain(interceptors.ChainParams{
		Config:         p.Config,
		Logger:         p.Logger,
		MetricsHandler: temporalMetrics,
	}))

	queueFilter := p.QueueFilter
	if queueFilter == nil && len(p.Config.Temporal.Worker.Queues) > 0 {
		queueFilter = &registry.QueueFilter{Queues: p.Config.Temporal.Worker.Queues}
	}

	for _, reg := range p.Registries {
		if !queueFilter.Allows(reg.GetTaskQueue()) {
			p.Logger.Info("skipping worker registry: task queue not in allowlist",
				zap.String("name", reg.GetName()),
				zap.String("taskQueue", reg.GetTaskQueue()),
			)
			continue
		}
		if err := manager.Register(reg); err != nil {
			return nil, fmt.Errorf("register worker %s: %w", reg.GetName(), err)
		}
	}

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return manager.StartAll(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return manager.StopAll(ctx)
		},
	})

	return manager, nil
}

func NewHealthChecker(manager *registry.WorkerManager, logger *zap.Logger) *health.Checker {
	return health.NewChecker(manager, logger)
}

var Module = fx.Module("temporaljobs",
	fx.Provide(NewTemporalClient),
	fx.Provide(NewWorkerManager),
	fx.Provide(NewHealthChecker),
)

var WorkerModule = fx.Module("temporal-workers",
	fx.Invoke(func(_ *registry.WorkerManager, _ *schedule.Scheduler) {}),
)
