package agentruntime

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"go.uber.org/fx"
)

// Module provides the runtime once, both as itself and as the AgentRuntime
// port. The durable runtime drives turns through the concrete Service, which
// can rebuild a turn from data; everything else only ever needs the port.
var Module = fx.Module("agentruntime",
	fx.Provide(
		fx.Annotate(
			New,
			fx.As(fx.Self()),
			fx.As(new(agentports.AgentRuntime)),
			fx.As(new(agentports.AgentRecommender)),
		),
		NewContextBuilder,
		RuntimePolicies,
	),
)
