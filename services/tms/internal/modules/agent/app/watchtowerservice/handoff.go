package watchtowerservice

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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
	for _, definition := range listed.Items {
		if definition != nil && definition.TriggerMode != agentdefinition.TriggerChat {
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
	if s.runs == nil {
		return nil, errortypes.NewBusinessError("Agents cannot be started on this deployment")
	}
	if item.IsResolved() {
		return nil, errortypes.NewBusinessError("This exception is resolved; reopen it first")
	}
	if err := s.assertNoAgentWorking(ctx, req.TenantInfo, item); err != nil {
		return nil, err
	}

	run, err := s.runs.StartForDefinition(ctx, &agentports.StartAgentRunForDefinitionRequest{
		DefinitionID: req.AgentDefinitionID,
		SubjectType:  item.SubjectType,
		SubjectID:    item.SubjectID,
		Trigger:      agent.RunTriggerManual,
		EventKind:    item.EventKind,
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

func (s *Service) assertNoAgentWorking(
	ctx context.Context,
	tenant pagination.TenantInfo,
	item *watchtower.Item,
) error {
	if !item.HasAgent() || s.runRepo == nil {
		return nil
	}

	run, err := s.runRepo.GetByID(ctx, agentruntimeports.GetAgentRunByIDRequest{
		ID:         item.AgentRunID,
		TenantInfo: &tenant,
	})
	if err != nil {
		if dberror.IsNotFoundError(err) || errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if item.AgentWorkFrom(run).Active() {
		return errortypes.NewBusinessError("An agent is already working on this")
	}

	return nil
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
