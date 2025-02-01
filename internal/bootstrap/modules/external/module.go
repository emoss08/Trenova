package external

import (
	"github.com/emoss08/trenova/internal/infrastructure/external/maps/pcmiler"
	"go.uber.org/fx"
)

var Module = fx.Module("external", fx.Provide(
	pcmiler.NewClient,
))
