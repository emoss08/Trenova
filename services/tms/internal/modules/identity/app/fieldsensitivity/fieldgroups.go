package fieldsensitivity

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

type GroupCheck func(ctx context.Context, resource permission.Resource) (bool, error)

func VisibleGroups(ctx context.Context, check GroupCheck) protectedfield.GroupSet {
	var visible protectedfield.GroupSet
	if check == nil {
		return visible
	}
	for _, group := range protectedfield.Groups() {
		allowed, err := check(ctx, group.Resource())
		if err == nil && allowed {
			visible = visible.With(group)
		}
	}
	return visible
}

// AgentGroupTier is the tier an agent's data access must reach on a group's
// resource before an agent read carries a figure of that group. A protected
// field of a record is Confidential and never reaches an agent; a computed
// figure (a margin percent on a capacity match, a tool's margin) has no tier
// of its own and is shown at this one, so an agent is held to the person's
// group and to its own data access, as a person is held to their group.
const AgentGroupTier = permission.SensitivityRestricted

// AgentVisibleGroups narrows the groups an actor holds to those an agent read
// may carry: each only where the agent's ceiling on the group's resource
// reaches AgentGroupTier.
func AgentVisibleGroups(
	ctx context.Context,
	held protectedfield.GroupSet,
	ceiling func(ctx context.Context, resource permission.Resource) permission.FieldSensitivity,
) protectedfield.GroupSet {
	var visible protectedfield.GroupSet
	if ceiling == nil {
		return visible
	}
	for _, group := range protectedfield.Groups() {
		if held.Has(group) && ceiling(ctx, group.Resource()).CanAccess(AgentGroupTier) {
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
) protectedfield.GroupSet {
	if actor == nil || actor.PrincipalType == permissionports.PrincipalTypeSystem {
		return protectedfield.AllGroups()
	}
	if a == nil || a.engine == nil {
		return 0
	}

	return VisibleGroups(
		ctx,
		func(ctx context.Context, resource permission.Resource) (bool, error) {
			result, err := a.engine.Check(ctx, actor.PermissionCheck(resource, permission.OpRead))
			if err != nil || result == nil {
				return false, err
			}
			return result.Allowed, nil
		},
	)
}
