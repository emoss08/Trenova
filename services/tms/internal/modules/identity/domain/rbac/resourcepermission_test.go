package rbac

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourcePermission_BeforeCreate(t *testing.T) {
	t.Parallel()

	t.Run("insert stores what was granted, once each and in order", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []permission.Operation{
				permission.OpUpdate,
				permission.OpApprove,
				permission.OpUpdate,
			},
		}

		err := rp.BeforeCreate(t.Context())
		require.NoError(t, err)

		assert.Equal(
			t,
			[]permission.Operation{permission.OpApprove, permission.OpUpdate},
			rp.Operations,
		)
	})

	t.Run("update stores what was granted without widening it", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []permission.Operation{permission.OpManage},
		}

		err := rp.BeforeUpdate(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []permission.Operation{permission.OpManage}, rp.Operations)
		assert.True(
			t,
			rp.HasOperation(permission.OpDelete),
			"manage still allows delete when evaluated",
		)
	})
}

func TestResourcePermission_HasOperation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		operations []permission.Operation
		check      permission.Operation
		expected   bool
	}{
		{
			name: "operation present",
			operations: []permission.Operation{
				permission.OpRead,
				permission.OpCreate,
				permission.OpUpdate,
			},
			check:    permission.OpRead,
			expected: true,
		},
		{
			name:       "operation not present",
			operations: []permission.Operation{permission.OpRead, permission.OpCreate},
			check:      permission.OpDelete,
			expected:   false,
		},
		{
			name:       "empty operations list",
			operations: []permission.Operation{},
			check:      permission.OpRead,
			expected:   false,
		},
		{
			name: "check last operation in list",
			operations: []permission.Operation{
				permission.OpRead,
				permission.OpCreate,
				permission.OpExport,
			},
			check:    permission.OpExport,
			expected: true,
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
			Operations: []permission.Operation{
				permission.OpRead,
				permission.OpCreate,
				permission.OpUpdate,
			},
		}

		set := rp.GetOperationSet()

		assert.True(t, set.Has(permission.OpRead))
		assert.True(t, set.Has(permission.OpCreate))
		assert.True(t, set.Has(permission.OpUpdate))
		assert.False(t, set.Has(permission.OpDelete))
	})

	t.Run("empty operations returns empty set", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []permission.Operation{},
		}

		set := rp.GetOperationSet()

		assert.Len(t, set, 0)
	})

	t.Run("duplicates are deduplicated in set", func(t *testing.T) {
		t.Parallel()

		rp := &ResourcePermission{
			Operations: []permission.Operation{
				permission.OpRead,
				permission.OpRead,
				permission.OpCreate,
			},
		}

		set := rp.GetOperationSet()

		assert.Len(t, set, 2)
		assert.True(t, set.Has(permission.OpRead))
		assert.True(t, set.Has(permission.OpCreate))
	})
}
