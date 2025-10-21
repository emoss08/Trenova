package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"go.uber.org/fx"
)

var ObservabilityModule = fx.Module("observability",
	TracerModule,
	MetricsModule,
	fx.Provide(observability.NewMiddleware),
)
