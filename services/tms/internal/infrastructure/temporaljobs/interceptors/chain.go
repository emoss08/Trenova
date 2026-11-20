package interceptors

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/temporalconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/platformmetrics"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"go.temporal.io/sdk/interceptor"
	"go.uber.org/zap"
)

type ChainParams struct {
	TemporalConfig *temporalconfig.Config
	Logger         *zap.Logger
	MetricsHandler *platformmetrics.Temporal
	Gate           platformcatalogports.FeatureGate
	Refusals       RefusalRecorder
}

func BuildWorkerInterceptorChain(p ChainParams) []interceptor.WorkerInterceptor {
	workerInterceptors := []interceptor.WorkerInterceptor{
		NewTenantScopeInterceptor(),
		NewEntitlementInterceptor(p.Gate, p.Refusals, p.Logger),
	}

	if p.MetricsHandler != nil && p.MetricsHandler.IsEnabled() {
		metricsInterceptor := NewMetricsInterceptor(p.MetricsHandler)
		workerInterceptors = append(workerInterceptors, metricsInterceptor)
		p.Logger.Debug("metrics interceptor enabled")
	}

	// Failures are always reported; enableLogging adds the routine lines.
	cfg := p.TemporalConfig.Interceptors
	if cfg.EnableLogging {
		workerInterceptors = append(workerInterceptors,
			NewLoggingInterceptor(p.Logger, cfg.GetLogLevel()))
		p.Logger.Debug("logging interceptor enabled",
			zap.String("logLevel", cfg.GetLogLevel()),
		)
	} else {
		workerInterceptors = append(workerInterceptors, NewFailureLoggingInterceptor(p.Logger))
	}

	// Innermost, so the interceptors above it see the error as Temporal will.
	return append(workerInterceptors, NewRefusalInterceptor())
}
