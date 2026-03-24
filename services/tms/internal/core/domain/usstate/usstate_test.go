package usstate

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestUsState_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()
		us := &UsState{}
		err := us.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)
		assert.False(t, us.ID.IsNil())
		assert.NotZero(t, us.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()
		existingID := pulid.MustNew("us_")
		us := &UsState{ID: existingID}
		err := us.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)
		assert.Equal(t, existingID, us.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()
		us := &UsState{}
		err := us.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)
		assert.NotZero(t, us.UpdatedAt)
	})
}
