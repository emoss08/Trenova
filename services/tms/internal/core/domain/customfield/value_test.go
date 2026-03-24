package customfield

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestCustomFieldValue_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("cfv_")
	v := &CustomFieldValue{ID: id}
	assert.Equal(t, id, v.GetID())
}

func TestCustomFieldValue_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	v := &CustomFieldValue{OrganizationID: orgID}
	assert.Equal(t, orgID, v.GetOrganizationID())
}

func TestCustomFieldValue_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	v := &CustomFieldValue{BusinessUnitID: buID}
	assert.Equal(t, buID, v.GetBusinessUnitID())
}

func TestCustomFieldValue_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID CreatedAt and UpdatedAt", func(t *testing.T) {
		t.Parallel()

		v := &CustomFieldValue{}
		require.True(t, v.ID.IsNil())

		err := v.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, v.ID.IsNil())
		assert.Equal(t, "cfv_", v.ID.Prefix())
		assert.NotZero(t, v.CreatedAt)
		assert.NotZero(t, v.UpdatedAt)
		assert.Equal(t, v.CreatedAt, v.UpdatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("cfv_")
		v := &CustomFieldValue{ID: existingID}

		err := v.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, v.ID)
	})

	t.Run("update sets UpdatedAt only", func(t *testing.T) {
		t.Parallel()

		v := &CustomFieldValue{
			ID:        pulid.MustNew("cfv_"),
			CreatedAt: 1000,
			UpdatedAt: 1000,
		}

		err := v.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, int64(1000), v.CreatedAt)
		assert.NotEqual(t, int64(1000), v.UpdatedAt)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		v := &CustomFieldValue{}

		err := v.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, v.ID.IsNil())
		assert.Zero(t, v.CreatedAt)
		assert.Zero(t, v.UpdatedAt)
	})
}
