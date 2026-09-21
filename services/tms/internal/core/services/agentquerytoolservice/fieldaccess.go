package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// fieldAccess answers which of a resource's fields a person may be shown,
// by the same sensitivity model the report compiler applies, so the worker
// a model reads through get_worker is the worker the person could open.
//
// Confidential fields are never sent to a model, whatever the person's
// role: a date of birth or a licence number in a chat transcript is a
// leak with nobody to blame. Restricted fields follow the person's role
// ceiling; Internal ones are shown to anyone who may read the record.
type fieldAccess struct {
	permissions serviceports.PermissionEngine
	registry    *permission.Registry
}

func newFieldAccess(permissions serviceports.PermissionEngine) fieldAccess {
	return fieldAccess{permissions: permissions, registry: permission.NewRegistry()}
}

// ceiling is the most sensitive tier the actor may be shown on a resource.
// An agent principal, or an authorization that cannot be resolved, reads at
// the Internal tier: enough for the record, none of the person behind it.
func (a fieldAccess) ceiling(
	ctx context.Context,
	params serviceports.QueryToolParams,
	resource permission.Resource,
) permission.FieldSensitivity {
	if a.permissions == nil || params.Actor == nil || !params.Actor.IsUser() {
		return permission.SensitivityInternal
	}

	detail, err := a.permissions.GetResourcePermissions(
		ctx, params.Actor.UserID, params.OrganizationID, resource.String(),
	)
	if err != nil || detail == nil {
		return permission.SensitivityInternal
	}
	if detail.MaxSensitivity.CanAccess(permission.SensitivityConfidential) {
		// Nothing above Restricted reaches a model, however far the role goes.
		return permission.SensitivityRestricted
	}

	return detail.MaxSensitivity
}

// visible reports whether one field of a resource may be shown under a
// ceiling. Confidential is refused outright.
func (a fieldAccess) visible(
	resource permission.Resource,
	field string,
	ceiling permission.FieldSensitivity,
) bool {
	sensitivity := a.registry.GetFieldSensitivity(resource.String(), field)
	if sensitivity == permission.SensitivityConfidential {
		return false
	}

	return ceiling.CanAccess(sensitivity)
}

// mayRead reports whether the actor may read a resource other than the one
// the tool is gated on, for a row that joins two: a tractor's position
// carries its driver's name, and the name is the worker's to show. The
// question goes to the engine as the tool guard's own would, so an agent
// principal, a person and an API key are each answered by their own rules.
func (a fieldAccess) mayRead(
	ctx context.Context,
	params serviceports.QueryToolParams,
	resource permission.Resource,
) bool {
	actor := params.Actor
	if actor == nil || a.permissions == nil {
		return false
	}

	result, err := a.permissions.Check(ctx, &serviceports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       resource.String(),
		Operation:      permission.OpRead,
	})

	return err == nil && result != nil && result.Allowed
}
