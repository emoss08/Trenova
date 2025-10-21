package permissionregistry

import "github.com/emoss08/trenova/internal/core/domain/permission"

type PermissionAware interface {
	GetResourceName() string
	GetSupportedOperations() []OperationDefinition
	GetCompositeOperations() map[string]permission.Operation
	GetDefaultOperation() permission.Operation
	GetOperationsRequiringApproval() []permission.Operation
}

type OperationDefinition struct {
	Code        permission.Operation
	Name        string
	DisplayName string
	Description string
}

func BuildOperationDefinition(
	code permission.Operation,
	name, displayName, description string,
) OperationDefinition {
	return OperationDefinition{
		Code:        code,
		Name:        name,
		DisplayName: displayName,
		Description: description,
	}
}
