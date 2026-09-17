package agentruntime

import "go.uber.org/fx"

var Module = fx.Module("agentruntime",
	fx.Provide(New, NewContextBuilder),
)
