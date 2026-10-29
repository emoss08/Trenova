package agentaccess

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func errAgentWithheld(definition *agentdefinition.Definition) error {
	return errortypes.NewAuthorizationError(
		"You do not have access to {0}. An administrator can give one of your roles access to it.",
		definition.Name,
	)
}

func ErrAgentsWithheld() error {
	return errortypes.NewAuthorizationError(
		"You don't have permission to perform this action: {0} {1}",
		permission.ResourceAssistant, permission.OpCreate,
	)
}

// ThreadReader is the person a conversation is read for. Conversations are
// read under the reader's own user id, so the reader is its owner.
func ThreadReader(userID pulid.ID, tenant pagination.TenantInfo) *permissionports.RequestActor {
	tenant.UserID = userID

	return permissionports.UserActor(tenant)
}

func markAll(threads []*conversation.Thread, reason conversation.ContinueRefusal) {
	for _, thread := range threads {
		if thread != nil {
			thread.MarkNotContinuable(reason)
		}
	}
}

// continueRefusal says why a reader who may use the assistant may not ask
// the agent, or "" when they may. An agent they may not use is refused as
// such before anything else is said about it, so a reader is never told an
// agent they could not use anyway was merely turned off.
func continueRefusal(
	definition *agentdefinition.Definition,
	usable *permissionports.UsableAgents,
) conversation.ContinueRefusal {
	switch {
	case definition == nil:
		return conversation.ContinueAgentDeleted
	case !usable.Allows(definition):
		return conversation.ContinueNoAccess
	case !definition.Enabled:
		return conversation.ContinueAgentDisabled
	case AssertChatAgent(definition) != nil:
		return conversation.ContinueAgentNotConversational
	default:
		return ""
	}
}
