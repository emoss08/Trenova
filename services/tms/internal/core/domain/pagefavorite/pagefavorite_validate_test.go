package pagefavorite

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldsWithErrors(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func TestPageFavorite_Validate_EmptyEntity(t *testing.T) {
	t.Parallel()

	pf := &PageFavorite{}
	multiErr := errortypes.NewMultiError()
	pf.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.ElementsMatch(t,
		[]string{"organizationId", "businessUnitId", "userId", "pageUrl", "pageTitle"},
		fieldsWithErrors(multiErr),
	)
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

	require.True(t, multiErr.HasErrors())
	assert.Equal(t, []string{"pageUrl"}, fieldsWithErrors(multiErr))
}

// An absolute URL would be rendered in the sidebar as if it were a page of the
// application, so it is refused rather than stored.
func TestPageFavorite_Validate_ExternalURL(t *testing.T) {
	t.Parallel()

	pf := &PageFavorite{
		ID:             pulid.MustNew("pf_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		PageURL:        "https://example.com/phish",
		PageTitle:      "Dashboard",
	}

	multiErr := errortypes.NewMultiError()
	pf.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Equal(t, []string{"pageUrl"}, fieldsWithErrors(multiErr))
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

	require.True(t, multiErr.HasErrors())
	assert.Equal(t, []string{"pageTitle"}, fieldsWithErrors(multiErr))
}

func TestPageFavorite_Validate_NilIDs(t *testing.T) {
	t.Parallel()

	pf := &PageFavorite{
		PageURL:   "/test",
		PageTitle: "Test",
	}

	multiErr := errortypes.NewMultiError()
	pf.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.ElementsMatch(t,
		[]string{"organizationId", "businessUnitId", "userId"},
		fieldsWithErrors(multiErr),
	)
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

	assert.Len(t, multiErr.Errors, 6)
}
