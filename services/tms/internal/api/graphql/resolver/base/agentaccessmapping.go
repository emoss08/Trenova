package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// agentDefinitionAccessRoles reads the roles granted an agent through the
// request's loader. A reader who may not read roles is shown none.
func (r *Resolver) AgentDefinitionAccessRoles(
	ctx context.Context,
	obj *agentdefinition.Definition,
) ([]*permission.Role, error) {
	authCtx, err := r.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !r.HasPermission(ctx, authCtx, permission.ResourceRole, permission.OpRead) {
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
func (r *Resolver) RoleAgents(
	ctx context.Context,
	obj *permission.Role,
) ([]*agentdefinition.Definition, error) {
	authCtx, err := r.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !r.HasPermission(ctx, authCtx, permission.ResourceAgentDefinition, permission.OpRead) {
		return []*agentdefinition.Definition{}, nil
	}

	loadersForRequest, ok := loaders.FromContext(ctx)
	if !ok || loadersForRequest == nil {
		return nil, errortypes.NewDatabaseError("Agent access loader is not configured")
	}

	return loadersForRequest.AgentsByRoleID.Load(ctx, obj.ID.String())
}
