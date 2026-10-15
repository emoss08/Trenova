package agentsuggestionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentsuggestionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Repository  agentsuggestionports.AgentSuggestionRepository
	Definitions agentdefinitionports.AgentDefinitionRepository
	Permissions permissionports.PermissionEngine
}

type Service struct {
	repo        agentsuggestionports.AgentSuggestionRepository
	definitions agentdefinitionports.AgentDefinitionRepository
	permissions permissionports.PermissionEngine
}

func New(p Params) agentsuggestionports.AgentSuggestionService {
	return &Service{repo: p.Repository, definitions: p.Definitions, permissions: p.Permissions}
}

func (s *Service) RecordMine(
	ctx context.Context,
	req *agentsuggestionports.RecordAgentSuggestionRequest,
	actor *permissionports.RequestActor,
) (*agentsuggestion.Suggestion, error) {
	if err := requirePerson(actor); err != nil {
		return nil, err
	}
	if !req.Outcome.Opens() {
		return nil, errortypes.NewValidationError(
			"outcome",
			errortypes.ErrInvalid,
			"A suggestion starts out shown, or moved to without asking",
		)
	}

	entity := &agentsuggestion.Suggestion{
		OrganizationID:   req.TenantInfo.OrgID,
		BusinessUnitID:   req.TenantInfo.BuID,
		UserID:           actor.UserID,
		SuggestedAgentID: req.SuggestedAgentID,
		Surface:          req.Surface,
		Outcome:          req.Outcome,
	}
	if req.CurrentAgentID.IsNotNil() {
		current := req.CurrentAgentID
		entity.CurrentAgentID = &current
	}

	me := errortypes.NewMultiError()
	entity.Validate(me)
	if me.HasErrors() {
		return nil, me
	}
	if err := s.requireAgents(ctx, req, actor); err != nil {
		return nil, err
	}

	return s.repo.Insert(ctx, entity)
}

func (s *Service) SettleMine(
	ctx context.Context,
	req *agentsuggestionports.SettleAgentSuggestionRequest,
	actor *permissionports.RequestActor,
) (*agentsuggestion.Suggestion, error) {
	if err := requirePerson(actor); err != nil {
		return nil, err
	}
	if req.ID.IsNil() {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrRequired,
			"Name the suggestion to settle",
		)
	}
	if !req.Outcome.Settles() {
		return nil, errortypes.NewValidationError(
			"outcome",
			errortypes.ErrInvalid,
			"A suggestion is settled by switching, sending to it, keeping your agent or passing it over",
		)
	}

	return s.repo.Settle(ctx, agentsuggestionports.SettleAgentSuggestionRow{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
		UserID:     actor.UserID,
		Outcome:    req.Outcome,
	})
}

func (s *Service) requireAgents(
	ctx context.Context,
	req *agentsuggestionports.RecordAgentSuggestionRequest,
	actor *permissionports.RequestActor,
) error {
	ids := []pulid.ID{req.SuggestedAgentID}
	if req.CurrentAgentID.IsNotNil() {
		ids = append(ids, req.CurrentAgentID)
	}

	found, err := s.definitions.ListByIDs(ctx, agentdefinitionports.ListAgentDefinitionsByIDsRequest{
		IDs:        ids,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}

	known := make(map[pulid.ID]*agentdefinition.Definition, len(found))
	for _, definition := range found {
		known[definition.ID] = definition
	}
	suggested, ok := known[req.SuggestedAgentID]
	if !ok {
		return errortypes.NewValidationError(
			"suggestedAgentId",
			errortypes.ErrInvalid,
			"The agent suggested is not one of this organization's agents",
		)
	}
	usable, err := s.permissions.MayUseAgent(ctx, actor, suggested)
	if err != nil {
		return err
	}
	if !usable {
		return errortypes.NewValidationError(
			"suggestedAgentId",
			errortypes.ErrInvalid,
			"The agent suggested is not one you may use",
		)
	}
	if req.CurrentAgentID.IsNotNil() {
		if _, ok := known[req.CurrentAgentID]; !ok {
			return errortypes.NewValidationError(
				"currentAgentId",
				errortypes.ErrInvalid,
				"The agent the question would have gone to is not one of this organization's agents",
			)
		}
	}

	return nil
}

func requirePerson(actor *permissionports.RequestActor) error {
	if !actor.IsUser() || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError("Only a person's own suggestions are recorded")
	}

	return nil
}
