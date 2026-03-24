package permission

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestUserRoleAssignment_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and AssignedAt", func(t *testing.T) {
		t.Parallel()

		ura := &UserRoleAssignment{}
		require.True(t, ura.ID.IsNil())

		err := ura.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, ura.ID.IsNil())
		assert.True(t, strings.HasPrefix(string(ura.ID), "ura_"))
		assert.NotZero(t, ura.AssignedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("ura_")
		ura := &UserRoleAssignment{ID: existingID}

		err := ura.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, ura.ID)
	})

	t.Run("update does not modify fields", func(t *testing.T) {
		t.Parallel()

		ura := &UserRoleAssignment{}

		err := ura.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.True(t, ura.ID.IsNil())
		assert.Zero(t, ura.AssignedAt)
	})
}

func TestUserRoleAssignment_IsExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expiresAt *int64
		expected  bool
	}{
		{
			name:      "nil ExpiresAt is not expired",
			expiresAt: nil,
			expected:  false,
		},
		{
			name: "future date is not expired",
			expiresAt: func() *int64 {
				v := time.Now().Add(24 * time.Hour).Unix()
				return &v
			}(),
			expected: false,
		},
		{
			name: "past date is expired",
			expiresAt: func() *int64 {
				v := time.Now().Add(-24 * time.Hour).Unix()
				return &v
			}(),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ura := &UserRoleAssignment{ExpiresAt: tt.expiresAt}
			assert.Equal(t, tt.expected, ura.IsExpired())
		})
	}
}

func TestUserRoleAssignment_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ura_")
	ura := &UserRoleAssignment{ID: id}
	assert.Equal(t, id, ura.GetID())
}
