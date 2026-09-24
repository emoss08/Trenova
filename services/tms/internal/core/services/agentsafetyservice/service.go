package agentsafetyservice

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentaccessservice"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
)

const (
	MaxAgentIDs  = 100
	listPageSize = pagination.MaxLimit
	maxAgents    = 2000
)

type toolPolicies interface {
	Get(name string) (services.ToolPolicy, bool)
	All() []services.ToolPolicy
}

type toolHolder interface {
	ToolSummaries(definition *agentdefinition.Definition) []agentdefinition.ToolSummary
}

type Params struct {
	fx.In

	Policies    *agenttoolpolicy.Catalog
	Runtime     services.AgentRuntime
	Definitions repositories.AgentDefinitionRepository
	Controls    repositories.AgentControlRepository
	Registry    *permission.Registry
}

type Service struct {
	policies    toolPolicies
	holder      toolHolder
	definitions repositories.AgentDefinitionRepository
	controls    repositories.AgentControlRepository
	sensitive   agentaccessservice.SensitiveToolRule
	views       []services.AgentToolPolicyView
}

//nolint:gocritic // dependency injection
func New(p Params) services.AgentSafetyService {
	return newService(serviceDeps{
		policies:    p.Policies,
		holder:      p.Runtime,
		definitions: p.Definitions,
		controls:    p.Controls,
		sensitive:   agentaccessservice.DefaultSensitiveRule(p.Registry),
	})
}

type serviceDeps struct {
	policies    toolPolicies
	holder      toolHolder
	definitions repositories.AgentDefinitionRepository
	controls    repositories.AgentControlRepository
	sensitive   agentaccessservice.SensitiveToolRule
}

func newService(deps serviceDeps) *Service {
	svc := &Service{
		policies:    deps.policies,
		holder:      deps.holder,
		definitions: deps.definitions,
		controls:    deps.controls,
		sensitive:   deps.sensitive,
	}
	svc.views = PolicyViews(deps.policies.All())

	return svc
}

func PolicyViews(policies []services.ToolPolicy) []services.AgentToolPolicyView {
	views := make([]services.AgentToolPolicyView, 0, len(policies))
	for idx := range policies {
		views = append(views, PolicyView(policies[idx]))
	}

	return views
}

func PolicyView(policy services.ToolPolicy) services.AgentToolPolicyView {
	view := services.AgentToolPolicyView{
		Policy:      policy,
		Title:       stringutils.HumanizeCamelCaseSentence(policy.Name),
		Promotable:  agenttoolpolicy.Promotable(policy),
		Leaves:      slices.ContainsFunc(policy.Egress, agent.EgressClass.Leaves),
		Explanation: agenttoolpolicy.ExplainTier(policy),
	}

	held := agentaccessservice.HeldToolOf(policy)
	if held.Resource != "" {
		view.Needs = &services.ToolGrant{Resource: held.Resource, Operation: held.Operation}
	}

	return view
}

func (s *Service) ToolPolicies() []services.AgentToolPolicyView {
	return slices.Clone(s.views)
}

func (s *Service) ListSubjects(
	ctx context.Context,
	req *services.ListAgentSafetyRequest,
) ([]*services.AgentSafetySubject, error) {
	definitions, err := s.listDefinitions(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(definitions) == 0 {
		return []*services.AgentSafetySubject{}, nil
	}

	control, err := s.controls.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, fmt.Errorf("read agent control for safety: %w", err)
	}

	subjects := make([]*services.AgentSafetySubject, 0, len(definitions))
	for _, definition := range definitions {
		subjects = append(subjects, &services.AgentSafetySubject{
			Agent:   definition,
			Control: control,
		})
	}

	return subjects, nil
}

func (s *Service) Assess(
	ctx context.Context,
	req *services.AssessAgentSafetyRequest,
) []services.AgentToolSafety {
	if req == nil || req.Subject == nil || req.Subject.Agent == nil {
		return []services.AgentToolSafety{}
	}

	trustByTool := make(map[string]*agent.ToolTrust, len(req.Trust))
	for _, row := range req.Trust {
		if row != nil {
			trustByTool[row.ToolName] = row
		}
	}

	held := s.heldPolicies(req.Subject.Agent)
	out := make([]services.AgentToolSafety, 0, len(held))
	for idx := range held {
		policy := held[idx]
		input := agenttoolpolicy.AssessInput{
			Policy:     policy,
			Definition: req.Subject.Agent,
			Trust:      trustByTool[policy.Name],
			Control:    req.Subject.Control,
		}
		clean := agenttoolpolicy.Assess(ctx, &input)
		input.Tainted = true
		tainted := agenttoolpolicy.Assess(ctx, &input)

		out = append(out, services.AgentToolSafety{
			PolicyName: policy.Name,
			Clean:      clean,
			Tainted:    tainted,
		})
	}

	return out
}

func (s *Service) ReachWarnings(req *services.AgentReachRequest) []services.AgentReachWarning {
	warnings := make([]services.AgentReachWarning, 0, 2)
	if req == nil || req.Subject == nil || req.Subject.Agent == nil {
		return warnings
	}

	definition := req.Subject.Agent
	if definition.OpenToEveryone() {
		policies := s.heldPolicies(definition)
		held := make([]agentaccessservice.HeldTool, 0, len(policies))
		for idx := range policies {
			held = append(held, agentaccessservice.HeldToolOf(policies[idx]))
		}
		if sensitive := agentaccessservice.SensitiveTools(held, s.sensitive); len(sensitive) > 0 {
			warnings = append(warnings, services.AgentReachWarning{
				Kind:  agentdefinition.ReachOpenWithSensitiveTools,
				Tools: sensitive,
			})
		}
	}
	if definition.ReachableByNobody(req.GrantedRoles) {
		warnings = append(warnings, services.AgentReachWarning{
			Kind:  agentdefinition.ReachNoAudience,
			Tools: []string{},
		})
	}

	return warnings
}

func (s *Service) heldPolicies(definition *agentdefinition.Definition) []services.ToolPolicy {
	if s.holder == nil || s.policies == nil {
		return []services.ToolPolicy{}
	}

	summaries := s.holder.ToolSummaries(definition)
	out := make([]services.ToolPolicy, 0, len(summaries))
	seen := make(map[string]struct{}, len(summaries))
	for _, summary := range summaries {
		if _, duplicate := seen[summary.Name]; duplicate {
			continue
		}
		seen[summary.Name] = struct{}{}

		if policy, ok := s.policies.Get(summary.Name); ok {
			out = append(out, policy)
		}
	}

	return out
}

func (s *Service) listDefinitions(
	ctx context.Context,
	req *services.ListAgentSafetyRequest,
) ([]*agentdefinition.Definition, error) {
	if req.AgentIDs != nil {
		return s.definitionsByID(ctx, req)
	}

	out := make([]*agentdefinition.Definition, 0, listPageSize)
	for offset := 0; offset < maxAgents; offset += listPageSize {
		page, err := s.definitions.List(ctx, &repositories.ListAgentDefinitionRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: req.TenantInfo,
				Pagination: pagination.Info{Limit: listPageSize, Offset: offset},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("list agents for safety: %w", err)
		}

		out = append(out, page.Items...)
		if len(page.Items) < listPageSize || len(out) >= page.Total {
			break
		}
	}

	return out, nil
}

func (s *Service) definitionsByID(
	ctx context.Context,
	req *services.ListAgentSafetyRequest,
) ([]*agentdefinition.Definition, error) {
	ids, err := validateAgentIDs(req.AgentIDs)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []*agentdefinition.Definition{}, nil
	}

	definitions, err := s.definitions.ListByIDs(ctx, repositories.ListAgentDefinitionsByIDsRequest{
		IDs:        ids,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, fmt.Errorf("read agents for safety: %w", err)
	}

	slices.SortStableFunc(definitions, func(a, b *agentdefinition.Definition) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return definitions, nil
}

func validateAgentIDs(ids []pulid.ID) ([]pulid.ID, error) {
	if len(ids) > MaxAgentIDs {
		return nil, errortypes.NewValidationError(
			"agentIds", errortypes.ErrInvalid, "At most 100 agents can be named at once",
		)
	}

	multiErr := errortypes.NewMultiError()
	seen := make(map[pulid.ID]struct{}, len(ids))
	out := make([]pulid.ID, 0, len(ids))
	for idx, id := range ids {
		if id.IsNil() {
			multiErr.Add(fmt.Sprintf("agentIds[%d]", idx), errortypes.ErrRequired,
				"Agent cannot be empty")
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return out, nil
}
