package permission

import "slices"

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
	OpPost      Operation = "post"
	// OpApproveOwn lets a reviewer approve a revision they submitted or authored, where the
	// two-person rule would otherwise send it to someone else.
	OpApproveOwn Operation = "approve_own"
)

var Dependencies = map[Operation][]Operation{
	OpRead:       {},
	OpCreate:     {OpRead},
	OpUpdate:     {OpRead},
	OpDelete:     {OpRead},
	OpExport:     {OpRead},
	OpImport:     {OpRead, OpCreate},
	OpApprove:    {OpRead, OpUpdate},
	OpReject:     {OpRead, OpUpdate},
	OpAssign:     {OpRead, OpUpdate},
	OpUnassign:   {OpRead, OpUpdate},
	OpArchive:    {OpRead, OpUpdate},
	OpRestore:    {OpRead, OpUpdate},
	OpSubmit:     {OpRead, OpUpdate},
	OpCancel:     {OpRead, OpUpdate},
	OpDuplicate:  {OpRead, OpCreate},
	OpClose:      {OpRead, OpUpdate},
	OpLock:       {OpRead, OpUpdate},
	OpUnlock:     {OpRead, OpUpdate},
	OpActivate:   {OpRead, OpUpdate},
	OpReopen:     {OpRead, OpUpdate},
	OpPin:        {OpRead},
	OpUnpin:      {OpRead},
	OpResolve:    {OpRead},
	OpManage:     {OpRead, OpUpdate, OpDelete},
	OpPost:       {OpRead},
	OpApproveOwn: {OpRead, OpUpdate, OpApprove},
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
	pending := ops.ToSlice()
	for len(pending) > 0 {
		op := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		for _, dep := range Dependencies[op] {
			if !expanded[dep] {
				expanded[dep] = true
				pending = append(pending, dep)
			}
		}
	}
	return expanded
}

func Normalize(ops []Operation) []Operation {
	normalized := NewOperationSet(ops...).ToSlice()
	slices.Sort(normalized)
	return normalized
}

func Implied(ops []Operation) []Operation {
	return Normalize(ExpandWithDependencies(NewOperationSet(ops...)).ToSlice())
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

// IsValid reports whether the operation is one the system defines. Dependencies
// is keyed by every operation, so it is the list rather than a copy of it that
// would drift.
func (o Operation) IsValid() bool {
	_, ok := Dependencies[o]
	return ok
}

func GrantingOperations(op Operation) []Operation {
	granting := []Operation{op}
	for candidate := range Dependencies {
		if candidate != op && slices.Contains(ImpliedByOperation(candidate), op) {
			granting = append(granting, candidate)
		}
	}
	slices.Sort(granting)
	return granting
}

func ImpliedByOperation(op Operation) []Operation {
	implied := Implied([]Operation{op})
	return slices.DeleteFunc(implied, func(dep Operation) bool { return dep == op })
}
