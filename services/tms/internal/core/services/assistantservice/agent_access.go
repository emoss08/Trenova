package assistantservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// assertMayUseAgent refuses a conversation, or a turn in one, with an agent
// the person may not use. A check that cannot be made is a refusal.
func (s *Service) assertMayUseAgent(
	ctx context.Context,
	actor *services.RequestActor,
	definition *agentdefinition.Definition,
) error {
	if s.permissions == nil || actor == nil {
		return errAgentWithheld(definition)
	}

	allowed, err := s.permissions.MayUseAgent(ctx, actor, definition)
	if err != nil {
		return fmt.Errorf("check access to agent %s: %w", definition.ID, err)
	}
	if !allowed {
		return errAgentWithheld(definition)
	}

	return nil
}

func errAgentWithheld(definition *agentdefinition.Definition) error {
	return errortypes.NewAuthorizationError(
		"You do not have access to {0}. An administrator can give one of your roles access to it.",
		definition.Name,
	)
}

func errAgentsWithheld() error {
	return errortypes.NewAuthorizationError(
		"You don't have permission to perform this action: {0} {1}",
		permission.ResourceAssistant, permission.OpCreate,
	)
}

// usableAgents is what the person may use of the organization's agents in a
// conversation. Without a permission engine nothing is usable.
func (s *Service) usableAgents(
	ctx context.Context,
	actor *services.RequestActor,
) (*services.UsableAgents, error) {
	if s.permissions == nil || actor == nil {
		return &services.UsableAgents{}, nil
	}

	usable, err := s.permissions.AgentsUsable(ctx, actor, permission.OpCreate)
	if err != nil {
		return nil, fmt.Errorf("check which agents the person may use: %w", err)
	}
	if usable == nil {
		return &services.UsableAgents{}, nil
	}

	return usable, nil
}

// threadReader is the person a conversation is read for. Conversations are
// read under the reader's own user id, so the reader is its owner.
func threadReader(userID pulid.ID, tenant pagination.TenantInfo) *services.RequestActor {
	tenant.UserID = userID

	return services.UserActor(tenant)
}

// markContinuable says on each conversation whether its reader may still ask
// its agent anything and, when not, why: the agent was removed, the reader
// may not use it, it was turned off, or it no longer takes conversations. A
// reader who may not use the assistant at all may not use any agent. The
// agents are read once for the whole page.
func (s *Service) markContinuable(
	ctx context.Context,
	reader *services.RequestActor,
	threads ...*conversation.Thread,
) error {
	if len(threads) == 0 {
		return nil
	}
	markAll(threads, conversation.ContinueNoAccess)
	if s.definitions == nil {
		return nil
	}

	usable, err := s.usableAgents(ctx, reader)
	if err != nil {
		return err
	}
	if !usable.Assistant {
		return nil
	}

	ids := make([]pulid.ID, 0, len(threads))
	seen := make(map[pulid.ID]struct{}, len(threads))
	for _, thread := range threads {
		if thread == nil || thread.AgentDefinitionID.IsNil() {
			continue
		}
		if _, ok := seen[thread.AgentDefinitionID]; ok {
			continue
		}
		seen[thread.AgentDefinitionID] = struct{}{}
		ids = append(ids, thread.AgentDefinitionID)
	}

	byID := make(map[pulid.ID]*agentdefinition.Definition, len(ids))
	if len(ids) > 0 {
		definitions, listErr := s.definitions.ListByIDs(
			ctx,
			repositories.ListAgentDefinitionsByIDsRequest{
				IDs:        ids,
				TenantInfo: reader.TenantInfo(),
			},
		)
		if listErr != nil {
			return fmt.Errorf("read the agents of the conversations: %w", listErr)
		}
		for _, definition := range definitions {
			byID[definition.ID] = definition
		}
	}

	for _, thread := range threads {
		if thread == nil {
			continue
		}
		refusal := continueRefusal(byID[thread.AgentDefinitionID], usable)
		if refusal == "" {
			thread.MarkContinuable()
			continue
		}
		thread.MarkNotContinuable(refusal)
	}

	return nil
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
	usable *services.UsableAgents,
) conversation.ContinueRefusal {
	switch {
	case definition == nil:
		return conversation.ContinueAgentDeleted
	case !usable.Allows(definition):
		return conversation.ContinueNoAccess
	case !definition.Enabled:
		return conversation.ContinueAgentDisabled
	case assertChatAgent(definition) != nil:
		return conversation.ContinueAgentNotConversational
	default:
		return ""
	}
}
