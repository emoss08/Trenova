package trailer

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*trailerPermission)(nil)

type trailerPermission struct{}

func NewTrailerPermission() permissionregistry.PermissionAware {
	return &trailerPermission{}
}

func (c trailerPermission) GetResourceName() string {
	return "trailer"
}

func (c trailerPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new trailers to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View trailer information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify trailer details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove trailers",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export trailers data for operational reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import trailers from other sources",
		),
	}
}

func (c trailerPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport,
		"equipment_manager": permission.OpRead | permission.OpCreate | permission.OpUpdate | permission.OpExport | permission.OpImport,
		"read_only":         permission.OpRead,
	}
}

func (c trailerPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (c trailerPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}
