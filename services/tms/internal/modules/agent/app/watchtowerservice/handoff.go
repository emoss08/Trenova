package watchtowerservice

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwork"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxHandOffAgents = 100

func (s *Service) HandOffAgents(
	ctx context.Context,
	tenant pagination.TenantInfo,
	_ *permissionports.RequestActor,
) ([]*agentdefinition.Definition, error) {
	return s.handOffCandidates(ctx, tenant)
}

func (s *Service) handOffCandidates(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]*agentdefinition.Definition, error) {
	listed, err := s.definitions.List(ctx, &agentdefinitionports.ListAgentDefinitionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: maxHandOffAgents},
		},
		EnabledOnly: true,
	})
	if err != nil {
		return nil, err
	}

	candidates := make([]*agentdefinition.Definition, 0, len(listed.Items))
	for _, definition := range withoutShadow(listed.Items) {
		if definition.TriggerMode != agentdefinition.TriggerChat {
			candidates = append(candidates, definition)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Name < candidates[j].Name
	})

	return candidates, nil
}

func (s *Service) handOffToAgent(
	ctx context.Context,
	req watchtowerports.HandOffWatchtowerItemRequest,
	item *watchtower.Item,
	actor *permissionports.RequestActor,
) (*watchtowerports.HandOffWatchtowerItemResult, error) {
	run, err := s.runs.StartForDefinition(ctx, &agentruntimeports.StartAgentRunForDefinitionRequest{
		DefinitionID: req.AgentDefinitionID,
		SubjectType:  item.SubjectType,
		SubjectID:    item.SubjectID,
		Trigger:      agent.RunTriggerManual,
		EventKind:    item.EventKind,
		Handed:       item.HandOffBrief(),
		Watched:      true,
		TenantInfo:   req.TenantInfo,
	}, actor)
	if err != nil {
		return nil, err
	}

	saved, err := s.exceptions.recordHandOff(ctx, &handOffRecord{
		tenant:       req.TenantInfo,
		itemID:       item.ID,
		definitionID: req.AgentDefinitionID,
		runID:        run.ID,
		actor:        actor,
	})
	if err != nil {
		return nil, err
	}

	return &watchtowerports.HandOffWatchtowerItemResult{Item: saved, Run: run}, nil
}

// refusePaused refuses to put an agent on an item while the organization's
// agents are paused: every run would be recorded as what the agent would
// have done, and the person would wait on a plan that can never be
// approved. A play is not an agent's work and is sent all the same.
func (s *Service) refusePaused(ctx context.Context, tenant pagination.TenantInfo) error {
	if s.shadow == nil {
		return nil
	}
	paused, err := s.shadow.Organization(ctx, tenant)
	if err != nil {
		return err
	}
	if paused {
		return errortypes.NewBusinessError(
			"Agents are paused for the organization, so none can be put on this. Lift the pause in AI control, or do it yourself.",
		)
	}

	return nil
}

// withoutShadow leaves out the agents switched to shadow. A hand-off asks
// for work a person can approve, and a shadow agent only records what it
// would have done, so it is neither picked for an item nor offered for one.
func withoutShadow(definitions []*agentdefinition.Definition) []*agentdefinition.Definition {
	live := make([]*agentdefinition.Definition, 0, len(definitions))
	for _, definition := range definitions {
		if definition != nil && !definition.ShadowMode {
			live = append(live, definition)
		}
	}

	return live
}

// agentStanding reads where the item's agent stands. It refuses an item an
// agent is working now, and answers the agent that already looked at the
// item as it still reads: that agent is not run on it again, named or
// picked, until the item says something new. Nil when no agent answered or
// the item changed since.
func (s *Service) agentStanding(
	ctx context.Context,
	tenant pagination.TenantInfo,
	item *watchtower.Item,
) (answered pulid.ID, err error) {
	if !item.HasAgent() {
		return pulid.Nil, nil
	}

	run, err := s.runRepo.GetByID(ctx, agentruntimeports.GetAgentRunByIDRequest{
		ID:         item.AgentRunID,
		TenantInfo: &tenant,
	})
	if err != nil {
		if dberror.IsNotFoundError(err) || errortypes.IsNotFoundError(err) {
			return pulid.Nil, nil
		}
		return pulid.Nil, err
	}
	if agentwork.From(item, run).Active() {
		return pulid.Nil, errortypes.NewBusinessError("An agent is already working on this")
	}
	if agentwork.AnsweredUnchanged(item, run) {
		return item.AgentDefinitionID, nil
	}

	return pulid.Nil, nil
}

// refuseAnswered says why the agent that already looked is not run again.
func (s *Service) refuseAnswered(
	ctx context.Context,
	tenant pagination.TenantInfo,
	definitionID pulid.ID,
) error {
	name := "That agent"
	definition, err := s.definitions.GetByID(ctx, agentdefinitionports.GetAgentDefinitionByIDRequest{
		ID:         definitionID,
		TenantInfo: tenant,
	})
	if err == nil && definition != nil && definition.Name != "" {
		name = definition.Name
	}

	return errortypes.NewBusinessError(
		"{0} already looked at this, and nothing about it has changed since. Do it yourself, or hand it to a different agent.",
		name,
	)
}

func without(
	definitions []*agentdefinition.Definition,
	id pulid.ID,
) []*agentdefinition.Definition {
	if id.IsNil() {
		return definitions
	}
	kept := make([]*agentdefinition.Definition, 0, len(definitions))
	for _, definition := range definitions {
		if definition != nil && definition.ID != id {
			kept = append(kept, definition)
		}
	}

	return kept
}

type handOffRecord struct {
	tenant       pagination.TenantInfo
	itemID       pulid.ID
	definitionID pulid.ID
	runID        pulid.ID
	actor        *permissionports.RequestActor
}

func (s *Exceptions) recordHandOff(
	ctx context.Context,
	record *handOffRecord,
) (*watchtower.Item, error) {
	return s.applyException(ctx, &exceptionChange{
		target: &watchtowerports.ExceptionTarget{
			TenantInfo: record.tenant,
			ItemID:     record.itemID,
		},
		actor: record.actor,
		mutate: func(item *watchtower.Item, now int64) ([]*watchtower.Event, error) {
			previous := item.AgentDefinitionID
			if err := item.HandTo(
				record.definitionID,
				record.runID,
				actorUserID(record.actor),
				now,
			); err != nil {
				return nil, err
			}
			return []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventHandedOff, actorUserID(record.actor)).
					WithChange(previous.String(), record.definitionID.String()),
			}, nil
		},
	})
}
