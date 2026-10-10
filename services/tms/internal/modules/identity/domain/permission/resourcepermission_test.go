package permission

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestResourcePermission_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and timestamps", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []Operation{OpCreate},
		}
		require.True(t, rp.ID.IsNil())

		err := rp.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, rp.ID.IsNil())
		assert.True(t, strings.HasPrefix(string(rp.ID), "rp_"))
		assert.NotZero(t, rp.CreatedAt)
		assert.NotZero(t, rp.UpdatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("rp_")
		rp := &ResourcePermission{
			ID:         existingID,
			Operations: []Operation{OpRead},
		}

		err := rp.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, rp.ID)
	})

	t.Run("insert expands operations with dependencies", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []Operation{OpCreate},
		}

		err := rp.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Contains(t, rp.Operations, OpCreate)
		assert.Contains(t, rp.Operations, OpRead)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []Operation{OpRead},
		}

		err := rp.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, rp.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []Operation{OpRead},
		}

		err := rp.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, rp.CreatedAt)
		assert.NotZero(t, rp.UpdatedAt)
	})

	t.Run("update expands operations with dependencies", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []Operation{OpUpdate},
		}

		err := rp.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Contains(t, rp.Operations, OpUpdate)
		assert.Contains(t, rp.Operations, OpRead)
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
