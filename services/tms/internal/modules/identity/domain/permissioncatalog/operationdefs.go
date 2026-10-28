package permissioncatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

// operationDefinitions is the one place an operation's display name and
// description are written down. The per-resource tables, the standard
// operation sets, and the list the API hands the UI are all read out of it, so
// a name cannot come to differ between the registry and the screen that shows
// it.
var operationDefinitions = map[permission.Operation]OperationDefinition{
	permission.OpRead: {
		Operation:   permission.OpRead,
		DisplayName: "View",
		Description: "View records",
	},
	permission.OpCreate: {
		Operation:   permission.OpCreate,
		DisplayName: "Create",
		Description: "Create new records",
	},
	permission.OpUpdate: {
		Operation:   permission.OpUpdate,
		DisplayName: "Update",
		Description: "Modify existing records",
	},
	permission.OpDelete: {
		Operation:   permission.OpDelete,
		DisplayName: "Delete",
		Description: "Delete records",
	},
	permission.OpExport: {
		Operation:   permission.OpExport,
		DisplayName: "Export",
		Description: "Export records to file",
	},
	permission.OpImport: {
		Operation:   permission.OpImport,
		DisplayName: "Import",
		Description: "Import records from file",
	},
	permission.OpApprove: {
		Operation:   permission.OpApprove,
		DisplayName: "Approve",
		Description: "Approve records",
	},
	permission.OpReject: {
		Operation:   permission.OpReject,
		DisplayName: "Reject",
		Description: "Reject records",
	},
	permission.OpAssign: {
		Operation:   permission.OpAssign,
		DisplayName: "Assign",
		Description: "Assign to users",
	},
	permission.OpUnassign: {
		Operation:   permission.OpUnassign,
		DisplayName: "Unassign",
		Description: "Remove assignments",
	},
	permission.OpArchive: {
		Operation:   permission.OpArchive,
		DisplayName: "Archive",
		Description: "Archive records",
	},
	permission.OpRestore: {
		Operation:   permission.OpRestore,
		DisplayName: "Restore",
		Description: "Restore archived records",
	},
	permission.OpSubmit: {
		Operation:   permission.OpSubmit,
		DisplayName: "Submit",
		Description: "Submit for processing",
	},
	permission.OpCancel: {
		Operation:   permission.OpCancel,
		DisplayName: "Cancel",
		Description: "Cancel records",
	},
	permission.OpDuplicate: {
		Operation:   permission.OpDuplicate,
		DisplayName: "Duplicate",
		Description: "Create copies",
	},
	permission.OpClose: {
		Operation:   permission.OpClose,
		DisplayName: "Close",
		Description: "Close records",
	},
	permission.OpLock: {
		Operation:   permission.OpLock,
		DisplayName: "Lock",
		Description: "Lock records",
	},
	permission.OpUnlock: {
		Operation:   permission.OpUnlock,
		DisplayName: "Unlock",
		Description: "Unlock records",
	},
	permission.OpActivate: {
		Operation:   permission.OpActivate,
		DisplayName: "Activate",
		Description: "Activate records",
	},
	permission.OpReopen: {
		Operation:   permission.OpReopen,
		DisplayName: "Reopen",
		Description: "Reopen records",
	},
	permission.OpPin: {
		Operation:   permission.OpPin,
		DisplayName: "Pin",
		Description: "Pin records",
	},
	permission.OpUnpin: {
		Operation:   permission.OpUnpin,
		DisplayName: "Unpin",
		Description: "Unpin records",
	},
	permission.OpResolve: {
		Operation:   permission.OpResolve,
		DisplayName: "Resolve",
		Description: "Resolve records",
	},
	permission.OpManage: {
		Operation:   permission.OpManage,
		DisplayName: "Manage",
		Description: "Manage records",
	},
	permission.OpPost: {
		Operation:   permission.OpPost,
		DisplayName: "Post",
		Description: "Post records to the ledger",
	},
	permission.OpApproveOwn: {
		Operation:   permission.OpApproveOwn,
		DisplayName: "Approve own changes",
		Description: "Approve a revision you submitted or edited yourself, without a second reviewer",
	},
}

// operations builds a definition slice in the order asked for. It allocates a
// fresh slice each time, so a caller may append to the result without writing
// through to the shared table.
func operations(ops ...permission.Operation) []OperationDefinition {
	result := make([]OperationDefinition, 0, len(ops))
	for _, op := range ops {
		result = append(result, operationDefinitions[op])
	}

	return result
}

func withImplications(defs []OperationDefinition) []OperationDefinition {
	result := make([]OperationDefinition, len(defs))
	for i, def := range defs {
		def.Implies = permission.ImpliedByOperation(def.Operation)
		result[i] = def
	}
	return result
}

var (
	standardOps = operations(
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpExport,
		permission.OpImport,
	)

	readOnlyOps = operations(permission.OpRead)

	standardOpsWithDelete = operations(
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpExport,
		permission.OpImport,
		permission.OpDelete,
	)
)

// listedOperations is the set the permissions screen offers a person to pick
// from. It is narrower than IsValid: delete is reachable only through a
// resource that declares it, and the agent-facing operations are granted
// through role grants rather than chosen by hand.
var listedOperations = []permission.Operation{
	permission.OpRead, permission.OpCreate, permission.OpUpdate, permission.OpExport, permission.OpImport,
	permission.OpApprove, permission.OpReject, permission.OpAssign, permission.OpUnassign,
	permission.OpArchive, permission.OpRestore, permission.OpSubmit, permission.OpCancel, permission.OpDuplicate,
	permission.OpClose, permission.OpLock, permission.OpUnlock, permission.OpActivate, permission.OpReopen,
}

func GetAllOperations() []OperationDefinition {
	return withImplications(operations(listedOperations...))
}

func relabel(
	defs []OperationDefinition,
	op permission.Operation,
	displayName, description string,
) []OperationDefinition {
	result := make([]OperationDefinition, len(defs))
	copy(result, defs)
	for i := range result {
		if result[i].Operation == op {
			result[i].DisplayName = displayName
			result[i].Description = description
		}
	}
	return result
}
