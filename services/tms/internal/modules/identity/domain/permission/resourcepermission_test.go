package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourcePermission_BeforeCreate(t *testing.T) {
	t.Parallel()

	t.Run("insert stores what was granted, once each and in order", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []Operation{OpUpdate, OpApprove, OpUpdate},
		}

		err := rp.BeforeCreate(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []Operation{OpApprove, OpUpdate}, rp.Operations)
	})

	t.Run("update stores what was granted without widening it", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []Operation{OpManage},
		}

		err := rp.BeforeUpdate(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []Operation{OpManage}, rp.Operations)
		assert.True(t, rp.HasOperation(OpDelete), "manage still allows delete when evaluated")
	})
}

func TestResourcePermission_HasOperation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		operations []Operation
		check      Operation
		expected   bool
	}{
		{
			name:       "operation present",
			operations: []Operation{OpRead, OpCreate, OpUpdate},
			check:      OpRead,
			expected:   true,
		},
		{
			name:       "operation not present",
			operations: []Operation{OpRead, OpCreate},
			check:      OpDelete,
			expected:   false,
		},
		{
			name:       "empty operations list",
			operations: []Operation{},
			check:      OpRead,
			expected:   false,
		},
		{
			name:       "check last operation in list",
			operations: []Operation{OpRead, OpCreate, OpExport},
			check:      OpExport,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rp := &ResourcePermission{Operations: tt.operations}
			assert.Equal(t, tt.expected, rp.HasOperation(tt.check))
		})
	}
}

func TestResourcePermission_GetOperationSet(t *testing.T) {
	t.Parallel()

	t.Run("returns set with all operations", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []Operation{OpRead, OpCreate, OpUpdate},
		}

		set := rp.GetOperationSet()

		assert.True(t, set.Has(OpRead))
		assert.True(t, set.Has(OpCreate))
		assert.True(t, set.Has(OpUpdate))
		assert.False(t, set.Has(OpDelete))
	})

	t.Run("empty operations returns empty set", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []Operation{},
		}

		set := rp.GetOperationSet()

		assert.Len(t, set, 0)
	})

	t.Run("duplicates are deduplicated in set", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []Operation{OpRead, OpRead, OpCreate},
		}

		set := rp.GetOperationSet()

		assert.Len(t, set, 2)
		assert.True(t, set.Has(OpRead))
		assert.True(t, set.Has(OpCreate))
	})
}
