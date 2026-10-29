package agentaccess

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/subjectaccess"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// AssertMayUseAgent refuses a conversation, or a turn in one, with an agent
// the person may not use. A check that cannot be made is a refusal.
func (s *Access) AssertMayUseAgent(
	ctx context.Context,
	actor *permissionports.RequestActor,
	definition *agentdefinition.Definition,
) error {
	if s.Permissions == nil || actor == nil {
		return errAgentWithheld(definition)
	}

	allowed, err := s.Permissions.MayUseAgent(ctx, actor, definition)
	if err != nil {
		return fmt.Errorf("check access to agent %s: %w", definition.ID, err)
	}
	if !allowed {
		return errAgentWithheld(definition)
	}

	return nil
}

// UsableAgents is what the person may use of the organization's agents in a
// conversation. Without a permission engine nothing is usable.
func (s *Access) UsableAgents(
	ctx context.Context,
	actor *permissionports.RequestActor,
) (*permissionports.UsableAgents, error) {
	if s.Permissions == nil || actor == nil {
		return &permissionports.UsableAgents{}, nil
	}

	usable, err := s.Permissions.AgentsUsable(ctx, actor, permission.OpCreate)
	if err != nil {
		return nil, fmt.Errorf("check which agents the person may use: %w", err)
	}
	if usable == nil {
		return &permissionports.UsableAgents{}, nil
	}

	return usable, nil
}

// MarkContinuable says on each conversation whether its reader may still ask
// its agent anything and, when not, why: the agent was removed, the reader
// may not use it, it was turned off, or it no longer takes conversations. A
// reader who may not use the assistant at all may not use any agent. The
// agents are read once for the whole page.
func (s *Access) MarkContinuable(
	ctx context.Context,
	reader *permissionports.RequestActor,
	threads ...*conversation.Thread,
) error {
	if len(threads) == 0 {
		return nil
	}
	markAll(threads, conversation.ContinueNoAccess)
	if s.Definitions == nil {
		return nil
	}

	usable, err := s.UsableAgents(ctx, reader)
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
		definitions, listErr := s.Definitions.ListByIDs(
			ctx,
			agentdefinitionports.ListAgentDefinitionsByIDsRequest{
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

func (s *Access) everyRead() []string {
	if s.Queries == nil {
		return nil
	}

	tools := s.Queries.All()
	reads := make([]string, 0, len(tools))
	for _, tool := range tools {
		if agentextension.IsExtensionTool(tool.Name()) {
			continue
		}
		reads = append(reads, tool.Name())
	}

	return reads
}

// UsableDefinition is the thread's agent, refused when it is switched off,
// is not a chat agent, or is not one the person may use. On a quick question's
// conversation it also holds every read, narrowed to the person's as any
// tool is.
func (s *Access) UsableDefinition(
	ctx context.Context,
	thread *conversation.Thread,
	actor *permissionports.RequestActor,
	req *assistantports.SendMessageRequest,
) (*agentdefinition.Definition, error) {
	definition, err := s.UsableAgent(ctx, thread.AgentDefinitionID, actor, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if thread.Origin.ReadsEverything() {
		return definition.HoldingReads(s.everyRead()), nil
	}

	return definition, nil
}

// UsableAgent is an agent the person may talk to: enabled, a chat agent,
// and one their roles give them.
func (s *Access) UsableAgent(
	ctx context.Context,
	id pulid.ID,
	actor *permissionports.RequestActor,
	tenant pagination.TenantInfo,
) (*agentdefinition.Definition, error) {
	definition, err := s.Definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID:         id,
			TenantInfo: tenant,
		},
	)
	if err != nil {
		return nil, err
	}
	if !definition.Enabled {
		return nil, errortypes.NewBusinessError(
			"Agent {0} is disabled and cannot be used", definition.Name,
		)
	}
	if err = AssertChatAgent(definition); err != nil {
		return nil, err
	}
	if err = s.AssertMayUseAgent(ctx, actor, definition); err != nil {
		return nil, err
	}

	return definition, nil
}

type Access struct {
	Permissions permissionports.PermissionEngine
	Definitions agentdefinitionports.AgentDefinitionRepository
	Queries     agenttoolports.AgentQueryToolRegistry
}

func (s *Access) MayReadSubject(
	ctx context.Context,
	actor *permissionports.RequestActor,
	subjectType agent.SubjectType,
) (bool, error) {
	return subjectaccess.MayRead(ctx, s.Permissions, actor, subjectType)
}

func (s *Access) AssertSubjectReadable(
	ctx context.Context,
	actor *permissionports.RequestActor,
	subjectType agent.SubjectType,
) error {
	return subjectaccess.AssertReadable(ctx, s.Permissions, actor, subjectType)
}
