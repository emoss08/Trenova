package rbac

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRoleAssignment_BeforeCreate(t *testing.T) {
	t.Parallel()

	t.Run("insert sets AssignedAt", func(t *testing.T) {
		t.Parallel()

		ura := &UserRoleAssignment{}

		err := ura.BeforeCreate(t.Context())
		require.NoError(t, err)

		assert.NotZero(t, ura.AssignedAt)
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
