package fieldsensitivity

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

// AgentDataAccess is the tier an agent definition's data access grants, read
// as a ceiling: Restricted at most, Internal below it, and empty when the
// definition sets none.
func AgentDataAccess(granted permission.FieldSensitivity) permission.FieldSensitivity {
	switch {
	case granted == "":
		return ""
	case granted.CanAccess(permission.SensitivityRestricted):
		return permission.SensitivityRestricted
	default:
		return permission.SensitivityInternal
	}
}

// AgentCeiling is the most sensitive tier an agent's read may be shown on a
// resource. An agent principal reads at its definition's data access,
// Internal when it has none. A person reads at their own role's tier,
// lowered to the agent's data access when an agent is working for them.
func AgentCeiling(
	ctx context.Context,
	engine permissionports.PermissionEngine,
	actor *permissionports.RequestActor,
	granted permission.FieldSensitivity,
	resource permission.Resource,
) permission.FieldSensitivity {
	granted = AgentDataAccess(granted)
	if actor == nil || !actor.IsUser() {
		if granted == "" {
			return permission.SensitivityInternal
		}

		return granted
	}

	person := PersonCeiling(ctx, engine, actor.UserID, actor.OrganizationID, resource)
	if granted != "" && person.Level() > granted.Level() {
		return granted
	}

	return person
}

// AgentCeilings remembers AgentCeiling per resource for the length of one
// read. It is safe for concurrent use by the field resolvers of one
// operation.
type AgentCeilings struct {
	engine  permissionports.PermissionEngine
	actor   *permissionports.RequestActor
	granted permission.FieldSensitivity

	mu    sync.Mutex
	cache map[permission.Resource]permission.FieldSensitivity
}

func NewAgentCeilings(
	engine permissionports.PermissionEngine,
	actor *permissionports.RequestActor,
	granted permission.FieldSensitivity,
) *AgentCeilings {
	return &AgentCeilings{
		engine:  engine,
		actor:   actor,
		granted: granted,
		cache:   make(map[permission.Resource]permission.FieldSensitivity, 4),
	}
}

func (c *AgentCeilings) For(
	ctx context.Context,
	resource permission.Resource,
) permission.FieldSensitivity {
	c.mu.Lock()
	ceiling, seen := c.cache[resource]
	c.mu.Unlock()
	if seen {
		return ceiling
	}

	ceiling = AgentCeiling(ctx, c.engine, c.actor, c.granted, resource)

	c.mu.Lock()
	c.cache[resource] = ceiling
	c.mu.Unlock()

	return ceiling
}
