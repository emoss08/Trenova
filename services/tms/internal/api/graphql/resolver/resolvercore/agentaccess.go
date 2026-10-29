package resolvercore

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/agentloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// agentDefinitionAccessRoles reads the roles granted an agent through the
// request's loader. A reader who may not read roles is shown none.
func (r *Core) AgentDefinitionAccessRoles(
	ctx context.Context,
	obj *agentdefinition.Definition,
) ([]*rbac.Role, error) {
	authCtx, err := r.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !r.HasPermission(ctx, authCtx, permission.ResourceRole, permission.OpRead) {
		return []*rbac.Role{}, nil
	}

	if !loadercore.Available(ctx) {
		return nil, errortypes.NewDatabaseError("Agent access loader is not configured")
	}

	return agentloaders.AccessRolesByAgentID(ctx, obj.ID.String())
}

// roleAgents reads the agents a role is granted through the request's
// loader. A reader who may not read agents is shown none.
func (r *Core) RoleAgents(
	ctx context.Context,
	obj *rbac.Role,
) ([]*agentdefinition.Definition, error) {
	authCtx, err := r.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !r.HasPermission(ctx, authCtx, permission.ResourceAgentDefinition, permission.OpRead) {
		return []*agentdefinition.Definition{}, nil
	}

	if !loadercore.Available(ctx) {
		return nil, errortypes.NewDatabaseError("Agent access loader is not configured")
	}

	return agentloaders.AgentsByRoleID(ctx, obj.ID.String())
}
