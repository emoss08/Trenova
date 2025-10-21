package ailogjobs

import "go.uber.org/fx"

var Module = fx.Module("ailogjobs",
	fx.Provide(NewActivities),
	fx.Provide(NewRegistry),
)
