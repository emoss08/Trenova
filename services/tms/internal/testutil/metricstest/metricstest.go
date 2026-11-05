package metricstest

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"go.uber.org/zap"
)

func Disabled() *metrics.Registry {
	registry, err := metrics.NewRegistry(&monitoringconfig.Config{}, zap.NewNop())
	if err != nil {
		panic(err)
	}

	return registry
}
