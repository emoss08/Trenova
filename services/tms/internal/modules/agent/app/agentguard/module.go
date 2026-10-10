package agentguard

import "go.uber.org/fx"

var Module = fx.Module("agent-guard",
	fx.Provide(New),
)
