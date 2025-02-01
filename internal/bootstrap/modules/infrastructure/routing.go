package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/external/maps/pcmiler"
	"go.uber.org/fx"
)

var RoutingModule = fx.Module("routing", fx.Provide(pcmiler.NewClient))
