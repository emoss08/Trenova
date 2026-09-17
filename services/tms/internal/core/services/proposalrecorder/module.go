package proposalrecorder

import "go.uber.org/fx"

var Module = fx.Module("proposalrecorder",
	fx.Provide(New),
)
