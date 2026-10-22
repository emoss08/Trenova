package formulalibraryservice

import "go.uber.org/fx"

var Module = fx.Module("formulalibraryservice",
	fx.Provide(New),
)
