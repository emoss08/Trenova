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
