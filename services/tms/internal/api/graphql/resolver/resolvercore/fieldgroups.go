package resolvercore

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
)

// VisibleFieldGroups is the protected field groups an operation's caller may
// be shown. A person holds a group through its read grant; an agent read is
// further held to its data access, by fieldsensitivity.AgentVisibleGroups.
// Each group's check goes through the operation's permission memo.
func VisibleFieldGroups(
	ctx context.Context,
	engine permissionports.PermissionEngine,
	authCtx *authctx.AuthContext,
) protectedfield.GroupSet {
	if authCtx == nil || engine == nil {
		return 0
	}
	held := fieldsensitivity.VisibleGroups(
		ctx,
		func(ctx context.Context, resource permission.Resource) (bool, error) {
			return CheckPermission(ctx, engine, authCtx, resource, permission.OpRead)
		},
	)
	if ceiling, agentRead := gqlctx.FieldCeilingFrom(ctx); agentRead {
		return fieldsensitivity.AgentVisibleGroups(ctx, held, ceiling.For)
	}

	return held
}

func (r *Core) VisibleFieldGroups(
	ctx context.Context,
	authCtx *authctx.AuthContext,
) protectedfield.GroupSet {
	return VisibleFieldGroups(ctx, r.PermissionEngine, authCtx)
}
