package interceptors

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"go.temporal.io/sdk/interceptor"
	"go.uber.org/zap"
)

type ChainParams struct {
	Config         *config.Config
	Logger         *zap.Logger
	MetricsHandler *metrics.Temporal
}

func BuildWorkerInterceptorChain(p ChainParams) []interceptor.WorkerInterceptor {
	var workerInterceptors []interceptor.WorkerInterceptor

	if p.MetricsHandler != nil && p.MetricsHandler.IsEnabled() {
		metricsInterceptor := NewMetricsInterceptor(p.MetricsHandler)
		workerInterceptors = append(workerInterceptors, metricsInterceptor)
		p.Logger.Debug("metrics interceptor enabled")
	}

	cfg := p.Config.Temporal.Interceptors
	if cfg.EnableLogging {
		loggingInterceptor := NewLoggingInterceptor(p.Logger, cfg.GetLogLevel())
		workerInterceptors = append(workerInterceptors, loggingInterceptor)
		p.Logger.Debug("logging interceptor enabled",
			zap.String("logLevel", cfg.GetLogLevel()),
		)
	}

	return workerInterceptors
}
