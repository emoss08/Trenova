package refview

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

// PersonAccess is a person's access for the length of one request. A person
// reads the records a resolver returned, and the records of kinds they can
// read, without a ceiling, as the screen always has; what they see of a
// linked record only through its reference view is held to their role's
// ceiling.
type PersonAccess struct {
	engine permissionports.PermissionEngine
	actor  *permissionports.RequestActor
	check  func(ctx context.Context, kind permission.Resource) (bool, error)
	detail func(
		ctx context.Context,
		kind permission.Resource,
	) (*permissionports.ResourcePermissionDetail, error)

	mu       sync.Mutex
	reads    map[permission.Resource]bool
	ceilings map[permission.Resource]permission.FieldSensitivity
}

// NewPersonAccess builds a person's access. check, when set, is how the
// surface already asks the engine for a read, so it shares that surface's
// memo; otherwise the engine is asked directly.
func NewPersonAccess(
	engine permissionports.PermissionEngine,
	actor *permissionports.RequestActor,
	check func(ctx context.Context, kind permission.Resource) (bool, error),
) *PersonAccess {
	return &PersonAccess{
		engine:   engine,
		actor:    actor,
		check:    check,
		reads:    make(map[permission.Resource]bool, 4),
		ceilings: make(map[permission.Resource]permission.FieldSensitivity, 4),
	}
}

func (a *PersonAccess) Reads(ctx context.Context, kind permission.Resource) bool {
	a.mu.Lock()
	readable, seen := a.reads[kind]
	a.mu.Unlock()
	if seen {
		return readable
	}

	readable = a.checkRead(ctx, kind)

	a.mu.Lock()
	a.reads[kind] = readable
	a.mu.Unlock()

	return readable
}

func (a *PersonAccess) checkRead(ctx context.Context, kind permission.Resource) bool {
	if a.check != nil {
		allowed, err := a.check(ctx, kind)
		return err == nil && allowed
	}
	if a.engine == nil || a.actor == nil {
		return false
	}
	result, err := a.engine.Check(ctx, a.actor.PermissionCheck(kind, permission.OpRead))

	return err == nil && result != nil && result.Allowed
}

func (a *PersonAccess) Ceiling(
	ctx context.Context,
	kind permission.Resource,
	reach Reach,
) (permission.FieldSensitivity, bool) {
	if reach != Nested || a.actor == nil || a.Reads(ctx, kind) {
		return "", false
	}
	a.mu.Lock()
	ceiling, seen := a.ceilings[kind]
	a.mu.Unlock()
	if seen {
		return ceiling, true
	}

	ceiling = a.personCeiling(ctx, kind)
	if ceiling == "" {
		ceiling = permission.SensitivityInternal
	}

	a.mu.Lock()
	a.ceilings[kind] = ceiling
	a.mu.Unlock()

	return ceiling, true
}

// WithDetail answers the person's ceilings from the surface's own read of
// their permission detail rather than asking the engine for each kind.
func (a *PersonAccess) WithDetail(
	detail func(
		ctx context.Context,
		kind permission.Resource,
	) (*permissionports.ResourcePermissionDetail, error),
) *PersonAccess {
	a.detail = detail
	return a
}

func (a *PersonAccess) personCeiling(
	ctx context.Context,
	kind permission.Resource,
) permission.FieldSensitivity {
	if a.detail == nil {
		return fieldsensitivity.PersonCeiling(
			ctx, a.engine, a.actor.UserID, a.actor.OrganizationID, kind,
		)
	}
	detail, err := a.detail(ctx, kind)
	if err != nil {
		return permission.SensitivityInternal
	}
	return fieldsensitivity.CeilingOf(detail)
}

// AgentAccess is an agent read's access: the agent's ceiling on every record
// it reads, and the kind's read for a record carried by another.
type AgentAccess struct {
	Ceilings interface {
		For(ctx context.Context, resource permission.Resource) permission.FieldSensitivity
		Reads(ctx context.Context, resource permission.Resource) bool
	}
}

func (a AgentAccess) Reads(ctx context.Context, kind permission.Resource) bool {
	if a.Ceilings == nil {
		return false
	}
	return a.Ceilings.Reads(ctx, kind)
}

func (a AgentAccess) Ceiling(
	ctx context.Context,
	kind permission.Resource,
	_ Reach,
) (permission.FieldSensitivity, bool) {
	if a.Ceilings == nil {
		return permission.SensitivityInternal, true
	}
	return a.Ceilings.For(ctx, kind), true
}

// RoleAccess is what a role would be shown, from its grants alone: what a
// person holding only that role (and the roles it inherits) reads, with the
// same ceiling on what they see of a linked record through its view. It is
// how an administrator previews a screen as a role, and it never reads a
// record.
type RoleAccess struct {
	registry *permission.Registry
	grants   permission.RoleGrants
}

func NewRoleAccess(registry *permission.Registry, grants permission.RoleGrants) RoleAccess {
	if registry == nil {
		registry = fieldsensitivity.DefaultRegistry()
	}
	return RoleAccess{registry: registry, grants: grants}
}

func (a RoleAccess) Reads(_ context.Context, kind permission.Resource) bool {
	return a.grants.Allows(a.registry, kind.String(), permission.OpRead)
}

// Allows reports whether the role may perform an operation on a resource.
func (a RoleAccess) Allows(resource string, op permission.Operation) bool {
	return a.grants.Allows(a.registry, resource, op)
}

func (a RoleAccess) Ceiling(
	ctx context.Context,
	kind permission.Resource,
	reach Reach,
) (permission.FieldSensitivity, bool) {
	if reach != Nested || a.Reads(ctx, kind) {
		return "", false
	}
	ceiling := fieldsensitivity.RoleCeiling(a.grants.MaxSensitivity)
	if ceiling == "" {
		ceiling = permission.SensitivityInternal
	}

	return ceiling, true
}
