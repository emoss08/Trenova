package agentsafetyservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type toolTrustLister interface {
	ListByDefinitionIDs(
		ctx context.Context,
		req repositories.ListToolTrustByDefinitionsRequest,
	) (map[pulid.ID][]*agent.ToolTrust, error)
}

type safetySurvey struct {
	unattended        map[string]struct{}
	openWithSensitive int
}

func (s *Service) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.AgentSafetySummary, error) {
	survey, err := s.survey(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	leaving := 0
	for idx := range s.entries {
		if s.entries[idx].view.Leaves {
			leaving++
		}
	}

	return &services.AgentSafetySummary{
		ToolCount:         len(s.entries),
		RunWithoutPerson:  len(survey.unattended),
		LeaveOrganization: leaving,
		OpenWithSensitive: survey.openWithSensitive,
		Resources:         slices.Clone(s.resources),
	}, nil
}

func (s *Service) survey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*safetySurvey, error) {
	definitions, err := s.listDefinitions(ctx, &services.ListAgentSafetyRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	out := &safetySurvey{unattended: make(map[string]struct{})}
	if len(definitions) == 0 {
		return out, nil
	}

	control, err := s.controls.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, fmt.Errorf("read agent control for safety: %w", err)
	}

	for chunk := range slices.Chunk(definitions, listPageSize) {
		trust, trustErr := s.trustFor(ctx, tenantInfo, chunk)
		if trustErr != nil {
			return nil, trustErr
		}

		for _, definition := range chunk {
			s.surveyAgent(ctx, &surveyAgentInput{
				definition: definition,
				control:    control,
				trust:      trust[definition.ID],
				out:        out,
			})
		}
	}

	return out, nil
}

type surveyAgentInput struct {
	definition *agentdefinition.Definition
	control    *tenant.AgentControl
	trust      []*agent.ToolTrust
	out        *safetySurvey
}

func (s *Service) surveyAgent(ctx context.Context, in *surveyAgentInput) {
	held := s.heldPolicies(in.definition)
	trustByTool := make(map[string]*agent.ToolTrust, len(in.trust))
	for _, row := range in.trust {
		if row != nil {
			trustByTool[row.ToolName] = row
		}
	}
	if in.definition.OpenToEveryone() && len(s.sensitiveTools(held)) > 0 {
		in.out.openWithSensitive++
	}

	for idx := range held {
		policy := held[idx]
		if policy.EffectiveEffect() != agent.ToolEffectChange {
			continue
		}
		if _, known := in.out.unattended[policy.Name]; known {
			continue
		}

		answer := agenttoolpolicy.Assess(ctx, &agenttoolpolicy.AssessInput{
			Policy:     policy,
			Definition: in.definition,
			Trust:      trustByTool[policy.Name],
			Control:    in.control,
		}).Answer
		if RunsWithoutPerson(answer) {
			in.out.unattended[policy.Name] = struct{}{}
		}
	}
}

func RunsWithoutPerson(answer agent.AutonomyAnswer) bool {
	return answer == agent.AutonomyRunsOnItsOwn || answer == agent.AutonomyConditional
}

func (s *Service) trustFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	definitions []*agentdefinition.Definition,
) (map[pulid.ID][]*agent.ToolTrust, error) {
	if s.trust == nil {
		return map[pulid.ID][]*agent.ToolTrust{}, nil
	}

	ids := make([]pulid.ID, 0, len(definitions))
	for _, definition := range definitions {
		ids = append(ids, definition.ID)
	}

	trust, err := s.trust.ListByDefinitionIDs(ctx, repositories.ListToolTrustByDefinitionsRequest{
		TenantInfo:         tenantInfo,
		AgentDefinitionIDs: ids,
	})
	if err != nil {
		return nil, fmt.Errorf("read tool trust for safety: %w", err)
	}

	return trust, nil
}
