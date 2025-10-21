package infrastructure

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"go.uber.org/fx"
)

var TemporalClientModule = fx.Module("temporal-infrastructure",
	temporaljobs.Module,
)
