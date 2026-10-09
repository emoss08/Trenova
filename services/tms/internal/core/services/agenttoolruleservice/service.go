package agenttoolruleservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

type toolPolicies interface {
	Get(name string) (services.ToolPolicy, bool)
}

type ruleImpact interface {
	RuleImpact(ctx context.Context, req *services.RuleImpactRequest) ([]services.AgentToolRuleImpact, error)
}

type Params struct {
	fx.In

	Repo     repositories.AgentToolRuleOverrideRepository
	Loader   *Loader
	Policies *agenttoolpolicy.Catalog
	Safety   services.AgentSafetyService
	Auditor  services.SecurityAuditor
}

type Service struct {
	repo     repositories.AgentToolRuleOverrideRepository
	loader   *Loader
	policies toolPolicies
	impact   ruleImpact
	auditor  services.SecurityAuditor
}

var _ services.AgentToolRuleService = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		repo:     p.Repo,
		loader:   p.Loader,
		policies: p.Policies,
		impact:   p.Safety,
		auditor:  p.Auditor,
	}
}

func (s *Service) For(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[string]*agent.ToolRuleOverride, error) {
	return s.loader.For(ctx, tenantInfo)
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	toolName string,
) (*agent.ToolRuleOverride, error) {
	return s.repo.Get(ctx, repositories.GetToolRuleOverrideRequest{
		TenantInfo: tenantInfo,
		ToolName:   toolName,
	})
}

func (s *Service) Impact(
	ctx context.Context,
	req *services.ToolRuleImpactRequest,
) ([]services.AgentToolRuleImpact, error) {
	declared, err := s.declared(req.ToolName)
	if err != nil {
		return nil, err
	}
	current, err := s.Get(ctx, req.TenantInfo, req.ToolName)
	if err != nil {
		return nil, err
	}

	return s.impact.RuleImpact(ctx, &services.RuleImpactRequest{
		TenantInfo: req.TenantInfo,
		ToolName:   req.ToolName,
		Before:     agenttoolpolicy.ApplyOverride(declared, current),
		After:      agenttoolpolicy.ApplyOverride(declared, overrideOf(declared, req.Change)),
	})
}

func (s *Service) Save(
	ctx context.Context,
	req *services.SaveToolRuleRequest,
	actor *services.RequestActor,
) (*services.SaveToolRuleResult, error) {
	declared, err := s.declared(req.ToolName)
	if err != nil {
		return nil, err
	}
	existing, err := s.Get(ctx, req.TenantInfo, req.ToolName)
	if err != nil {
		return nil, err
	}

	loaded := int64(0)
	if existing != nil {
		loaded = existing.Version
	}
	if req.Version != loaded {
		return nil, dberror.CreateVersionMismatchError("Tool rule", req.ToolName)
	}

	next := overrideOf(declared, req.Change)
	next.Reason = strings.TrimSpace(req.Reason)
	before := agenttoolpolicy.ApplyOverride(declared, existing)
	after := agenttoolpolicy.ApplyOverride(declared, next)

	multiErr := errortypes.NewMultiError()
	next.Validate(agent.ToolRuleBounds{
		MaxTier:       declared.MaxTier,
		ReadsExternal: declared.ReadsExternal,
		Changes:       declared.EffectiveEffect() == agent.ToolEffectChange,
	}, multiErr)
	if before.MaxTier != after.MaxTier && next.Reason == "" {
		multiErr.Add("reason", errortypes.ErrRequired,
			"Say why the most freedom changes; the reason is kept in the audit trail")
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	affected, err := s.impact.RuleImpact(ctx, &services.RuleImpactRequest{
		TenantInfo: req.TenantInfo,
		ToolName:   req.ToolName,
		Before:     before,
		After:      after,
	})
	if err != nil {
		return nil, err
	}

	saved, err := s.write(ctx, &writeRequest{
		tenantInfo: req.TenantInfo,
		toolName:   req.ToolName,
		existing:   existing,
		next:       next,
		actor:      actor,
	})
	if err != nil {
		return nil, err
	}
	s.loader.Forget(req.TenantInfo)
	s.record(ctx, &recordRequest{
		tenantInfo: req.TenantInfo,
		toolName:   req.ToolName,
		before:     before,
		after:      after,
		reason:     next.Reason,
		affected:   affected,
		actor:      actor,
	})

	return &services.SaveToolRuleResult{Override: saved, Affected: changedOnly(affected)}, nil
}

func (s *Service) declared(toolName string) (services.ToolPolicy, error) {
	policy, ok := s.policies.Get(toolName)
	if !ok {
		return services.ToolPolicy{}, errortypes.NewValidationError(
			"toolName", errortypes.ErrInvalid, "No tool is named {0}", toolName,
		)
	}

	return policy, nil
}

func overrideOf(declared services.ToolPolicy, change services.ToolRuleChange) *agent.ToolRuleOverride {
	override := &agent.ToolRuleOverride{ToolName: declared.Name}
	if change.MaxTier != "" && change.MaxTier != declared.MaxTier {
		override.MaxTier = change.MaxTier
	}
	if change.ReadsExternal != "" && change.ReadsExternal != declared.ReadsExternal {
		override.ReadsExternal = change.ReadsExternal
	}

	return override
}

type writeRequest struct {
	tenantInfo pagination.TenantInfo
	toolName   string
	existing   *agent.ToolRuleOverride
	next       *agent.ToolRuleOverride
	actor      *services.RequestActor
}

func (s *Service) write(ctx context.Context, req *writeRequest) (*agent.ToolRuleOverride, error) {
	updatedBy := req.actor.PersonUserID()
	if req.existing == nil {
		return s.repo.Create(ctx, &agent.ToolRuleOverride{
			OrganizationID: req.tenantInfo.OrgID,
			BusinessUnitID: req.tenantInfo.BuID,
			ToolName:       req.toolName,
			MaxTier:        req.next.MaxTier,
			ReadsExternal:  req.next.ReadsExternal,
			Reason:         req.next.Reason,
			UpdatedByID:    updatedBy,
		})
	}

	updated := *req.existing
	updated.MaxTier = req.next.MaxTier
	updated.ReadsExternal = req.next.ReadsExternal
	updated.Reason = req.next.Reason
	updated.UpdatedByID = updatedBy

	return s.repo.Update(ctx, &updated)
}

type recordRequest struct {
	tenantInfo pagination.TenantInfo
	toolName   string
	before     services.ToolPolicy
	after      services.ToolPolicy
	reason     string
	affected   []services.AgentToolRuleImpact
	actor      *services.RequestActor
}

func (s *Service) record(ctx context.Context, req *recordRequest) {
	if s.auditor == nil {
		return
	}

	changed := changedOnly(req.affected)
	agents := make([]map[string]string, 0, len(changed))
	for _, impact := range changed {
		agents = append(agents, map[string]string{
			"agentId": impact.AgentID.String(),
			"name":    impact.AgentName,
			"before":  string(impact.Before),
			"after":   string(impact.After),
		})
	}

	s.auditor.RecordChange(ctx, &services.SecurityChange{
		Resource:       permission.ResourceAgentControl,
		ResourceID:     req.toolName,
		Operation:      permission.OpUpdate,
		Actor:          req.actor.AuditActor(),
		OrganizationID: req.tenantInfo.OrgID,
		BusinessUnitID: req.tenantInfo.BuID,
		Before: map[string]any{
			"tool":          req.toolName,
			"maxTier":       string(req.before.MaxTier),
			"readsExternal": string(req.before.ReadsExternal),
		},
		After: map[string]any{
			"tool":          req.toolName,
			"maxTier":       string(req.after.MaxTier),
			"readsExternal": string(req.after.ReadsExternal),
		},
		Comment:  "Tool rule changed: " + req.reason,
		Metadata: map[string]any{"reason": req.reason, "affectedAgents": agents},
	})
}

func changedOnly(impacts []services.AgentToolRuleImpact) []services.AgentToolRuleImpact {
	out := make([]services.AgentToolRuleImpact, 0, len(impacts))
	for _, impact := range impacts {
		if impact.Before != impact.After {
			out = append(out, impact)
		}
	}

	return out
}
