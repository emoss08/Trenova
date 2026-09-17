package carrierok_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/carrierok"
	"github.com/stretchr/testify/assert"
)

func TestProfileQueryValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query carrierok.ProfileQuery
		valid bool
	}{
		{name: "dot", query: carrierok.ProfileQuery{DOTNumber: "818175"}, valid: true},
		{name: "docket", query: carrierok.ProfileQuery{DocketNumber: "MC277621"}, valid: true},
		{
			name:  "plate with state",
			query: carrierok.ProfileQuery{PlateNumber: "P1", PlateState: "IL"},
			valid: true,
		},
		{
			name:  "unit with type",
			query: carrierok.ProfileQuery{UnitNumber: "101", UnitType: "truck"},
			valid: true,
		},
		{name: "none", query: carrierok.ProfileQuery{}},
		{name: "blank", query: carrierok.ProfileQuery{DOTNumber: "   "}},
		{name: "two identifiers", query: carrierok.ProfileQuery{DOTNumber: "1", EIN: "2"}},
		{name: "non numeric dot", query: carrierok.ProfileQuery{DOTNumber: "81A"}},
		{
			name:  "plate state alone",
			query: carrierok.ProfileQuery{DOTNumber: "1", PlateState: "IL"},
		},
		{name: "unit type alone", query: carrierok.ProfileQuery{DOTNumber: "1", UnitType: "TRUCK"}},
		{
			name:  "unit type invalid",
			query: carrierok.ProfileQuery{UnitNumber: "101", UnitType: "VAN"},
		},
	}

	for _, tt := range tests {
		err := tt.query.Validate()
		if tt.valid {
			assert.NoError(t, err, tt.name)
			continue
		}
		assert.ErrorIs(t, err, carrierok.ErrInvalidQuery, tt.name)
	}
}

func TestFMCSAQueryValidate(t *testing.T) {
	t.Parallel()

	assert.NoError(t, carrierok.FMCSAQuery{DOTNumber: "818175"}.Validate())
	assert.NoError(t, carrierok.FMCSAQuery{DocketNumber: "MC277621"}.Validate())
	assert.ErrorIs(t, carrierok.FMCSAQuery{}.Validate(), carrierok.ErrInvalidQuery)
	assert.ErrorIs(
		t,
		carrierok.FMCSAQuery{DOTNumber: "1", DocketNumber: "MC1"}.Validate(),
		carrierok.ErrInvalidQuery,
	)
	assert.ErrorIs(t, carrierok.FMCSAQuery{DOTNumber: "x1"}.Validate(), carrierok.ErrInvalidQuery)
}

func TestSearchParamsValidate(t *testing.T) {
	t.Parallel()

	assert.NoError(t, (&carrierok.SearchParams{Query: "acme"}).Validate())
	assert.NoError(t, (&carrierok.SearchParams{State: "TX"}).Validate())
	assert.ErrorIs(t, (&carrierok.SearchParams{}).Validate(), carrierok.ErrInvalidQuery)
	assert.ErrorIs(
		t,
		(&carrierok.SearchParams{Query: "acme", State: "Texas"}).Validate(),
		carrierok.ErrInvalidQuery,
	)
	assert.ErrorIs(
		t,
		(&carrierok.SearchParams{Query: "acme", Limit: -1}).Validate(),
		carrierok.ErrInvalidQuery,
	)
}
