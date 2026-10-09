package assistantcaseservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"go.uber.org/zap"
)

// abilities says who can take each step the checklist offers. The person's
// own permissions are asked once, for every tool the steps name; the
// conversation's agent is read once; the person's other agents only when
// the conversation's cannot take a step. It is advice for the Desk, so a
// failure to work it out is logged and the Desk asks the agent as before.
func (s *Service) abilities(
	ctx context.Context,
	req *serviceports.CaseThreadRequest,
	thread *conversation.Thread,
	checklist *deskcase.Checklist,
) map[deskcase.StepKey]deskcase.StepAbility {
	steps := checklist.Steps()
	if len(steps) == 0 || s.runtime == nil || s.definitions == nil {
		return nil
	}

	out, err := s.resolveAbilities(ctx, req, thread, steps)
	if err != nil {
		s.l.Warn("could not work out who can take a case's steps",
			zap.String("thread", thread.ID.String()), zap.Error(err))
		return nil
	}

	return out
}

func (s *Service) resolveAbilities(
	ctx context.Context,
	req *serviceports.CaseThreadRequest,
	thread *conversation.Thread,
	steps []deskcase.StepKey,
) (map[deskcase.StepKey]deskcase.StepAbility, error) {
	permitted := make(map[string]struct{})
	for _, tool := range s.runtime.PermittedTools(ctx, req.Actor, deskcase.ToolsFor(steps)) {
		permitted[tool] = struct{}{}
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         thread.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, fmt.Errorf("read the conversation's agent: %w", err)
	}
	own := s.holder(definition)

	in := &deskcase.AbilityInputs{Steps: steps, Own: own, Permitted: permitted}
	if deskcase.NeedsOthers(steps, own, permitted) {
		if in.Others, err = s.otherHolders(ctx, req, definition); err != nil {
			return nil, err
		}
	}

	return deskcase.Abilities(in), nil
}

// otherHolders are the other enabled chat agents the person may use, by
// name, each with the tools it holds.
func (s *Service) otherHolders(
	ctx context.Context,
	req *serviceports.CaseThreadRequest,
	own *agentdefinition.Definition,
) ([]*deskcase.Holder, error) {
	others, err := agentruntime.OtherUsableAgents(ctx, &agentruntime.OtherUsableAgentsRequest{
		Permissions: s.permissions,
		Definitions: s.definitions,
		Actor:       req.Actor,
		Self:        own.ID,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*deskcase.Holder, 0, len(others))
	for _, definition := range others {
		out = append(out, s.holder(definition))
	}

	return out, nil
}

func (s *Service) holder(definition *agentdefinition.Definition) *deskcase.Holder {
	summaries := s.runtime.ToolSummaries(definition)
	tools := make(map[string]struct{}, len(summaries))
	for _, summary := range summaries {
		tools[summary.Name] = struct{}{}
	}

	return &deskcase.Holder{ID: definition.ID, Name: definition.Name, Tools: tools}
}
