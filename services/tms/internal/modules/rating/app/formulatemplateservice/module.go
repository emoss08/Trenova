package formulatemplateservice

import "go.uber.org/fx"

var Module = fx.Module("formulatemplateservice",
	fx.Provide(New),
)
