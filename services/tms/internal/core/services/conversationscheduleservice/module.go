package conversationscheduleservice

import "go.uber.org/fx"

var Module = fx.Module("conversationscheduleservice",
	fx.Provide(New, NewRunner),
)
