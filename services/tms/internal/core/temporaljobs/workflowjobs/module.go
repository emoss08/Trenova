package workflowjobs

import "go.uber.org/fx"

// Module provides the workflow jobs module for dependency injection
var Module = fx.Module("workflowjobs",
	fx.Provide(
		NewRegistry,
		NewActivities,
		NewActionHandlers,
	),
)
