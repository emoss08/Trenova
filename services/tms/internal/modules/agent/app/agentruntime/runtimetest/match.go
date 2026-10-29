package runtimetest

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
)

func AgentHolding(name string, tools ...string) *agentdefinition.Definition {
	definition := ChatAgent(name)
	definition.ToolNames = tools

	return definition
}
