package pagefavorite

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestPageFavorite_Validate(t *testing.T) {
	t.Parallel()

	pf := &PageFavorite{
		ID:             pulid.MustNew("pf_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		PageURL:        "/dashboard",
		PageTitle:      "Dashboard",
	}

	multiErr := errortypes.NewMultiError()
	pf.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestPageFavorite_GetTableName(t *testing.T) {
	t.Parallel()

	pf := &PageFavorite{}
	assert.Equal(t, "page_favorites", pf.GetTableName())
}

func TestPageFavorite_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		pf := &PageFavorite{}
		require.True(t, pf.ID.IsNil())

		err := pf.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, pf.ID.IsNil())
		assert.NotZero(t, pf.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("pf_")
		pf := &PageFavorite{ID: existingID}

		err := pf.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, pf.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		pf := &PageFavorite{}

		err := pf.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, pf.UpdatedAt)
	})
}

func TestPageFavorite_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("pf_")
	pf := &PageFavorite{ID: id}
	assert.Equal(t, id, pf.GetID())
}

func TestPageFavorite_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	pf := &PageFavorite{OrganizationID: orgID}
	assert.Equal(t, orgID, pf.GetOrganizationID())
}

func TestPageFavorite_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	pf := &PageFavorite{BusinessUnitID: buID}
	assert.Equal(t, buID, pf.GetBusinessUnitID())
}
