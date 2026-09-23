package agentruneventservice

import "go.uber.org/fx"

// Module is registered in bootstrap.Options rather than the API service list,
// because the runs that most need recording are the background ones and those
// execute in the worker.
var Module = fx.Module("agentrunevent",
	fx.Provide(New),
)
