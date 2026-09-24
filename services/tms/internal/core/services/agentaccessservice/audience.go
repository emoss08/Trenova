package agentaccessservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const maxPreviewTools = 200

func (s *Service) SuggestAudience(
	ctx context.Context,
	req *services.SuggestAgentAudienceRequest,
) (*services.AgentAudienceSuggestion, error) {
	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.AgentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return s.audience(ctx, req.TenantInfo, definition, true)
}

// PreviewAudience works the suggestion out for the agent a form holds: the
// saved agent, when one is being edited, with the form's tools and access in
// place of its own, or a new agent holding only the form's tools. Nothing is
// written. Tools that are not registered are passed over, as they are for a
// saved agent.
func (s *Service) PreviewAudience(
	ctx context.Context,
	req *services.PreviewAgentAudienceRequest,
) (*services.AgentAudienceSuggestion, error) {
	toolNames, err := previewToolNames(req.ToolNames)
	if err != nil {
		return nil, err
	}
	if !req.Mode.IsValid() {
		return nil, errortypes.NewValidationError(
			"accessMode", errortypes.ErrInvalid, "Access must be Everyone or Roles",
		)
	}

	preview := &agentdefinition.Definition{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		TriggerMode:    agentdefinition.TriggerChat,
	}
	saved := req.AgentID.IsNotNil()
	if saved {
		existing, getErr := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
			ID:         req.AgentID,
			TenantInfo: req.TenantInfo,
		})
		if getErr != nil {
			return nil, getErr
		}
		copied := *existing
		preview = &copied
	}
	preview.ToolNames = agentdefinition.WithoutCoreTools(toolNames)
	preview.AccessMode = req.Mode

	return s.audience(ctx, req.TenantInfo, preview, saved)
}

func previewToolNames(names []string) ([]string, error) {
	if len(names) > maxPreviewTools {
		return nil, errortypes.NewValidationError(
			"toolNames", errortypes.ErrInvalid, "At most 200 tools can be previewed at once",
		)
	}

	return sliceutils.DedupeStrings(names), nil
}

// audience is every role in the tenant with its coverage of what the agent
// holds, and, while the agent is open to everyone, the tools it holds that
// are sensitive. withGrants reads which roles are granted the agent now; an
// agent not yet saved has none.
func (s *Service) audience(
	ctx context.Context,
	tenant pagination.TenantInfo,
	definition *agentdefinition.Definition,
	withGrants bool,
) (*services.AgentAudienceSuggestion, error) {
	held := s.HeldTools(definition)
	roles, err := s.grants.ListGrantableRoles(ctx, repositories.ListGrantableRolesRequest{
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	roleIDs := make([]pulid.ID, 0, len(roles))
	for _, role := range roles {
		roleIDs = append(roleIDs, role.ID)
	}

	coverage, err := s.permissions.RoleCoverage(ctx, &services.RoleCoverageRequest{
		OrganizationID: tenant.OrgID,
		RoleIDs:        roleIDs,
		Required:       requiredGrants(held),
	})
	if err != nil {
		return nil, err
	}

	var granted []*permission.Role
	if withGrants {
		byAgent, grantErr := s.grants.ListRolesByAgents(ctx, repositories.ListGrantsByAgentsRequest{
			TenantInfo: tenant,
			AgentIDs:   []pulid.ID{definition.ID},
		})
		if grantErr != nil {
			return nil, grantErr
		}
		granted = byAgent[definition.ID]
	}

	suggestion := &services.AgentAudienceSuggestion{
		Agent:          definition,
		Roles:          audienceRoles(roles, coverage, granted),
		SensitiveTools: []string{},
	}
	if definition.OpenToEveryone() {
		suggestion.SensitiveTools = SensitiveTools(held, s.sensitive)
	}

	return suggestion, nil
}

// HeldTools is every registered tool the agent holds with the grant it needs
// of the person using it and where its work can go. A tool that acts only on
// the person's own records needs no grant and names no resource.
func (s *Service) HeldTools(definition *agentdefinition.Definition) []HeldTool {
	if s.runtime == nil || definition == nil {
		return []HeldTool{}
	}

	summaries := s.runtime.ToolSummaries(definition)
	held := make([]HeldTool, 0, len(summaries))
	seen := make(map[string]struct{}, len(summaries))
	for _, summary := range summaries {
		if _, duplicate := seen[summary.Name]; duplicate {
			continue
		}
		seen[summary.Name] = struct{}{}

		if tool, ok := s.heldTool(summary.Name); ok {
			held = append(held, tool)
		}
	}

	return held
}

func (s *Service) heldTool(name string) (HeldTool, bool) {
	if s.policies == nil {
		return HeldTool{}, false
	}

	policy, ok := s.policies.Get(name)
	if !ok {
		return HeldTool{}, false
	}

	return HeldToolOf(policy), true
}

func HeldToolOf(policy services.ToolPolicy) HeldTool {
	held := HeldTool{Name: policy.Name, Egress: policy.Egress}
	if policy.Scope == agent.ToolScopeSelf || policy.Resource == "" {
		return held
	}

	held.Resource = policy.Resource
	held.Operation = policy.Operation
	if held.Operation == "" {
		held.Operation = permission.OpRead
	}

	return held
}

func requiredGrants(held []HeldTool) []services.RequiredGrant {
	required := make([]services.RequiredGrant, 0, len(held))
	for _, tool := range held {
		if tool.Resource == "" {
			continue
		}
		required = append(required, services.RequiredGrant{
			Tool:      tool.Name,
			Resource:  tool.Resource,
			Operation: tool.Operation,
		})
	}

	return required
}

func audienceRoles(
	roles []*permission.Role,
	coverage []services.RoleCoverage,
	granted []*permission.Role,
) []services.AgentAudienceRole {
	byRole := make(map[pulid.ID]services.RoleCoverage, len(coverage))
	for _, entry := range coverage {
		byRole[entry.RoleID] = entry
	}

	grantedIDs := make(map[pulid.ID]struct{}, len(granted))
	for _, role := range granted {
		grantedIDs[role.ID] = struct{}{}
	}

	out := make([]services.AgentAudienceRole, 0, len(roles))
	for _, role := range roles {
		entry, ok := byRole[role.ID]
		if !ok {
			entry = services.RoleCoverage{
				RoleID:           role.ID,
				Coverage:         agentdefinition.CoverageNone,
				MissingResources: []permission.Resource{},
			}
		}
		_, isGranted := grantedIDs[role.ID]
		out = append(out, services.AgentAudienceRole{
			Role:             role,
			Coverage:         entry.Coverage,
			MissingResources: entry.MissingResources,
			Granted:          isGranted,
		})
	}

	return out
}
