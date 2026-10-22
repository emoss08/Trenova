package permission

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

func (e *engine) resourceDetail(
	perms *permissionports.CachedPermissions,
	resource string,
) *permissionports.ResourcePermissionDetail {
	own := perms.Resources[resource]
	var parent *permissionports.CachedResourcePermission
	if effective := e.registry.GetEffectiveResource(resource); effective != resource {
		parent = perms.Resources[effective]
	}

	ops := make([]permission.Operation, 0, 8)
	if own != nil {
		for _, op := range own.Operations {
			ops = append(ops, permission.Operation(op))
		}
	}
	if parent != nil {
		declared := e.registry.GetOperationsForResource(resource)
		for _, op := range parent.Operations {
			implied := permission.Operation(op)
			if slices.Contains(declared, implied) && !slices.Contains(ops, implied) {
				ops = append(ops, implied)
			}
		}
	}

	return &permissionports.ResourcePermissionDetail{
		Resource:       resource,
		Operations:     permission.Normalize(ops),
		DataScope:      detailScope(own, parent),
		MaxSensitivity: permission.FieldSensitivity(perms.MaxSensitivity),
	}
}

func detailScope(own, parent *permissionports.CachedResourcePermission) permission.DataScope {
	read := string(permission.OpRead)
	switch {
	case own != nil && slices.Contains(own.Operations, read):
		return permission.DataScope(own.DataScope)
	case parent != nil && slices.Contains(parent.Operations, read):
		return permission.DataScope(parent.DataScope)
	case own != nil:
		return permission.DataScope(own.DataScope)
	case parent != nil:
		return permission.DataScope(parent.DataScope)
	default:
		return ""
	}
}
