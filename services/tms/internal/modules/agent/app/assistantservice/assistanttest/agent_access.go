package assistanttest

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
)

func RestrictedChatDefinition(name string) *agentdefinition.Definition {
	definition := ChatDefinition(name, "")
	definition.AccessMode = agentdefinition.AccessRoles

	return definition
}
