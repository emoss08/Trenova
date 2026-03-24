package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOperationSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ops      []Operation
		expected map[Operation]bool
	}{
		{
			name:     "empty set",
			ops:      []Operation{},
			expected: map[Operation]bool{},
		},
		{
			name:     "single operation",
			ops:      []Operation{OpRead},
			expected: map[Operation]bool{OpRead: true},
		},
		{
			name:     "multiple operations",
			ops:      []Operation{OpRead, OpCreate, OpUpdate},
			expected: map[Operation]bool{OpRead: true, OpCreate: true, OpUpdate: true},
		},
		{
			name:     "duplicate operations",
			ops:      []Operation{OpRead, OpRead, OpCreate},
			expected: map[Operation]bool{OpRead: true, OpCreate: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			set := NewOperationSet(tt.ops...)
			assert.Equal(t, len(tt.expected), len(set))
			for op := range tt.expected {
				assert.True(t, set.Has(op), "expected operation %s to be in set", op)
			}
		})
	}
}

func TestOperationSet_Has(t *testing.T) {
	t.Parallel()

	set := NewOperationSet(OpRead, OpCreate)

	assert.True(t, set.Has(OpRead))
	assert.True(t, set.Has(OpCreate))
	assert.False(t, set.Has(OpUpdate))
	assert.False(t, set.Has(OpExport))
}

func TestOperationSet_Add(t *testing.T) {
	t.Parallel()

	set := NewOperationSet(OpRead)
	assert.True(t, set.Has(OpRead))
	assert.False(t, set.Has(OpCreate))

	set.Add(OpCreate, OpUpdate)
	assert.True(t, set.Has(OpRead))
	assert.True(t, set.Has(OpCreate))
	assert.True(t, set.Has(OpUpdate))
}

func TestOperationSet_Remove(t *testing.T) {
	t.Parallel()

	set := NewOperationSet(OpRead, OpCreate, OpUpdate)
	assert.True(t, set.Has(OpCreate))

	set.Remove(OpCreate)
	assert.False(t, set.Has(OpCreate))
	assert.True(t, set.Has(OpRead))
	assert.True(t, set.Has(OpUpdate))
}

func TestOperationSet_ToSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ops      []Operation
		expected int
	}{
		{
			name:     "empty set",
			ops:      []Operation{},
			expected: 0,
		},
		{
			name:     "single operation",
			ops:      []Operation{OpRead},
			expected: 1,
		},
		{
			name:     "multiple operations",
			ops:      []Operation{OpRead, OpCreate, OpUpdate},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			set := NewOperationSet(tt.ops...)
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

	original := NewOperationSet(OpRead, OpCreate)
	clone := original.Clone()

	assert.Equal(t, len(original), len(clone))
	assert.True(t, clone.Has(OpRead))
	assert.True(t, clone.Has(OpCreate))

	clone.Add(OpUpdate)
	assert.True(t, clone.Has(OpUpdate))
	assert.False(t, original.Has(OpUpdate), "modifying clone should not affect original")
}

func TestExpandWithDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ops      []Operation
		expected []Operation
	}{
		{
			name:     "read has no dependencies",
			ops:      []Operation{OpRead},
			expected: []Operation{OpRead},
		},
		{
			name:     "create requires read",
			ops:      []Operation{OpCreate},
			expected: []Operation{OpCreate, OpRead},
		},
		{
			name:     "update requires read",
			ops:      []Operation{OpUpdate},
			expected: []Operation{OpUpdate, OpRead},
		},
		{
			name:     "export requires read",
			ops:      []Operation{OpExport},
			expected: []Operation{OpExport, OpRead},
		},
		{
			name:     "import requires read and create",
			ops:      []Operation{OpImport},
			expected: []Operation{OpImport, OpRead, OpCreate},
		},
		{
			name:     "approve requires read and update",
			ops:      []Operation{OpApprove},
			expected: []Operation{OpApprove, OpRead, OpUpdate},
		},
		{
			name:     "duplicate requires read and create",
			ops:      []Operation{OpDuplicate},
			expected: []Operation{OpDuplicate, OpRead, OpCreate},
		},
		{
			name:     "multiple operations merge dependencies",
			ops:      []Operation{OpCreate, OpUpdate},
			expected: []Operation{OpCreate, OpUpdate, OpRead},
		},
		{
			name:     "already has dependencies",
			ops:      []Operation{OpCreate, OpRead},
			expected: []Operation{OpCreate, OpRead},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			set := NewOperationSet(tt.ops...)
			expanded := ExpandWithDependencies(set)

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
		initial  []Operation
		revoke   Operation
		expected []Operation
	}{
		{
			name:     "revoke standalone operation",
			initial:  []Operation{OpRead, OpExport},
			revoke:   OpExport,
			expected: []Operation{OpRead},
		},
		{
			name:     "revoke read cascades to create",
			initial:  []Operation{OpRead, OpCreate},
			revoke:   OpRead,
			expected: []Operation{},
		},
		{
			name:     "revoke read cascades to multiple dependents",
			initial:  []Operation{OpRead, OpCreate, OpUpdate, OpExport},
			revoke:   OpRead,
			expected: []Operation{},
		},
		{
			name:     "revoke update cascades to approve",
			initial:  []Operation{OpRead, OpUpdate, OpApprove},
			revoke:   OpUpdate,
			expected: []Operation{OpRead},
		},
		{
			name:     "revoke create cascades to import and duplicate",
			initial:  []Operation{OpRead, OpCreate, OpImport, OpDuplicate},
			revoke:   OpCreate,
			expected: []Operation{OpRead},
		},
		{
			name:     "revoke non-existent operation",
			initial:  []Operation{OpRead, OpCreate},
			revoke:   OpExport,
			expected: []Operation{OpRead, OpCreate},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			set := NewOperationSet(tt.initial...)
			collapsed := CollapseOnRevoke(set, tt.revoke)

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
		ops      []Operation
		expected uint32
	}{
		{
			name:     "empty operations",
			ops:      []Operation{},
			expected: 0,
		},
		{
			name:     "read only",
			ops:      []Operation{OpRead},
			expected: ClientOpRead,
		},
		{
			name:     "read and create",
			ops:      []Operation{OpRead, OpCreate},
			expected: ClientOpRead | ClientOpCreate,
		},
		{
			name:     "all standard operations",
			ops:      []Operation{OpRead, OpCreate, OpUpdate, OpExport, OpImport},
			expected: ClientOpRead | ClientOpCreate | ClientOpUpdate | ClientOpExport | ClientOpImport,
		},
		{
			name:     "extended operations",
			ops:      []Operation{OpApprove, OpReject, OpAssign},
			expected: ClientOpApprove | ClientOpReject | ClientOpAssign,
		},
		{
			name:     "unknown operation is ignored",
			ops:      []Operation{OpRead, Operation("unknown")},
			expected: ClientOpRead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := OperationsToBitmask(tt.ops)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDependencies(t *testing.T) {
	t.Parallel()

	require.NotNil(t, Dependencies)

	assert.Contains(t, Dependencies[OpCreate], OpRead)
	assert.Contains(t, Dependencies[OpUpdate], OpRead)
	assert.Contains(t, Dependencies[OpImport], OpRead)
	assert.Contains(t, Dependencies[OpImport], OpCreate)
	assert.Contains(t, Dependencies[OpApprove], OpRead)
	assert.Contains(t, Dependencies[OpApprove], OpUpdate)
}

func TestDependents(t *testing.T) {
	t.Parallel()

	require.NotNil(t, Dependents)

	assert.Contains(t, Dependents[OpRead], OpCreate)
	assert.Contains(t, Dependents[OpRead], OpUpdate)
	assert.Contains(t, Dependents[OpRead], OpExport)
	assert.Contains(t, Dependents[OpUpdate], OpApprove)
	assert.Contains(t, Dependents[OpCreate], OpImport)
}
