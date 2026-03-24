package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"go.uber.org/fx"
)

func asTracer(tp *observability.TracerProvider) observability.Tracer {
	return tp
}

var ObservabilityModule = fx.Module("observability",
	fx.Provide(
		observability.NewTracerProvider,
		asTracer,
		metrics.NewRegistry,
		observability.NewMiddleware,
	),
)
