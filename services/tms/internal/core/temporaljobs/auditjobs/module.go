package auditjobs

import "go.uber.org/fx"

var Module = fx.Module("audit-jobs",
	fx.Provide(NewActivities),
	fx.Provide(NewRegistry),
)
