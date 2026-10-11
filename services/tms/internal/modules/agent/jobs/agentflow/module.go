package agentflow

import "go.uber.org/fx"

// Module provides a turn's activities once, for every queue that runs turns.
var Module = fx.Module("agentflow",
	fx.Provide(NewActivities),
)
