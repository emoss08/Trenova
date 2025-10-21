package routing

import (
	"github.com/emoss08/trenova/pkg/googlemaps"
	"go.uber.org/fx"
)

var Module = fx.Module("routing",
	fx.Options(
		googlemaps.Module,
	),
)
