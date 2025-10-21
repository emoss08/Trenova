package permissionregistry

import "github.com/emoss08/trenova/internal/core/domain/permission"

var (
	CompositeManageFull = permission.OpCreate | permission.OpRead | permission.OpUpdate | permission.OpDelete |
		permission.OpExport | permission.OpImport | permission.OpArchive | permission.OpRestore | permission.OpDuplicate
	CompositeBasicCRUD = permission.OpCreate | permission.OpRead | permission.OpUpdate | permission.OpDelete
	CompositeReadOnly  = permission.OpRead | permission.OpExport
	CompositeWorkflow  = permission.OpApprove | permission.OpReject | permission.OpSubmit
)

func StandardCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage":     CompositeManageFull,
		"basic_crud": CompositeBasicCRUD,
		"read_only":  CompositeReadOnly,
		"workflow":   CompositeWorkflow,
	}
}

func StandardCRUDOperations() []OperationDefinition {
	return []OperationDefinition{
		BuildOperationDefinition(permission.OpCreate, "create", "Create", "Create new records"),
		BuildOperationDefinition(permission.OpRead, "read", "Read", "View records"),
		BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify existing records",
		),
		BuildOperationDefinition(permission.OpDelete, "delete", "Delete", "Remove records"),
		BuildOperationDefinition(
			permission.OpDuplicate,
			"duplicate",
			"Duplicate",
			"Duplicate records",
		),
	}
}

func StandardExportImportOperations() []OperationDefinition {
	return []OperationDefinition{
		BuildOperationDefinition(permission.OpExport, "export", "Export", "Export data to files"),
		BuildOperationDefinition(permission.OpImport, "import", "Import", "Import data from files"),
	}
}

func StandardArchiveOperations() []OperationDefinition {
	return []OperationDefinition{
		BuildOperationDefinition(permission.OpArchive, "archive", "Archive", "Archive old records"),
		BuildOperationDefinition(
			permission.OpRestore,
			"restore",
			"Restore",
			"Restore archived records",
		),
	}
}

func StandardWorkflowOperations() []OperationDefinition {
	return []OperationDefinition{
		BuildOperationDefinition(
			permission.OpApprove,
			"approve",
			"Approve",
			"Approve workflows or requests",
		),
		BuildOperationDefinition(
			permission.OpReject,
			"reject",
			"Reject",
			"Reject workflows or requests",
		),
		BuildOperationDefinition(permission.OpSubmit, "submit", "Submit", "Submit for processing"),
	}
}

func StandardAssignmentOperations() []OperationDefinition {
	return []OperationDefinition{
		BuildOperationDefinition(
			permission.OpAssign,
			"assign",
			"Assign",
			"Assign to users or resources",
		),
		BuildOperationDefinition(permission.OpShare, "share", "Share", "Share with other users"),
	}
}

func HasOperation(bitfield, operation uint32) bool {
	return (bitfield & operation) == operation
}

func AddOperation(bitfield, operation uint32) uint32 {
	return bitfield | operation
}

func RemoveOperation(bitfield, operation uint32) uint32 {
	return bitfield &^ operation
}

func GetOperationsFromBitfield(bitfield uint32) []string {
	operations := make([]string, 0)
	if HasOperation(bitfield, permission.OpCreate.ToUint32()) {
		operations = append(operations, "create")
	}
	if HasOperation(bitfield, permission.OpRead.ToUint32()) {
		operations = append(operations, "read")
	}
	if HasOperation(bitfield, permission.OpUpdate.ToUint32()) {
		operations = append(operations, "update")
	}
	if HasOperation(bitfield, permission.OpDelete.ToUint32()) {
		operations = append(operations, "delete")
	}
	if HasOperation(bitfield, permission.OpExport.ToUint32()) {
		operations = append(operations, "export")
	}
	if HasOperation(bitfield, permission.OpImport.ToUint32()) {
		operations = append(operations, "import")
	}
	if HasOperation(bitfield, permission.OpApprove.ToUint32()) {
		operations = append(operations, "approve")
	}
	if HasOperation(bitfield, permission.OpReject.ToUint32()) {
		operations = append(operations, "reject")
	}
	if HasOperation(bitfield, permission.OpShare.ToUint32()) {
		operations = append(operations, "share")
	}
	if HasOperation(bitfield, permission.OpArchive.ToUint32()) {
		operations = append(operations, "archive")
	}
	if HasOperation(bitfield, permission.OpRestore.ToUint32()) {
		operations = append(operations, "restore")
	}
	if HasOperation(bitfield, permission.OpSubmit.ToUint32()) {
		operations = append(operations, "submit")
	}
	if HasOperation(bitfield, permission.OpCopy.ToUint32()) {
		operations = append(operations, "copy")
	}
	if HasOperation(bitfield, permission.OpAssign.ToUint32()) {
		operations = append(operations, "assign")
	}
	if HasOperation(bitfield, permission.OpDuplicate.ToUint32()) {
		operations = append(operations, "duplicate")
	}

	return operations
}

func BuildBitfieldFromOperations(operationNames []string) uint32 {
	var bitfield uint32

	for _, name := range operationNames {
		switch name {
		case "create":
			bitfield |= permission.OpCreate.ToUint32()
		case "read":
			bitfield |= permission.OpRead.ToUint32()
		case "update":
			bitfield |= permission.OpUpdate.ToUint32()
		case "delete":
			bitfield |= permission.OpDelete.ToUint32()
		case "export":
			bitfield |= permission.OpExport.ToUint32()
		case "import":
			bitfield |= permission.OpImport.ToUint32()
		case "approve":
			bitfield |= permission.OpApprove.ToUint32()
		case "reject":
			bitfield |= permission.OpReject.ToUint32()
		case "share":
			bitfield |= permission.OpShare.ToUint32()
		case "archive":
			bitfield |= permission.OpArchive.ToUint32()
		case "restore":
			bitfield |= permission.OpRestore.ToUint32()
		case "submit":
			bitfield |= permission.OpSubmit.ToUint32()
		case "copy":
			bitfield |= permission.OpCopy.ToUint32()
		case "assign":
			bitfield |= permission.OpAssign.ToUint32()
		case "duplicate":
			bitfield |= permission.OpDuplicate.ToUint32()
		}
	}

	return bitfield
}
