package permissioncatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/operationbits"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOperationSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ops      []permission.Operation
		expected map[permission.Operation]bool
	}{
		{
			name:     "empty set",
			ops:      []permission.Operation{},
			expected: map[permission.Operation]bool{},
		},
		{
			name:     "single operation",
			ops:      []permission.Operation{permission.OpRead},
			expected: map[permission.Operation]bool{permission.OpRead: true},
		},
		{
			name: "multiple operations",
			ops: []permission.Operation{
				permission.OpRead,
				permission.OpCreate,
				permission.OpUpdate,
			},
			expected: map[permission.Operation]bool{
				permission.OpRead:   true,
				permission.OpCreate: true,
				permission.OpUpdate: true,
			},
		},
		{
			name: "duplicate operations",
			ops: []permission.Operation{
				permission.OpRead,
				permission.OpRead,
				permission.OpCreate,
			},
			expected: map[permission.Operation]bool{
				permission.OpRead:   true,
				permission.OpCreate: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			set := permission.NewOperationSet(tt.ops...)
			assert.Equal(t, len(tt.expected), len(set))
			for op := range tt.expected {
				assert.True(t, set.Has(op), "expected operation %s to be in set", op)
			}
		})
	}
}

func TestOperationSet_Has(t *testing.T) {
	t.Parallel()

	set := permission.NewOperationSet(permission.OpRead, permission.OpCreate)

	assert.True(t, set.Has(permission.OpRead))
	assert.True(t, set.Has(permission.OpCreate))
	assert.False(t, set.Has(permission.OpUpdate))
	assert.False(t, set.Has(permission.OpExport))
}

func TestOperationSet_Add(t *testing.T) {
	t.Parallel()

	set := permission.NewOperationSet(permission.OpRead)
	assert.True(t, set.Has(permission.OpRead))
	assert.False(t, set.Has(permission.OpCreate))

	set.Add(permission.OpCreate, permission.OpUpdate)
	assert.True(t, set.Has(permission.OpRead))
	assert.True(t, set.Has(permission.OpCreate))
	assert.True(t, set.Has(permission.OpUpdate))
}

func TestOperationSet_Remove(t *testing.T) {
	t.Parallel()

	set := permission.NewOperationSet(permission.OpRead, permission.OpCreate, permission.OpUpdate)
	assert.True(t, set.Has(permission.OpCreate))

	set.Remove(permission.OpCreate)
	assert.False(t, set.Has(permission.OpCreate))
	assert.True(t, set.Has(permission.OpRead))
	assert.True(t, set.Has(permission.OpUpdate))
}

func TestOperationSet_ToSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ops      []permission.Operation
		expected int
	}{
		{
			name:     "empty set",
			ops:      []permission.Operation{},
			expected: 0,
		},
		{
			name:     "single operation",
			ops:      []permission.Operation{permission.OpRead},
			expected: 1,
		},
		{
			name: "multiple operations",
			ops: []permission.Operation{
				permission.OpRead,
				permission.OpCreate,
				permission.OpUpdate,
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			set := permission.NewOperationSet(tt.ops...)
			slice := set.ToSlice()
			assert.Len(t, slice, tt.expected)

			for _, op := range tt.ops {
				assert.Contains(t, slice, op)
			}
		})
	}
}

func TestOperationSet_Clone(t *testing.T) {
	t.Parallel()

	original := permission.NewOperationSet(permission.OpRead, permission.OpCreate)
	clone := original.Clone()

	assert.Equal(t, len(original), len(clone))
	assert.True(t, clone.Has(permission.OpRead))
	assert.True(t, clone.Has(permission.OpCreate))

	clone.Add(permission.OpUpdate)
	assert.True(t, clone.Has(permission.OpUpdate))
	assert.False(t, original.Has(permission.OpUpdate), "modifying clone should not affect original")
}

func TestExpandWithDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ops      []permission.Operation
		expected []permission.Operation
	}{
		{
			name:     "read has no dependencies",
			ops:      []permission.Operation{permission.OpRead},
			expected: []permission.Operation{permission.OpRead},
		},
		{
			name:     "create requires read",
			ops:      []permission.Operation{permission.OpCreate},
			expected: []permission.Operation{permission.OpCreate, permission.OpRead},
		},
		{
			name:     "update requires read",
			ops:      []permission.Operation{permission.OpUpdate},
			expected: []permission.Operation{permission.OpUpdate, permission.OpRead},
		},
		{
			name:     "export requires read",
			ops:      []permission.Operation{permission.OpExport},
			expected: []permission.Operation{permission.OpExport, permission.OpRead},
		},
		{
			name: "import requires read and create",
			ops:  []permission.Operation{permission.OpImport},
			expected: []permission.Operation{
				permission.OpImport,
				permission.OpRead,
				permission.OpCreate,
			},
		},
		{
			name: "approve requires read and update",
			ops:  []permission.Operation{permission.OpApprove},
			expected: []permission.Operation{
				permission.OpApprove,
				permission.OpRead,
				permission.OpUpdate,
			},
		},
		{
			name: "duplicate requires read and create",
			ops:  []permission.Operation{permission.OpDuplicate},
			expected: []permission.Operation{
				permission.OpDuplicate,
				permission.OpRead,
				permission.OpCreate,
			},
		},
		{
			name: "multiple operations merge dependencies",
			ops:  []permission.Operation{permission.OpCreate, permission.OpUpdate},
			expected: []permission.Operation{
				permission.OpCreate,
				permission.OpUpdate,
				permission.OpRead,
			},
		},
		{
			name:     "already has dependencies",
			ops:      []permission.Operation{permission.OpCreate, permission.OpRead},
			expected: []permission.Operation{permission.OpCreate, permission.OpRead},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			set := permission.NewOperationSet(tt.ops...)
			expanded := permission.ExpandWithDependencies(set)

			assert.Len(t, expanded, len(tt.expected))
			for _, op := range tt.expected {
				assert.True(t, expanded.Has(op), "expected operation %s in expanded set", op)
			}
		})
	}
}

func TestCollapseOnRevoke(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		initial  []permission.Operation
		revoke   permission.Operation
		expected []permission.Operation
	}{
		{
			name:     "revoke standalone operation",
			initial:  []permission.Operation{permission.OpRead, permission.OpExport},
			revoke:   permission.OpExport,
			expected: []permission.Operation{permission.OpRead},
		},
		{
			name:     "revoke read cascades to create",
			initial:  []permission.Operation{permission.OpRead, permission.OpCreate},
			revoke:   permission.OpRead,
			expected: []permission.Operation{},
		},
		{
			name: "revoke read cascades to multiple dependents",
			initial: []permission.Operation{
				permission.OpRead,
				permission.OpCreate,
				permission.OpUpdate,
				permission.OpExport,
			},
			revoke:   permission.OpRead,
			expected: []permission.Operation{},
		},
		{
			name: "revoke update cascades to approve",
			initial: []permission.Operation{
				permission.OpRead,
				permission.OpUpdate,
				permission.OpApprove,
			},
			revoke:   permission.OpUpdate,
			expected: []permission.Operation{permission.OpRead},
		},
		{
			name: "revoke create cascades to import and duplicate",
			initial: []permission.Operation{
				permission.OpRead,
				permission.OpCreate,
				permission.OpImport,
				permission.OpDuplicate,
			},
			revoke:   permission.OpCreate,
			expected: []permission.Operation{permission.OpRead},
		},
		{
			name:     "revoke non-existent operation",
			initial:  []permission.Operation{permission.OpRead, permission.OpCreate},
			revoke:   permission.OpExport,
			expected: []permission.Operation{permission.OpRead, permission.OpCreate},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			set := permission.NewOperationSet(tt.initial...)
			collapsed := permission.CollapseOnRevoke(set, tt.revoke)

			assert.Len(t, collapsed, len(tt.expected))
			for _, op := range tt.expected {
				assert.True(t, collapsed.Has(op), "expected operation %s in collapsed set", op)
			}
		})
	}
}

func TestOperationsToBitmask(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ops      []permission.Operation
		expected uint32
	}{
		{
			name:     "empty operations",
			ops:      []permission.Operation{},
			expected: 0,
		},
		{
			name:     "read only",
			ops:      []permission.Operation{permission.OpRead},
			expected: operationbits.Read,
		},
		{
			name:     "read and create",
			ops:      []permission.Operation{permission.OpRead, permission.OpCreate},
			expected: operationbits.Read | operationbits.Create,
		},
		{
			name: "all standard operations",
			ops: []permission.Operation{
				permission.OpRead,
				permission.OpCreate,
				permission.OpUpdate,
				permission.OpExport,
				permission.OpImport,
			},
			expected: operationbits.Read | operationbits.Create | operationbits.Update | operationbits.Export | operationbits.Import,
		},
		{
			name: "extended operations",
			ops: []permission.Operation{
				permission.OpApprove,
				permission.OpReject,
				permission.OpAssign,
			},
			expected: operationbits.Approve | operationbits.Reject | operationbits.Assign,
		},
		{
			name:     "unknown operation is ignored",
			ops:      []permission.Operation{permission.OpRead, permission.Operation("unknown")},
			expected: operationbits.Read,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := operationbits.Mask(tt.ops)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDependencies(t *testing.T) {
	t.Parallel()

	require.NotNil(t, permission.Dependencies)

	assert.Contains(t, permission.Dependencies[permission.OpCreate], permission.OpRead)
	assert.Contains(t, permission.Dependencies[permission.OpUpdate], permission.OpRead)
	assert.Contains(t, permission.Dependencies[permission.OpImport], permission.OpRead)
	assert.Contains(t, permission.Dependencies[permission.OpImport], permission.OpCreate)
	assert.Contains(t, permission.Dependencies[permission.OpApprove], permission.OpRead)
	assert.Contains(t, permission.Dependencies[permission.OpApprove], permission.OpUpdate)
}

func TestDependents(t *testing.T) {
	t.Parallel()

	require.NotNil(t, permission.Dependents)

	assert.Contains(t, permission.Dependents[permission.OpRead], permission.OpCreate)
	assert.Contains(t, permission.Dependents[permission.OpRead], permission.OpUpdate)
	assert.Contains(t, permission.Dependents[permission.OpRead], permission.OpExport)
	assert.Contains(t, permission.Dependents[permission.OpUpdate], permission.OpApprove)
	assert.Contains(t, permission.Dependents[permission.OpCreate], permission.OpImport)
}

func TestImplied_FollowsDependenciesTransitively(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]permission.Operation{permission.OpApprove, permission.OpRead, permission.OpUpdate},
		permission.Implied([]permission.Operation{permission.OpApprove}),
	)
	assert.Equal(
		t,
		[]permission.Operation{
			permission.OpDelete,
			permission.OpManage,
			permission.OpRead,
			permission.OpUpdate,
		},
		permission.Implied([]permission.Operation{permission.OpManage}),
	)
	assert.Empty(t, permission.Implied(nil))
}

func TestImpliedByOperation_LeavesOutTheOperationItself(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]permission.Operation{permission.OpCreate, permission.OpRead},
		permission.ImpliedByOperation(permission.OpImport),
	)
	assert.Empty(t, permission.ImpliedByOperation(permission.OpRead))
}

func TestRegistry_OperationsCarryWhatTheyImply(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	def, ok := reg.Get(permission.ResourceShipment.String())
	require.True(t, ok)

	for _, op := range def.Operations {
		assert.Equal(t, permission.ImpliedByOperation(op.Operation), op.Implies, "%s", op.Operation)
	}

	for _, op := range GetAllOperations() {
		assert.Equal(t, permission.ImpliedByOperation(op.Operation), op.Implies, "%s", op.Operation)
	}
}

func TestGrantingOperations_NamesEveryOperationThatBringsIt(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]permission.Operation{permission.OpDelete, permission.OpManage},
		permission.GrantingOperations(permission.OpDelete),
	)
	assert.Equal(
		t,
		[]permission.Operation{permission.OpApprove, permission.OpApproveOwn},
		permission.GrantingOperations(permission.OpApprove),
	)
	assert.Contains(t, permission.GrantingOperations(permission.OpUpdate), permission.OpApprove)
	assert.Contains(t, permission.GrantingOperations(permission.OpUpdate), permission.OpManage)
	assert.NotContains(t, permission.GrantingOperations(permission.OpUpdate), permission.OpCreate)
	assert.Len(t, permission.GrantingOperations(permission.OpRead), len(permission.Dependencies))
}
