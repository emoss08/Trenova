package completionrouter

import "go.uber.org/fx"

var Module = fx.Module("completion-router",
	fx.Provide(New),
)
