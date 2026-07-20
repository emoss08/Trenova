package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/foony"
	"go.uber.org/fx"
)

var FoonyClientModule = fx.Module("foony-client", fx.Provide(foony.New))
