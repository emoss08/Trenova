package accounting

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*accountTypePermission)(nil)

type accountTypePermission struct{}

func NewAccountTypePermission() permissionregistry.PermissionAware {
	return &accountTypePermission{}
}

func (c accountTypePermission) GetResourceName() string {
	return "account_type"
}

func (c accountTypePermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new account types to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View account type information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify account type details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export account types data for compliance reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import account types from other sources",
		),
	}
}

func (c accountTypePermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpExport | permission.OpImport,
		"read_only": permission.OpRead,
	}
}

func (c accountTypePermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (c accountTypePermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
	}
}
