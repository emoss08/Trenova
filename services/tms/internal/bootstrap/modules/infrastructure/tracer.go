package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"go.uber.org/fx"
)

var TracerModule = fx.Module("tracer",
	fx.Provide(observability.NewTracerProvider),
)
