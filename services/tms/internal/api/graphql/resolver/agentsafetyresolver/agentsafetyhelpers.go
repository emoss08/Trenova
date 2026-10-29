package agentsafetyresolver

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/agentloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func ToolPolicyViewToModel(view *agenttoolports.AgentToolPolicyView) *gqlmodel.AgentToolPolicy {
	policy := &view.Policy
	out := &gqlmodel.AgentToolPolicy{
		ID:                 policy.Name,
		Name:               policy.Name,
		Title:              view.Title,
		Kind:               policy.Kind,
		Scope:              policy.Scope,
		DefaultTier:        tierOr(policy.DefaultTier, agent.TierPropose),
		MaxTier:            tierOr(policy.MaxTier, agent.TierAutoExecute),
		PromotableTier:     view.Promotable,
		Egress:             slices.Clone(policy.Egress),
		LeavesOrganization: view.Leaves,
		HasClassify:        policy.Classify != nil,
		HasCondition:       policy.Condition != nil,
		PersonalExemption:  policy.PersonalRunsUnasked,
		Effect:             policy.EffectiveEffect(),
		Artifact:           policy.Artifact,
		Reversible:         policy.Reversible,
		Idempotent:         policy.Idempotent,
		ReadsExternal:      policy.ReadsExternal,
		CarriesTaint:       policy.CarriesTaint,
		Rationale:          policy.Rationale,
		Explanation:        view.Explanation,
	}
	if !out.Effect.IsValid() {
		out.Effect = agenttooltrust.ToolEffectLookup
	}
	if !out.ReadsExternal.IsValid() {
		out.ReadsExternal = agenttooltrust.ExternalReadNever
	}
	if view.Needs != nil {
		out.Needs = &gqlmodel.AgentToolRequirement{
			Resource:  view.Needs.Resource.String(),
			Operation: string(view.Needs.Operation),
		}
	}
	if policy.Condition != nil {
		description := policy.Condition.Description
		out.ConditionDescription = &description
	}
	if policy.Source.IsValid() {
		source := policy.Source
		out.Source = &source
	}
	applyRule(out, view)

	return out
}

func applyRule(out *gqlmodel.AgentToolPolicy, view *agenttoolports.AgentToolPolicyView) {
	declared := &view.Policy
	if view.Declared != nil {
		declared = view.Declared
	}
	out.DeclaredMaxTier = tierOr(declared.MaxTier, agent.TierAutoExecute)
	out.DeclaredReadsExternal = declared.ReadsExternal
	if !out.DeclaredReadsExternal.IsValid() {
		out.DeclaredReadsExternal = agenttooltrust.ExternalReadNever
	}
	if view.Override == nil {
		return
	}
	out.RuleVersion = int(view.Override.Version)
	if !view.Override.Empty() {
		out.Rule = view.Override
	}
}

func (r *Deps) AgentToolSafetyPolicy(
	obj *agenttoolports.AgentToolSafety,
) (*gqlmodel.AgentToolPolicy, error) {
	if obj == nil {
		return nil, errortypes.NewNotFoundError("Tool not found")
	}
	if obj.Policy != nil {
		return ToolPolicyViewToModel(obj.Policy), nil
	}

	view, ok := r.AgentSafetyService.ToolPolicy(obj.PolicyName)
	if !ok {
		return nil, errortypes.NewNotFoundError("Tool policy not found")
	}

	return ToolPolicyViewToModel(&view), nil
}

func tierOr(tier, fallback agent.AutonomyTier) agent.AutonomyTier {
	if tier.IsValid() {
		return tier
	}

	return fallback
}

func (r *Deps) AgentSafetyTools(
	ctx context.Context,
	obj *agenttoolports.AgentSafetySubject,
) ([]*agenttoolports.AgentToolSafety, error) {
	if obj == nil || obj.Agent == nil {
		return []*agenttoolports.AgentToolSafety{}, nil
	}

	if !loadercore.Available(ctx) {
		return nil, errortypes.NewDatabaseError("Agent trust loader is not configured")
	}

	trust, err := agentloaders.ToolTrustByAgentID(ctx, obj.Agent.ID.String())
	if err != nil {
		return nil, err
	}

	assessed := r.AgentSafetyService.Assess(ctx, &agenttoolports.AssessAgentSafetyRequest{
		Subject: obj,
		Trust:   trust,
	})
	out := make([]*agenttoolports.AgentToolSafety, 0, len(assessed))
	for idx := range assessed {
		out = append(out, &assessed[idx])
	}

	return out, nil
}

func (r *Deps) AgentSafetyReach(
	ctx context.Context,
	obj *agenttoolports.AgentSafetySubject,
) (*gqlmodel.AgentReach, error) {
	if obj == nil || obj.Agent == nil {
		return nil, errortypes.NewNotFoundError("Agent not found")
	}

	authCtx, err := r.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	if !loadercore.Available(ctx) {
		return nil, errortypes.NewDatabaseError("Agent access loader is not configured")
	}

	granted, err := agentloaders.AccessRolesByAgentID(ctx, obj.Agent.ID.String())
	if err != nil {
		return nil, err
	}

	warnings := r.AgentSafetyService.ReachWarnings(&agenttoolports.AgentReachRequest{
		Subject:      obj,
		GrantedRoles: len(granted),
	})
	reach := &gqlmodel.AgentReach{
		AccessMode: obj.Agent.EffectiveAccessMode(),
		Roles:      []*rbac.Role{},
		Warnings:   make([]*agenttoolports.AgentReachWarning, 0, len(warnings)),
	}
	for idx := range warnings {
		reach.Warnings = append(reach.Warnings, &warnings[idx])
	}
	if r.HasPermission(ctx, authCtx, permission.ResourceRole, permission.OpRead) {
		reach.Roles = granted
	}

	return reach, nil
}

func toolRuleChange(input *gqlmodel.AgentToolRuleInput) agenttoolports.ToolRuleChange {
	var change agenttoolports.ToolRuleChange
	if input.MaxTier != nil {
		change.MaxTier = *input.MaxTier
	}
	if input.ReadsExternal != nil {
		change.ReadsExternal = *input.ReadsExternal
	}

	return change
}

func impactPointers(
	impacts []agenttoolports.AgentToolRuleImpact,
) []*agenttoolports.AgentToolRuleImpact {
	out := make([]*agenttoolports.AgentToolRuleImpact, len(impacts))
	for idx := range impacts {
		out[idx] = &impacts[idx]
	}

	return out
}
