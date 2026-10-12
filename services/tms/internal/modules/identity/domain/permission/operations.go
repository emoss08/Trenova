package permission

type Operation string

const (
	OpRead   Operation = "read"
	OpCreate Operation = "create"
	OpUpdate Operation = "update"
	OpDelete Operation = "delete"
	OpExport Operation = "export"
	OpImport Operation = "import"
)

const (
	OpApprove   Operation = "approve"
	OpReject    Operation = "reject"
	OpAssign    Operation = "assign"
	OpUnassign  Operation = "unassign"
	OpArchive   Operation = "archive"
	OpRestore   Operation = "restore"
	OpSubmit    Operation = "submit"
	OpCancel    Operation = "cancel"
	OpDuplicate Operation = "duplicate"
	OpClose     Operation = "close"
	OpLock      Operation = "lock"
	OpUnlock    Operation = "unlock"
	OpActivate  Operation = "activate"
	OpReopen    Operation = "reopen"
	OpPin       Operation = "pin"
	OpUnpin     Operation = "unpin"
	OpResolve   Operation = "resolve"
	OpManage    Operation = "manage"
)

var Dependencies = map[Operation][]Operation{
	OpRead:      {},
	OpCreate:    {OpRead},
	OpUpdate:    {OpRead},
	OpDelete:    {OpRead},
	OpExport:    {OpRead},
	OpImport:    {OpRead, OpCreate},
	OpApprove:   {OpRead, OpUpdate},
	OpReject:    {OpRead, OpUpdate},
	OpAssign:    {OpRead, OpUpdate},
	OpUnassign:  {OpRead, OpUpdate},
	OpArchive:   {OpRead, OpUpdate},
	OpRestore:   {OpRead, OpUpdate},
	OpSubmit:    {OpRead, OpUpdate},
	OpCancel:    {OpRead, OpUpdate},
	OpDuplicate: {OpRead, OpCreate},
	OpClose:     {OpRead, OpUpdate},
	OpLock:      {OpRead, OpUpdate},
	OpUnlock:    {OpRead, OpUpdate},
	OpActivate:  {OpRead, OpUpdate},
	OpReopen:    {OpRead, OpUpdate},
	OpPin:       {OpRead},
	OpUnpin:     {OpRead},
	OpResolve:   {OpRead},
	OpManage:    {OpRead, OpUpdate, OpDelete},
}

var Dependents = computeDependents()

func computeDependents() map[Operation][]Operation {
	result := make(map[Operation][]Operation)
	for op, deps := range Dependencies {
		for _, dep := range deps {
			result[dep] = append(result[dep], op)
		}
	}
	return result
}

type OperationSet map[Operation]bool

func NewOperationSet(ops ...Operation) OperationSet {
	set := make(OperationSet)
	for _, op := range ops {
		set[op] = true
	}
	return set
}

func (s OperationSet) Has(op Operation) bool {
	return s[op]
}

func (s OperationSet) Add(ops ...Operation) {
	for _, op := range ops {
		s[op] = true
	}
}

func (s OperationSet) Remove(op Operation) {
	delete(s, op)
}

func (s OperationSet) ToSlice() []Operation {
	result := make([]Operation, 0, len(s))
	for op := range s {
		result = append(result, op)
	}
	return result
}

func (s OperationSet) Clone() OperationSet {
	clone := make(OperationSet, len(s))
	for op := range s {
		clone[op] = true
	}
	return clone
}

func ExpandWithDependencies(ops OperationSet) OperationSet {
	expanded := ops.Clone()
	for op := range ops {
		for _, dep := range Dependencies[op] {
			expanded[dep] = true
		}
	}
	return expanded
}

func CollapseOnRevoke(ops OperationSet, revoked Operation) OperationSet {
	result := ops.Clone()
	result.Remove(revoked)

	var removeDependents func(op Operation)
	removeDependents = func(op Operation) {
		for _, dependent := range Dependents[op] {
			if result.Has(dependent) {
				result.Remove(dependent)
				removeDependents(dependent)
			}
		}
	}
	removeDependents(revoked)

	return result
}

const (
	ClientOpRead   uint32 = 1 << 0
	ClientOpCreate uint32 = 1 << 1
	ClientOpUpdate uint32 = 1 << 2
	ClientOpDelete uint32 = 1 << 3
	ClientOpExport uint32 = 1 << 4
	ClientOpImport uint32 = 1 << 5

	ClientOpApprove   uint32 = 1 << 8
	ClientOpReject    uint32 = 1 << 9
	ClientOpAssign    uint32 = 1 << 10
	ClientOpUnassign  uint32 = 1 << 11
	ClientOpArchive   uint32 = 1 << 12
	ClientOpRestore   uint32 = 1 << 13
	ClientOpSubmit    uint32 = 1 << 14
	ClientOpCancel    uint32 = 1 << 15
	ClientOpDuplicate uint32 = 1 << 16
	ClientOpClose     uint32 = 1 << 17
	ClientOpLock      uint32 = 1 << 18
	ClientOpUnlock    uint32 = 1 << 19
	ClientOpActivate  uint32 = 1 << 20
	ClientOpReopen    uint32 = 1 << 21
	ClientOpPin       uint32 = 1 << 22
	ClientOpUnpin     uint32 = 1 << 23
	ClientOpResolve   uint32 = 1 << 24
	ClientOpManage    uint32 = 1 << 25
)

var OperationToBit = map[Operation]uint32{
	OpRead:      ClientOpRead,
	OpCreate:    ClientOpCreate,
	OpUpdate:    ClientOpUpdate,
	OpDelete:    ClientOpDelete,
	OpExport:    ClientOpExport,
	OpImport:    ClientOpImport,
	OpApprove:   ClientOpApprove,
	OpReject:    ClientOpReject,
	OpAssign:    ClientOpAssign,
	OpUnassign:  ClientOpUnassign,
	OpArchive:   ClientOpArchive,
	OpRestore:   ClientOpRestore,
	OpSubmit:    ClientOpSubmit,
	OpCancel:    ClientOpCancel,
	OpDuplicate: ClientOpDuplicate,
	OpClose:     ClientOpClose,
	OpLock:      ClientOpLock,
	OpUnlock:    ClientOpUnlock,
	OpActivate:  ClientOpActivate,
	OpReopen:    ClientOpReopen,
	OpPin:       ClientOpPin,
	OpUnpin:     ClientOpUnpin,
	OpResolve:   ClientOpResolve,
	OpManage:    ClientOpManage,
}

func OperationsToBitmask(ops []Operation) uint32 {
	var bits uint32
	for _, op := range ops {
		if bit, ok := OperationToBit[op]; ok {
			bits |= bit
		}
	}
	return bits
}

// IsValid reports whether the operation is one the system defines. Dependencies
// is keyed by every operation, so it is the list rather than a copy of it that
// would drift.
func (o Operation) IsValid() bool {
	_, ok := Dependencies[o]
	return ok
}

// operationDefinitions is the one place an operation's display name and
// description are written down. The per-resource tables, the standard
// operation sets, and the list the API hands the UI are all read out of it, so
// a name cannot come to differ between the registry and the screen that shows
// it.
var operationDefinitions = map[Operation]OperationDefinition{
	OpRead:     {Operation: OpRead, DisplayName: "Read", Description: "View records"},
	OpCreate:   {Operation: OpCreate, DisplayName: "Create", Description: "Create new records"},
	OpUpdate:   {Operation: OpUpdate, DisplayName: "Update", Description: "Modify existing records"},
	OpDelete:   {Operation: OpDelete, DisplayName: "Delete", Description: "Delete records"},
	OpExport:   {Operation: OpExport, DisplayName: "Export", Description: "Export records to file"},
	OpImport:   {Operation: OpImport, DisplayName: "Import", Description: "Import records from file"},
	OpApprove:  {Operation: OpApprove, DisplayName: "Approve", Description: "Approve records"},
	OpReject:   {Operation: OpReject, DisplayName: "Reject", Description: "Reject records"},
	OpAssign:   {Operation: OpAssign, DisplayName: "Assign", Description: "Assign to users"},
	OpUnassign: {Operation: OpUnassign, DisplayName: "Unassign", Description: "Remove assignments"},
	OpArchive:  {Operation: OpArchive, DisplayName: "Archive", Description: "Archive records"},
	OpRestore: {
		Operation:   OpRestore,
		DisplayName: "Restore",
		Description: "Restore archived records",
	},
	OpSubmit: {Operation: OpSubmit, DisplayName: "Submit", Description: "Submit for processing"},
	OpCancel: {Operation: OpCancel, DisplayName: "Cancel", Description: "Cancel records"},
	OpDuplicate: {
		Operation:   OpDuplicate,
		DisplayName: "Duplicate",
		Description: "Create copies",
	},
	OpClose:    {Operation: OpClose, DisplayName: "Close", Description: "Close records"},
	OpLock:     {Operation: OpLock, DisplayName: "Lock", Description: "Lock records"},
	OpUnlock:   {Operation: OpUnlock, DisplayName: "Unlock", Description: "Unlock records"},
	OpActivate: {Operation: OpActivate, DisplayName: "Activate", Description: "Activate records"},
	OpReopen:   {Operation: OpReopen, DisplayName: "Reopen", Description: "Reopen records"},
	OpPin:      {Operation: OpPin, DisplayName: "Pin", Description: "Pin records"},
	OpUnpin:    {Operation: OpUnpin, DisplayName: "Unpin", Description: "Unpin records"},
	OpResolve:  {Operation: OpResolve, DisplayName: "Resolve", Description: "Resolve records"},
	OpManage:   {Operation: OpManage, DisplayName: "Manage", Description: "Manage records"},
}

// operations builds a definition slice in the order asked for. It allocates a
// fresh slice each time, so a caller may append to the result without writing
// through to the shared table.
func operations(ops ...Operation) []OperationDefinition {
	result := make([]OperationDefinition, 0, len(ops))
	for _, op := range ops {
		result = append(result, operationDefinitions[op])
	}

	return result
}

var (
	standardOps = operations(OpRead, OpCreate, OpUpdate, OpExport, OpImport)

	readOnlyOps = operations(OpRead)

	standardOpsWithDelete = operations(
		OpRead, OpCreate, OpUpdate, OpExport, OpImport, OpDelete,
	)
)

// listedOperations is the set the permissions screen offers a person to pick
// from. It is narrower than IsValid: delete is reachable only through a
// resource that declares it, and the agent-facing operations are granted
// through role grants rather than chosen by hand.
var listedOperations = []Operation{
	OpRead, OpCreate, OpUpdate, OpExport, OpImport,
	OpApprove, OpReject, OpAssign, OpUnassign,
	OpArchive, OpRestore, OpSubmit, OpCancel, OpDuplicate,
	OpClose, OpLock, OpUnlock, OpActivate, OpReopen,
}

func GetAllOperations() []OperationDefinition {
	return operations(listedOperations...)
}
