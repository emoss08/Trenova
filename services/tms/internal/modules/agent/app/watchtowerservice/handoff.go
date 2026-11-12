package watchtowerservice

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
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
	if item.IsResolved() {
		return nil, errortypes.NewBusinessError("This exception is resolved; reopen it first")
	}
	if err := s.assertNoAgentWorking(ctx, req.TenantInfo, item); err != nil {
		return nil, err
	}

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

func (s *Service) assertNoAgentWorking(
	ctx context.Context,
	tenant pagination.TenantInfo,
	item *watchtower.Item,
) error {
	if !item.HasAgent() {
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

// handOffTemplates names the agent that works a kind of item no event
// announces. A missed stop, a late projection, a quiet truck and a tender run
// dry are found by a sweep, so no agent subscribes to them, and handing one
// over used to answer "choose who works it" every time. They are a
// dispatcher's exceptions, and the dispatch agent is the one that covers a
// move, so a hand-off with no agent named goes to it. An exception an agent
// raised is not listed: it already asked for a person.
//
//nolint:exhaustive // only the kinds no event announces and one agent plainly works
var handOffTemplates = map[watchtower.SourceKind]agenttemplate.Template{
	watchtower.SourceMissedPickup:    agenttemplate.TemplateDispatchAssignment,
	watchtower.SourceMissedDelivery:  agenttemplate.TemplateDispatchAssignment,
	watchtower.SourceEtaAtRisk:       agenttemplate.TemplateDispatchAssignment,
	watchtower.SourceNoCheckCall:     agenttemplate.TemplateDispatchAssignment,
	watchtower.SourceTenderExhausted: agenttemplate.TemplateDispatchAssignment,
}

// workerOf is the organization's agent that works this kind of item, among
// the ones a hand-off may go to: the first, by name, built from the template
// the kind is worked by. Nil when the kind has none or the organization runs
// no such agent, and the person chooses.
func workerOf(
	kind watchtower.SourceKind,
	candidates []*agentdefinition.Definition,
) *agentdefinition.Definition {
	template, ok := handOffTemplates[kind]
	if !ok {
		return nil
	}
	for _, candidate := range candidates {
		if candidate != nil && candidate.Template == template {
			return candidate
		}
	}

	return nil
}
