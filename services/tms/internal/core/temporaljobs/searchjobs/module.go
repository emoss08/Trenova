package searchjobs

import "go.uber.org/fx"

var Module = fx.Module("search-jobs",
	fx.Provide(NewActivities),
	fx.Provide(NewRegistry),
)
