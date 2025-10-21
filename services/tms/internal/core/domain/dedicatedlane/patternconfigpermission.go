package dedicatedlane

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*patternConfigPermission)(nil)

type patternConfigPermission struct{}

func NewPatternConfigPermission() permissionregistry.PermissionAware {
	return &patternConfigPermission{}
}

func (c patternConfigPermission) GetResourceName() string {
	return "pattern_config"
}

func (c patternConfigPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View pattern config information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify pattern config details",
		),
	}
}

func (c patternConfigPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage":    permission.OpRead | permission.OpUpdate,
		"read_only": permission.OpRead,
	}
}

func (c patternConfigPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (c patternConfigPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpUpdate,
	}
}
