package reportjobs

import "go.uber.org/fx"

var Module = fx.Module("report-jobs",
	fx.Provide(NewActivities),
	fx.Provide(NewRegistry),
)
