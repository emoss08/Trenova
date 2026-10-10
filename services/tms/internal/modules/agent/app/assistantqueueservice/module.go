package assistantqueueservice

import "go.uber.org/fx"

var Module = fx.Module("assistantqueueservice",
	fx.Provide(New, NewSettler),
)
