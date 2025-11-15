package docker

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*dockerPermission)(nil)

type dockerPermission struct{}

func NewDockerPermission() permissionregistry.PermissionAware {
	return &dockerPermission{}
}

func (c dockerPermission) GetResourceName() string {
	return "docker"
}

func (c dockerPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View docker information",
		),
	}
}

func (c dockerPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpRead,
	}
}

func (c dockerPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (c dockerPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpRead,
	}
}
