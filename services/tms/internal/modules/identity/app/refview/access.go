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

	ceiling = fieldsensitivity.PersonCeiling(
		ctx, a.engine, a.actor.UserID, a.actor.OrganizationID, kind,
	)
	if ceiling == "" {
		ceiling = permission.SensitivityInternal
	}

	a.mu.Lock()
	a.ceilings[kind] = ceiling
	a.mu.Unlock()

	return ceiling, true
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
