package fieldsensitivity

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

type GroupCheck func(ctx context.Context, resource permission.Resource) (bool, error)

func VisibleGroups(ctx context.Context, check GroupCheck) permission.FieldGroupSet {
	var visible permission.FieldGroupSet
	if check == nil {
		return visible
	}
	for _, group := range permission.FieldGroups() {
		allowed, err := check(ctx, group.Resource())
		if err == nil && allowed {
			visible = visible.With(group)
		}
	}
	return visible
}

type FieldAccess struct {
	engine permissionChecker
}

func NewFieldAccess(engine permissionports.PermissionEngine) *FieldAccess {
	return &FieldAccess{engine: engine}
}

func (a *FieldAccess) Visible(
	ctx context.Context,
	actor *permissionports.RequestActor,
) permission.FieldGroupSet {
	if actor == nil || actor.PrincipalType == permissionports.PrincipalTypeSystem {
		return permission.AllFieldGroups()
	}
	if a == nil || a.engine == nil {
		return 0
	}

	return VisibleGroups(ctx, func(ctx context.Context, resource permission.Resource) (bool, error) {
		result, err := a.engine.Check(ctx, actor.PermissionCheck(resource, permission.OpRead))
		if err != nil || result == nil {
			return false, err
		}
		return result.Allowed, nil
	})
}
