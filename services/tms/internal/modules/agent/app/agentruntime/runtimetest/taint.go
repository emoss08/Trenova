package runtimetest

import (
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
)

func MoneyTool() *agenttest.StubActionTool {
	amount := int64(10_000)

	return &agenttest.StubActionTool{
		ToolName:   "apply_payment",
		Tier:       agent.TierAutoExecute,
		Egress:     agent.EgressMoney,
		MoneyMinor: &amount,
	}
}

func AutoDefinition(tools ...string) *agentdefinition.Definition {
	definition := Definition(tools...)
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = make(map[string]agent.AutonomyTier, len(tools))
	for _, tool := range tools {
		definition.ToolTiers[tool] = agent.TierAutoExecute
	}

	return definition
}
