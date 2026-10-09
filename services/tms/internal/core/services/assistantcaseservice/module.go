package assistantcaseservice

import "go.uber.org/fx"

var Module = fx.Module("assistantcaseservice",
	fx.Provide(NewStates, ProvideStates, New),
)
