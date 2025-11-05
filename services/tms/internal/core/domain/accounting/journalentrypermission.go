package accounting

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*journalEntryPermission)(nil)

type journalEntryPermission struct{}

func NewJournalEntryPermission() permissionregistry.PermissionAware {
	return &journalEntryPermission{}
}

func (c journalEntryPermission) GetResourceName() string {
	return "journal_entry"
}

func (c journalEntryPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Create new journal entries",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View journal entry information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify journal entry details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Delete draft journal entries",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export journal entries data",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import journal entries from external sources",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpApprove,
			"approve",
			"Approve",
			"Approve journal entries for posting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpReject,
			"reject",
			"Reject",
			"Reject journal entries",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpSubmit,
			"submit",
			"Submit",
			"Submit journal entries for approval",
		),
	}
}

func (c journalEntryPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport |
			permission.OpApprove | permission.OpReject | permission.OpSubmit,
		"read_only": permission.OpRead,
		"approver":  permission.OpRead | permission.OpApprove | permission.OpReject,
		"creator": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpSubmit,
	}
}

func (c journalEntryPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (c journalEntryPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}
