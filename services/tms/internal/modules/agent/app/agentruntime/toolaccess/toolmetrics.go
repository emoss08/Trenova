package toolaccess

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
)

// UnregisteredToolLabel is the tool a call is counted under when it names
// nothing the runtime knows. The name comes from the model, and a label taken
// from it as sent would let one confused model mint a series per invented tool.
const UnregisteredToolLabel = "unregistered"

func (s *Access) ToolMetricLabel(name string) string {
	if s.ToolNamed(name) != nil {
		return name
	}
	if _, runtime := runtimetools.PolicyNamed(name); runtime {
		return name
	}

	return UnregisteredToolLabel
}
