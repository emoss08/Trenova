package fieldsensitivity

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

// PersonCeiling is the most sensitive tier a person may be shown on a
// resource: their role's reach, never above Restricted, and Internal when the
// authorization cannot be resolved.
func PersonCeiling(
	ctx context.Context,
	engine serviceports.PermissionEngine,
	userID, organizationID pulid.ID,
	resource permission.Resource,
) permission.FieldSensitivity {
	if engine == nil {
		return permission.SensitivityInternal
	}

	detail, err := engine.GetResourcePermissions(ctx, userID, organizationID, resource.String())
	if err != nil || detail == nil {
		return permission.SensitivityInternal
	}
	if detail.MaxSensitivity.CanAccess(permission.SensitivityConfidential) {
		return permission.SensitivityRestricted
	}

	return detail.MaxSensitivity
}

var defaultRegistry = sync.OnceValue(permission.NewRegistry)

// DefaultRegistry is the resource registry built once for the process, for
// code that classifies fields without a registry of its own to hand.
func DefaultRegistry() *permission.Registry {
	return defaultRegistry()
}

// Level is the sensitivity the registry gives one field of a resource.
func Level(
	registry *permission.Registry,
	resource permission.Resource,
	field string,
) permission.FieldSensitivity {
	if registry == nil {
		return permission.SensitivityInternal
	}

	return registry.GetFieldSensitivity(resource.String(), field)
}

// Visible reports whether one field of a resource may be shown under a
// ceiling. Confidential is refused outright.
func Visible(
	registry *permission.Registry,
	resource permission.Resource,
	field string,
	ceiling permission.FieldSensitivity,
) bool {
	return VisibleAt(Level(registry, resource, field), ceiling)
}

// VisibleAt reports whether a value recorded at a sensitivity may be shown
// under a ceiling.
func VisibleAt(level, ceiling permission.FieldSensitivity) bool {
	if level == permission.SensitivityConfidential {
		return false
	}

	return ceiling.CanAccess(level)
}

// Ceilings remembers one person's ceiling per resource for the length of a
// request, so a page of rows touching the same resource asks the engine once.
// It is safe for concurrent use by the field resolvers of one request.
type Ceilings struct {
	engine         serviceports.PermissionEngine
	userID         pulid.ID
	organizationID pulid.ID

	mu    sync.Mutex
	cache map[permission.Resource]permission.FieldSensitivity
}

func NewCeilings(
	engine serviceports.PermissionEngine,
	userID, organizationID pulid.ID,
) *Ceilings {
	return &Ceilings{
		engine:         engine,
		userID:         userID,
		organizationID: organizationID,
		cache:          make(map[permission.Resource]permission.FieldSensitivity, 4),
	}
}

type permissionChecker interface {
	Check(
		ctx context.Context,
		req *serviceports.PermissionCheckRequest,
	) (*serviceports.PermissionCheckResult, error)
}

// ReadAccess remembers whether one reader may read each resource for the
// length of a request. A check that cannot be made is a no, and is not
// remembered, so a transient failure is asked again. It is safe for
// concurrent use.
type ReadAccess struct {
	engine permissionChecker
	actor  *serviceports.RequestActor

	mu    sync.Mutex
	cache map[permission.Resource]bool
}

func NewReadAccess(engine permissionChecker, actor *serviceports.RequestActor) *ReadAccess {
	return &ReadAccess{
		engine: engine,
		actor:  actor,
		cache:  make(map[permission.Resource]bool, 4),
	}
}

func (r *ReadAccess) MayRead(ctx context.Context, resource permission.Resource) bool {
	if r == nil || r.engine == nil || r.actor == nil || resource == "" {
		return false
	}

	r.mu.Lock()
	allowed, seen := r.cache[resource]
	r.mu.Unlock()
	if seen {
		return allowed
	}

	result, err := r.engine.Check(ctx, r.actor.PermissionCheck(resource, permission.OpRead))
	if err != nil || result == nil {
		return false
	}

	r.mu.Lock()
	r.cache[resource] = result.Allowed
	r.mu.Unlock()

	return result.Allowed
}

func (c *Ceilings) For(
	ctx context.Context,
	resource permission.Resource,
) permission.FieldSensitivity {
	c.mu.Lock()
	ceiling, seen := c.cache[resource]
	c.mu.Unlock()
	if seen {
		return ceiling
	}

	ceiling = PersonCeiling(ctx, c.engine, c.userID, c.organizationID, resource)

	c.mu.Lock()
	c.cache[resource] = ceiling
	c.mu.Unlock()

	return ceiling
}
