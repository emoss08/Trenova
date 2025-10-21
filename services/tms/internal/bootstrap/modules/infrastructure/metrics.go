package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"go.uber.org/fx"
)

var MetricsModule = fx.Module("metrics",
	fx.Provide(observability.NewMetricsRegistry),
)
