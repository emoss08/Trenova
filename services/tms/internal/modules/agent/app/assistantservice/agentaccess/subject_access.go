package agentaccess

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// AssertChatAgent refuses a conversation with an agent built to run on its
// own. A desk's earned autonomy was earned on its own work; a person talking
// to it would borrow that autonomy for writes of their own choosing.
func AssertChatAgent(definition *agentdefinition.Definition) error {
	if definition.TriggerMode != "" && definition.TriggerMode != agentdefinition.TriggerChat {
		return errortypes.NewBusinessError(
			"Agent {0} runs on its own and cannot be talked to", definition.Name,
		)
	}

	return nil
}
