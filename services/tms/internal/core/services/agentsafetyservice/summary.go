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
	unattended    map[string]struct{}
	openSensitive []pulid.ID
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
		ToolCount:             len(s.entries),
		RunWithoutPerson:      len(survey.unattended),
		LeaveOrganization:     leaving,
		OpenWithSensitive:     len(survey.openSensitive),
		Resources:             slices.Clone(s.resources),
		EgressCounts:          s.egressCounts(),
		UnattendedTools:       s.titlesOf(survey.unattended),
		OpenSensitiveAgentIDs: survey.openSensitive,
	}, nil
}

func (s *Service) egressCounts() []services.AgentEgressCount {
	counts := make(map[agent.EgressClass]int, len(agent.EgressClasses()))
	for idx := range s.entries {
		policy := s.entries[idx].view.Policy
		if policy.EffectiveEffect() != agent.ToolEffectChange {
			continue
		}
		counts[widestEgress(policy.Egress)]++
	}

	out := make([]services.AgentEgressCount, 0, len(counts))
	for _, class := range agent.EgressClasses() {
		if counts[class] > 0 {
			out = append(out, services.AgentEgressCount{Egress: class, Count: counts[class]})
		}
	}

	return out
}

func widestEgress(classes []agent.EgressClass) agent.EgressClass {
	order := agent.EgressClasses()
	widest := agent.EgressNone
	for _, class := range classes {
		if slices.Index(order, class) > slices.Index(order, widest) {
			widest = class
		}
	}

	return widest
}

func (s *Service) titlesOf(names map[string]struct{}) []string {
	titles := make([]string, 0, len(names))
	for idx := range s.entries {
		view := s.entries[idx].view
		if _, ok := names[view.Policy.Name]; ok {
			titles = append(titles, view.Title)
		}
	}
	slices.Sort(titles)

	return titles
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

	out := &safetySurvey{unattended: make(map[string]struct{}), openSensitive: []pulid.ID{}}
	if len(definitions) == 0 {
		return out, nil
	}

	control, err := s.controls.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, fmt.Errorf("read agent control for safety: %w", err)
	}
	rules, err := s.overridesFor(ctx, tenantInfo)
	if err != nil {
		return nil, err
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
				rules:      rules,
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
	rules      map[string]*agent.ToolRuleOverride
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
		in.out.openSensitive = append(in.out.openSensitive, in.definition.ID)
	}

	for idx := range held {
		policy := agenttoolpolicy.ApplyOverrides(held[idx], in.rules)
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

func (s *Service) ToolHolders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]services.AgentToolHolders, error) {
	definitions, err := s.listDefinitions(ctx, &services.ListAgentSafetyRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	byTool := make(map[string][]pulid.ID, len(s.entries))
	for _, definition := range definitions {
		for _, policy := range s.heldPolicies(definition) {
			holders := byTool[policy.Name]
			if len(holders) > 0 && holders[len(holders)-1] == definition.ID {
				continue
			}
			byTool[policy.Name] = append(holders, definition.ID)
		}
	}

	out := make([]services.AgentToolHolders, 0, len(byTool))
	for idx := range s.entries {
		name := s.entries[idx].view.Policy.Name
		if holders, ok := byTool[name]; ok {
			out = append(out, services.AgentToolHolders{PolicyName: name, AgentIDs: holders})
		}
	}

	return out, nil
}

func (s *Service) RuleImpact(
	ctx context.Context,
	req *services.RuleImpactRequest,
) ([]services.AgentToolRuleImpact, error) {
	definitions, err := s.listDefinitions(ctx, &services.ListAgentSafetyRequest{
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	holders := make([]*agentdefinition.Definition, 0, len(definitions))
	for _, definition := range definitions {
		if slices.ContainsFunc(s.heldPolicies(definition), func(policy services.ToolPolicy) bool {
			return policy.Name == req.ToolName
		}) {
			holders = append(holders, definition)
		}
	}
	out := make([]services.AgentToolRuleImpact, 0, len(holders))
	if len(holders) == 0 {
		return out, nil
	}

	control, err := s.controls.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, fmt.Errorf("read agent control for safety: %w", err)
	}
	trust, err := s.trustFor(ctx, req.TenantInfo, holders)
	if err != nil {
		return nil, err
	}

	for _, definition := range holders {
		var held *agent.ToolTrust
		for _, row := range trust[definition.ID] {
			if row != nil && row.ToolName == req.ToolName {
				held = row
			}
		}
		answer := func(policy services.ToolPolicy) agent.AutonomyAnswer {
			return agenttoolpolicy.Assess(ctx, &agenttoolpolicy.AssessInput{
				Policy:     policy,
				Definition: definition,
				Trust:      held,
				Control:    control,
			}).Answer
		}
		out = append(out, services.AgentToolRuleImpact{
			AgentID:   definition.ID,
			AgentName: definition.Name,
			Before:    answer(req.Before),
			After:     answer(req.After),
		})
	}

	return out, nil
}
