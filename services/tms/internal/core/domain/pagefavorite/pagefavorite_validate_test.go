package pagefavorite

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestPageFavorite_Validate_EmptyEntity(t *testing.T) {
	t.Parallel()

	pf := &PageFavorite{}
	multiErr := errortypes.NewMultiError()
	pf.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestPageFavorite_Validate_FullyPopulated(t *testing.T) {
	t.Parallel()

	pf := &PageFavorite{
		ID:             pulid.MustNew("pf_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		PageURL:        "/settings/general",
		PageTitle:      "General Settings",
		Version:        1,
		CreatedAt:      1700000000,
		UpdatedAt:      1700000000,
	}

	multiErr := errortypes.NewMultiError()
	pf.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestPageFavorite_Validate_EmptyURL(t *testing.T) {
	t.Parallel()

	pf := &PageFavorite{
		ID:             pulid.MustNew("pf_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		PageURL:        "",
		PageTitle:      "Dashboard",
	}

	multiErr := errortypes.NewMultiError()
	pf.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestPageFavorite_Validate_EmptyTitle(t *testing.T) {
	t.Parallel()

	pf := &PageFavorite{
		ID:             pulid.MustNew("pf_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		PageURL:        "/dashboard",
		PageTitle:      "",
	}

	multiErr := errortypes.NewMultiError()
	pf.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestPageFavorite_Validate_NilIDs(t *testing.T) {
	t.Parallel()

	pf := &PageFavorite{
		PageURL:   "/test",
		PageTitle: "Test",
	}

	multiErr := errortypes.NewMultiError()
	pf.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestPageFavorite_Validate_MultipleCallsAccumulate(t *testing.T) {
	t.Parallel()

	pf := &PageFavorite{
		PageURL:   "/test",
		PageTitle: "Test",
	}

	multiErr := errortypes.NewMultiError()
	pf.Validate(multiErr)
	pf.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
}
