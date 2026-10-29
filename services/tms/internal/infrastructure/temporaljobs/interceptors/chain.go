package interceptors

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/temporalconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/platformmetrics"
	"go.temporal.io/sdk/interceptor"
	"go.uber.org/zap"
)

type ChainParams struct {
	TemporalConfig *temporalconfig.Config
	Logger         *zap.Logger
	MetricsHandler *platformmetrics.Temporal
}

func BuildWorkerInterceptorChain(p ChainParams) []interceptor.WorkerInterceptor {
	workerInterceptors := []interceptor.WorkerInterceptor{NewTenantScopeInterceptor()}

	if p.MetricsHandler != nil && p.MetricsHandler.IsEnabled() {
		metricsInterceptor := NewMetricsInterceptor(p.MetricsHandler)
		workerInterceptors = append(workerInterceptors, metricsInterceptor)
		p.Logger.Debug("metrics interceptor enabled")
	}

	cfg := p.TemporalConfig.Interceptors
	if cfg.EnableLogging {
		loggingInterceptor := NewLoggingInterceptor(p.Logger, cfg.GetLogLevel())
		workerInterceptors = append(workerInterceptors, loggingInterceptor)
		p.Logger.Debug("logging interceptor enabled",
			zap.String("logLevel", cfg.GetLogLevel()),
		)
	}

	return workerInterceptors
}
