package reflectionjobs

import "go.uber.org/fx"

var Module = fx.Module("agent-reflection-jobs",
	fx.Provide(NewActivities, NewScheduler),
)
