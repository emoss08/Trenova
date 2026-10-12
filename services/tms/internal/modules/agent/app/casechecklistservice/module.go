package casechecklistservice

import "go.uber.org/fx"

var Module = fx.Module("casechecklistservice",
	fx.Provide(New),
)
