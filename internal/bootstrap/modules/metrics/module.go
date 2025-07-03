package metrics

import (
	"github.com/emoss08/trenova/internal/api/handlers"
	"go.uber.org/fx"
)

// Module provides metrics functionality
var Module = fx.Module(
	"metrics",
	fx.Provide(
		handlers.NewMetricsHandler,
	),
)