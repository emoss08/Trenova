package accounting

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*fiscalYearPermission)(nil)

type fiscalYearPermission struct{}

func NewFiscalYearPermission() permissionregistry.PermissionAware {
	return &fiscalYearPermission{}
}

func (c fiscalYearPermission) GetResourceName() string {
	return "fiscal_year"
}

func (c fiscalYearPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new fiscal years to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View fiscal year information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify fiscal year details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export fiscal years data for compliance reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import fiscal years from other sources",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpClose,
			"close",
			"Close",
			"Close fiscal year",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpLock,
			"lock",
			"Lock",
			"Lock fiscal year",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUnlock,
			"unlock",
			"Unlock",
			"Unlock fiscal year",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpActivate,
			"activate",
			"Activate",
			"Activate fiscal year",
		),
	}
}

func (c fiscalYearPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpExport | permission.OpImport | permission.OpClose |
			permission.OpLock | permission.OpUnlock | permission.OpActivate,
		"read_only": permission.OpRead,
	}
}

func (c fiscalYearPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (c fiscalYearPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
	}
}
