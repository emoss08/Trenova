package notificationjobs

import "go.uber.org/fx"

var Module = fx.Module("notification-jobs",
	fx.Provide(NewActivities),
	fx.Provide(NewRegistry),
)
