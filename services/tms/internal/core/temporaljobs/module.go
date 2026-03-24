package temporaljobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/temporaljobs/health"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type WorkerManagerParams struct {
	fx.In

	Client     client.Client
	Logger     *zap.Logger
	LC         fx.Lifecycle
	Registries []registry.WorkerRegistry `group:"worker_registries"`
}

func NewWorkerManager(p WorkerManagerParams) (*registry.WorkerManager, error) {
	manager := registry.NewWorkerManager(p.Client, p.Logger)

	for _, reg := range p.Registries {
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
