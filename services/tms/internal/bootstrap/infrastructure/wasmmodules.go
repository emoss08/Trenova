package infrastructure

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const wasmModuleWarmupTimeout = 2 * time.Minute

type wasmModulesParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Logger    *zap.Logger
	Metrics   *metrics.Registry
}

type warmedModule struct {
	runtime *wasmmodule.Runtime
	failure string
}

var WasmModulesModule = fx.Module("wasmmodules",
	fx.Invoke(startWasmModuleWarmup),
)

func startWasmModuleWarmup(p wasmModulesParams) {
	rulesRuntime := rules.Default().Runtime
	rulesRuntime.Instrument(p.Logger.Named("rules"), p.Metrics.Rules)
	modules := []warmedModule{
		{
			runtime: rulesRuntime,
			failure: "rules module failed to warm up; every formula check, rating evaluation and allocation that calls it will fail",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go warmModules(ctx, modules, p.Logger, done)
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			cancel()
			select {
			case <-done:
			case <-stopCtx.Done():
			}
			return nil
		},
	})
}

func warmModules(
	ctx context.Context,
	modules []warmedModule,
	logger *zap.Logger,
	done chan<- struct{},
) {
	defer close(done)
	ctx, cancel := context.WithTimeout(ctx, wasmModuleWarmupTimeout)
	defer cancel()

	for _, module := range modules {
		named := logger.Named(module.runtime.Name())
		started := time.Now()
		if err := module.runtime.Warm(ctx); err != nil {
			named.Error(module.failure, zap.Error(err))
			continue
		}
		named.Info("module ready",
			zap.Duration("took", time.Since(started)),
			zap.Int("poolSize", module.runtime.PoolSize()),
		)
	}
}
