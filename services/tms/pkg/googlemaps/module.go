package googlemaps

import "go.uber.org/fx"

var Module = fx.Module("googlemaps",
	fx.Provide(
		NewClient,
		NewAutoCompleteService,
	),
)
