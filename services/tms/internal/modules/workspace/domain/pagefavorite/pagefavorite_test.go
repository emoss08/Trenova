package pagefavorite

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
