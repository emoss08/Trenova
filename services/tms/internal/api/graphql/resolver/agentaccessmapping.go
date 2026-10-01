package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxMyAgentIDs = 100

// myAgentFilters narrows the agents the caller may use to what the picker
// asked for. Only these filters are accepted: a free filter over agents
// would let a reader probe fields they are not shown. None is true when the
// input asks for no agent at all.
func myAgentFilters(
	input *gqlmodel.MyAgentsInput,
) (filters []*gqlmodel.FieldFilterInput, none bool, err error) {
	filters = make([]*gqlmodel.FieldFilterInput, 0, 3)
	if input.Origin != nil {
		switch *input.Origin {
		case gqlmodel.MyAgentOriginTemplate:
			filters = append(filters, &gqlmodel.FieldFilterInput{
				Field: "template", Operator: "isnotnull",
			})
		case gqlmodel.MyAgentOriginCustom:
			filters = append(filters, &gqlmodel.FieldFilterInput{
				Field: "template", Operator: "isnull",
			})
		case gqlmodel.MyAgentOriginAll:
		}
	}

	excluded, err := myAgentIDList("excludeIds", input.ExcludeIds)
	if err != nil {
		return nil, false, err
	}
	if len(excluded) > 0 {
		filters = append(filters, &gqlmodel.FieldFilterInput{
			Field: "id", Operator: "notin", Value: excluded,
		})
	}

	if input.Ids != nil {
		only, idErr := myAgentIDList("ids", input.Ids)
		if idErr != nil {
			return nil, false, idErr
		}
		if len(only) == 0 {
			return nil, true, nil
		}
		filters = append(filters, &gqlmodel.FieldFilterInput{
			Field: "id", Operator: "in", Value: only,
		})
	}

	return filters, false, nil
}

func myAgentIDList(field string, ids []string) ([]any, error) {
	if len(ids) > maxMyAgentIDs {
		return nil, errortypes.NewValidationError(
			field, errortypes.ErrInvalid, "At most 100 agents can be named at once",
		)
	}

	parsed, err := parseIDs(ids)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(parsed))
	for _, id := range parsed {
		out = append(out, id.String())
	}

	return out, nil
}

func myAgentConnectionToModel(
	result *pagination.CursorListResult[*agentdefinition.Definition],
) (*gqlmodel.MyAgentConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *agentdefinition.Definition, cursor string) *gqlmodel.MyAgentEdge {
			return &gqlmodel.MyAgentEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.MyAgentEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.MyAgentConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func emptyMyAgentConnection(ctx context.Context) *gqlmodel.MyAgentConnection {
	connection := &gqlmodel.MyAgentConnection{
		Edges:    []*gqlmodel.MyAgentEdge{},
		PageInfo: &gqlmodel.PageInfo{},
	}
	if connectionFieldRequested(ctx, connectionTotalCountField) {
		zero := 0
		connection.TotalCount = &zero
	}

	return connection
}

// agentDefinitionAccessRoles reads the roles granted an agent through the
// request's loader. A reader who may not read roles is shown none.
func (r *Resolver) agentDefinitionAccessRoles(
	ctx context.Context,
	obj *agentdefinition.Definition,
) ([]*permission.Role, error) {
	authCtx, err := r.requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !r.hasPermission(ctx, authCtx, permission.ResourceRole, permission.OpRead) {
		return []*permission.Role{}, nil
	}

	loadersForRequest, ok := loaders.FromContext(ctx)
	if !ok || loadersForRequest == nil {
		return nil, errortypes.NewDatabaseError("Agent access loader is not configured")
	}

	return loadersForRequest.AccessRolesByAgentID.Load(ctx, obj.ID.String())
}

// roleAgents reads the agents a role is granted through the request's
// loader. A reader who may not read agents is shown none.
func (r *Resolver) roleAgents(
	ctx context.Context,
	obj *permission.Role,
) ([]*agentdefinition.Definition, error) {
	authCtx, err := r.requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !r.hasPermission(ctx, authCtx, permission.ResourceAgentDefinition, permission.OpRead) {
		return []*agentdefinition.Definition{}, nil
	}

	loadersForRequest, ok := loaders.FromContext(ctx)
	if !ok || loadersForRequest == nil {
		return nil, errortypes.NewDatabaseError("Agent access loader is not configured")
	}

	return loadersForRequest.AgentsByRoleID.Load(ctx, obj.ID.String())
}

func audienceSuggestionToModel(
	suggestion *services.AgentAudienceSuggestion,
) *gqlmodel.AgentAudienceSuggestion {
	return &gqlmodel.AgentAudienceSuggestion{
		AgentID:        suggestion.Agent.ID.String(),
		AccessMode:     suggestion.Agent.EffectiveAccessMode(),
		Roles:          audienceRolesToModel(suggestion.Roles),
		SensitiveTools: suggestion.SensitiveTools,
	}
}

func accessPreviewToModel(
	suggestion *services.AgentAudienceSuggestion,
) *gqlmodel.AgentAccessPreview {
	return &gqlmodel.AgentAccessPreview{
		AccessMode:     suggestion.Agent.EffectiveAccessMode(),
		Roles:          audienceRolesToModel(suggestion.Roles),
		SensitiveTools: suggestion.SensitiveTools,
	}
}

func audienceRolesToModel(entries []services.AgentAudienceRole) []*gqlmodel.AgentAudienceRole {
	roles := make([]*gqlmodel.AgentAudienceRole, 0, len(entries))
	for i := range entries {
		entry := &entries[i]
		missing := make([]string, 0, len(entry.MissingResources))
		for _, resource := range entry.MissingResources {
			missing = append(missing, resource.String())
		}
		roles = append(roles, &gqlmodel.AgentAudienceRole{
			Role:             entry.Role,
			Coverage:         entry.Coverage,
			MissingResources: missing,
			Granted:          entry.Granted,
		})
	}

	return roles
}

// accessPreviewRequest reads the form's agent from the preview input. The
// agent being edited is optional; a new one has no id yet.
func accessPreviewRequest(
	tenant pagination.TenantInfo,
	input *gqlmodel.AgentAccessPreviewInput,
) (*services.PreviewAgentAudienceRequest, error) {
	req := &services.PreviewAgentAudienceRequest{
		TenantInfo: tenant,
		ToolNames:  input.ToolNames,
		Mode:       input.AccessMode,
	}
	if input.AgentID == nil || *input.AgentID == "" {
		return req, nil
	}

	agentID, err := pulid.MustParse(*input.AgentID)
	if err != nil {
		return nil, err
	}
	req.AgentID = agentID

	return req, nil
}

func myAgentColumns(ctx context.Context, nodePathPrefix string) []string {
	return definitionColumns(ctx, projection.MyAgentSpec, nodePathPrefix)
}

func definitionTemplate(obj *agentdefinition.Definition) *agentdefinition.Template {
	if obj.Template == "" {
		return nil
	}
	template := obj.Template

	return &template
}
