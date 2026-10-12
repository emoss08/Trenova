package agenttoolstest

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

func CallLimit(call agentports.CallPolicy) agent.AutonomyTier {
	limit := call.Egress.Ceiling()
	if call.MaxTier.IsValid() {
		limit = limit.AtMost(call.MaxTier)
	}

	return limit
}
