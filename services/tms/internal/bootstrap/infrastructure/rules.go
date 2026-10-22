package infrastructure

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/rules"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const rulesWarmupTimeout = 2 * time.Minute

type rulesWarmupParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Logger    *zap.Logger
	Metrics   *metrics.Registry
}

var RulesModule = fx.Module("rules",
	fx.Invoke(startRulesWarmup),
)

func startRulesWarmup(p rulesWarmupParams) {
	logger := p.Logger.Named("rules")
	runtime := rules.Default()
	runtime.Instrument(logger, p.Metrics.Rules)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go warmRules(ctx, runtime, logger, done)
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

func warmRules(
	ctx context.Context,
	runtime *rules.Runtime,
	logger *zap.Logger,
	done chan<- struct{},
) {
	defer close(done)
	ctx, cancel := context.WithTimeout(ctx, rulesWarmupTimeout)
	defer cancel()

	started := time.Now()
	if err := runtime.Warm(ctx); err != nil {
		logger.Error(
			"rules module failed to warm up; every formula check, rating evaluation and allocation that calls it will fail",
			zap.Error(err),
		)
		return
	}
	logger.Info("rules module ready",
		zap.Duration("took", time.Since(started)),
		zap.Int("poolSize", runtime.PoolSize()),
	)
}
