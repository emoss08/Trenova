package commodity

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*commodityPermission)(nil)

type commodityPermission struct{}

func NewCommodityPermission() permissionregistry.PermissionAware {
	return &commodityPermission{}
}

func (c commodityPermission) GetResourceName() string {
	return "commodity"
}

func (c commodityPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new commodities to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View commodity information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify commodity details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove commodities (requires approval due to safety regulations)",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export commodities data for compliance reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import commodities from UN database or other sources",
		),
	}
}

func (c commodityPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport,
		"safety_officer": permission.OpRead | permission.OpCreate | permission.OpUpdate | permission.OpExport,
		"compliance":     permission.OpRead | permission.OpExport,
		"read_only":      permission.OpRead,
	}
}

func (c commodityPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (c commodityPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}
