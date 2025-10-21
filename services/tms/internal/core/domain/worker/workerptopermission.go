package worker

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*workerPTOPermission)(nil)

type workerPTOPermission struct{}

func NewWorkerPTOPermission() permissionregistry.PermissionAware {
	return &workerPTOPermission{}
}

func (h *workerPTOPermission) GetResourceName() string {
	return "worker_pto"
}

func (h *workerPTOPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new worker PTOs to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View worker PTO information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpApprove,
			"approve",
			"Approve",
			"Approve worker PTOs",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpReject,
			"reject",
			"Reject",
			"Reject worker PTOs",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify worker PTO details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove worker PTOs",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export worker PTOs data for operational reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import worker PTOs from other sources",
		),
	}
}

func (h *workerPTOPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport | permission.OpApprove | permission.OpReject,
		"read_only": permission.OpRead,
	}
}

func (h *workerPTOPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (h *workerPTOPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
		permission.OpReject,
		permission.OpApprove,
	}
}
