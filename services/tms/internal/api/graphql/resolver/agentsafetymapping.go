package resolver

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentsafetyservice"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/emoss08/trenova/shared/pulid"
)

func toolPolicyViewsToModel(
	views []services.AgentToolPolicyView,
) []*gqlmodel.AgentToolPolicy {
	out := make([]*gqlmodel.AgentToolPolicy, 0, len(views))
	for idx := range views {
		out = append(out, toolPolicyViewToModel(&views[idx]))
	}

	return out
}

func toolPolicyViewToModel(view *services.AgentToolPolicyView) *gqlmodel.AgentToolPolicy {
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
		out.Effect = agent.ToolEffectLookup
	}
	if !out.ReadsExternal.IsValid() {
		out.ReadsExternal = agent.ExternalReadNever
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

	return out
}

func toolPolicyConnectionRequest(
	input *gqlmodel.AgentToolPolicyConnectionInput,
) memtable.Request {
	req := memtable.Request{
		First: intValue(input.First),
		After: stringValue(input.After),
		Query: stringValue(input.Query),
	}
	if input.Egress != nil {
		req.FieldFilters = append(req.FieldFilters, domaintypes.FieldFilter{
			Field:    "egress",
			Operator: dbtype.OpEqual,
			Value:    input.Egress.String(),
		})
	}
	if input.Resource != nil {
		req.FieldFilters = append(req.FieldFilters, domaintypes.FieldFilter{
			Field:    "resource",
			Operator: dbtype.OpEqual,
			Value:    *input.Resource,
		})
	}
	if input.Kind != nil {
		req.FieldFilters = append(req.FieldFilters, domaintypes.FieldFilter{
			Field:    "kind",
			Operator: dbtype.OpEqual,
			Value:    input.Kind.String(),
		})
	}
	if input.RunsWithoutPerson != nil {
		req.FieldFilters = append(req.FieldFilters, domaintypes.FieldFilter{
			Field:    agentsafetyservice.FieldRunsWithoutPerson,
			Operator: dbtype.OpEqual,
			Value:    *input.RunsWithoutPerson,
		})
	}

	return req
}

func agentToolSafetyPageToModel(
	page *services.AgentToolSafetyPage,
) *gqlmodel.AgentToolSafetyConnection {
	out := &gqlmodel.AgentToolSafetyConnection{
		Edges:      make([]*gqlmodel.AgentToolSafetyEdge, 0, len(page.Edges)),
		PageInfo:   &gqlmodel.PageInfo{HasNextPage: page.HasNextPage},
		TotalCount: page.TotalCount,
	}
	for idx := range page.Edges {
		edge := &page.Edges[idx]
		out.Edges = append(out.Edges, &gqlmodel.AgentToolSafetyEdge{
			Node:   &edge.Node,
			Cursor: edge.Cursor,
		})
	}
	if count := len(out.Edges); count > 0 {
		endCursor := out.Edges[count-1].Cursor
		out.PageInfo.EndCursor = &endCursor
	}

	return out
}

func toolPolicyPageToModel(
	page *services.AgentToolPolicyPage,
) *gqlmodel.AgentToolPolicyConnection {
	out := &gqlmodel.AgentToolPolicyConnection{
		Edges:      make([]*gqlmodel.AgentToolPolicyEdge, 0, len(page.Edges)),
		PageInfo:   &gqlmodel.PageInfo{HasNextPage: page.HasNextPage},
		TotalCount: page.TotalCount,
	}
	for idx := range page.Edges {
		edge := &page.Edges[idx]
		node := toolPolicyViewToModel(&edge.View)
		node.RunsWithoutPerson = edge.RunsWithoutPerson
		out.Edges = append(out.Edges, &gqlmodel.AgentToolPolicyEdge{
			Node:   node,
			Cursor: edge.Cursor,
		})
	}
	if count := len(out.Edges); count > 0 {
		endCursor := out.Edges[count-1].Cursor
		out.PageInfo.EndCursor = &endCursor
	}

	return out
}

func (r *Resolver) agentToolSafetyPolicy(
	obj *services.AgentToolSafety,
) (*gqlmodel.AgentToolPolicy, error) {
	if obj == nil {
		return nil, errortypes.NewNotFoundError("Tool not found")
	}

	view, ok := r.agentSafetyService.ToolPolicy(obj.PolicyName)
	if !ok {
		return nil, errortypes.NewNotFoundError("Tool policy not found")
	}

	return toolPolicyViewToModel(&view), nil
}

func tierOr(tier, fallback agent.AutonomyTier) agent.AutonomyTier {
	if tier.IsValid() {
		return tier
	}

	return fallback
}

func agentSafetyIDs(ids []string) ([]pulid.ID, error) {
	if ids == nil {
		return nil, nil
	}

	return parseIDs(ids)
}

func (r *Resolver) agentSafetyTools(
	ctx context.Context,
	obj *services.AgentSafetySubject,
) ([]*services.AgentToolSafety, error) {
	if obj == nil || obj.Agent == nil {
		return []*services.AgentToolSafety{}, nil
	}

	loadersForRequest, ok := loaders.FromContext(ctx)
	if !ok || loadersForRequest == nil {
		return nil, errortypes.NewDatabaseError("Agent trust loader is not configured")
	}

	trust, err := loadersForRequest.ToolTrustByAgentID.Load(ctx, obj.Agent.ID.String())
	if err != nil {
		return nil, err
	}

	assessed := r.agentSafetyService.Assess(ctx, &services.AssessAgentSafetyRequest{
		Subject: obj,
		Trust:   trust,
	})
	out := make([]*services.AgentToolSafety, 0, len(assessed))
	for idx := range assessed {
		out = append(out, &assessed[idx])
	}

	return out, nil
}

func (r *Resolver) agentSafetyReach(
	ctx context.Context,
	obj *services.AgentSafetySubject,
) (*gqlmodel.AgentReach, error) {
	if obj == nil || obj.Agent == nil {
		return nil, errortypes.NewNotFoundError("Agent not found")
	}

	authCtx, err := r.requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	loadersForRequest, ok := loaders.FromContext(ctx)
	if !ok || loadersForRequest == nil {
		return nil, errortypes.NewDatabaseError("Agent access loader is not configured")
	}

	granted, err := loadersForRequest.AccessRolesByAgentID.Load(ctx, obj.Agent.ID.String())
	if err != nil {
		return nil, err
	}

	warnings := r.agentSafetyService.ReachWarnings(&services.AgentReachRequest{
		Subject:      obj,
		GrantedRoles: len(granted),
	})
	reach := &gqlmodel.AgentReach{
		AccessMode: obj.Agent.EffectiveAccessMode(),
		Roles:      []*permission.Role{},
		Warnings:   make([]*services.AgentReachWarning, 0, len(warnings)),
	}
	for idx := range warnings {
		reach.Warnings = append(reach.Warnings, &warnings[idx])
	}
	if r.hasPermission(ctx, authCtx, permission.ResourceRole, permission.OpRead) {
		reach.Roles = granted
	}

	return reach, nil
}
