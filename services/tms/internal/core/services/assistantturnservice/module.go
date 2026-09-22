package assistantturnservice

import "go.uber.org/fx"

var Module = fx.Module("assistantturnservice",
	fx.Provide(New),
)
