package agentquerytoolservice

import "go.uber.org/fx"

var Module = fx.Module("agent-query-tool-service",
	fx.Provide(
		fx.Annotate(newGetShipmentTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newSearchShipmentsTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newGetWorkerTool, fx.ResultTags(`group:"agent_query_tools"`)),
		fx.Annotate(newSearchWorkersTool, fx.ResultTags(`group:"agent_query_tools"`)),
		NewRegistry,
	),
)
