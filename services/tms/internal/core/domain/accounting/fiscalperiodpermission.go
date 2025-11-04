package accounting

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*fiscalPeriodPermission)(nil)

type fiscalPeriodPermission struct{}

func NewFiscalPeriodPermission() permissionregistry.PermissionAware {
	return &fiscalPeriodPermission{}
}

func (c fiscalPeriodPermission) GetResourceName() string {
	return "fiscal_period"
}

func (c fiscalPeriodPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new fiscal periods to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View fiscal period information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify fiscal period details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export fiscal periods data for compliance reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import fiscal periods from other sources",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpClose,
			"close",
			"Close",
			"Close fiscal period",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpLock,
			"lock",
			"Lock",
			"Lock fiscal period",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUnlock,
			"unlock",
			"Unlock",
			"Unlock fiscal period",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpActivate,
			"activate",
			"Activate",
			"Activate fiscal period",
		),
	}
}

func (c fiscalPeriodPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpExport | permission.OpImport | permission.OpClose |
			permission.OpLock | permission.OpUnlock | permission.OpActivate,
		"read_only": permission.OpRead,
	}
}

func (c fiscalPeriodPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (c fiscalPeriodPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
	}
}
