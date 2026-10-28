package permissioncatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func withImplications(
	defs []permissionspec.OperationDefinition,
) []permissionspec.OperationDefinition {
	result := make([]permissionspec.OperationDefinition, len(defs))
	for i, def := range defs {
		def.Implies = permission.ImpliedByOperation(def.Operation)
		result[i] = def
	}
	return result
}

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

func GetAllOperations() []permissionspec.OperationDefinition {
	return withImplications(permissionspec.Operations(listedOperations...))
}
