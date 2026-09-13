package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jurisdiction(countryCode, code, name string, sortOrder int) *ifta.Jurisdiction {
	return &ifta.Jurisdiction{
		ID:           pulid.MustNew("ifj_"),
		CountryCode:  countryCode,
		Code:         code,
		Name:         name,
		SortOrder:    sortOrder,
		IsIftaMember: true,
	}
}

// The picker reads like a fuel-tax return: US jurisdictions first, then Canada,
// then Mexico, and the seeded sort order inside each country.
func TestSortJurisdictions_OrdersByCountryThenSortOrder(t *testing.T) {
	t.Parallel()

	jurisdictions := []*ifta.Jurisdiction{
		jurisdiction("MX", "BCN", "Baja California", 1),
		jurisdiction("US", "TX", "Texas", 44),
		jurisdiction("CA", "ON", "Ontario", 9),
		jurisdiction("US", "AL", "Alabama", 1),
	}

	sortJurisdictions(jurisdictions)

	codes := make([]string, 0, len(jurisdictions))
	for _, entity := range jurisdictions {
		codes = append(codes, entity.Code)
	}
	assert.Equal(t, []string{"AL", "TX", "ON", "BCN"}, codes)
}

// A country the rank map does not name still has to sort somewhere rather than
// tie with the United States, or the ordering flips between calls.
func TestSortJurisdictions_UnknownCountrySortsLast(t *testing.T) {
	t.Parallel()

	jurisdictions := []*ifta.Jurisdiction{
		jurisdiction("ZZ", "QQ", "Elsewhere", 1),
		jurisdiction("US", "AL", "Alabama", 1),
	}

	sortJurisdictions(jurisdictions)

	assert.Equal(t, "AL", jurisdictions[0].Code)
	assert.Equal(t, "QQ", jurisdictions[1].Code)
}

func TestMatchingJurisdictions_SearchesCodeAndName(t *testing.T) {
	t.Parallel()

	jurisdictions := []*ifta.Jurisdiction{
		jurisdiction("US", "TX", "Texas", 44),
		jurisdiction("US", "OK", "Oklahoma", 37),
		jurisdiction("CA", "ON", "Ontario", 9),
	}

	byCode := matchingJurisdictions(jurisdictions, " tx ")
	require.Len(t, byCode, 1)
	assert.Equal(t, "TX", byCode[0].Code)

	byName := matchingJurisdictions(jurisdictions, "okla")
	require.Len(t, byName, 1)
	assert.Equal(t, "OK", byName[0].Code)

	assert.Len(t, matchingJurisdictions(jurisdictions, ""), 3)
	assert.Empty(t, matchingJurisdictions(jurisdictions, "zzz"))
}

func TestIFTAJurisdictionSelectOptionItem_LabelsCodeAndName(t *testing.T) {
	t.Parallel()

	item := iftaJurisdictionSelectOptionItem(&ifta.Jurisdiction{
		ID:           pulid.MustNew("ifj_"),
		CountryCode:  "US",
		Code:         "TX",
		Name:         "Texas",
		IsIftaMember: true,
		HasSurcharge: false,
		CreatedAt:    1780415890,
	})

	assert.Equal(t, "TX — Texas", item.option.Label)
	require.NotNil(t, item.option.Description)
	assert.Equal(t, "Texas", *item.option.Description)
	assert.Equal(t, "US", item.option.Meta["countryCode"])
	assert.Equal(t, true, item.option.Meta["isIftaMember"])
}

// Non-member jurisdictions stay selectable for mileage entry, so the option has
// to say why it is different rather than be dropped from the list.
func TestIFTAJurisdictionSelectOptionItem_FlagsNonMembers(t *testing.T) {
	t.Parallel()

	item := iftaJurisdictionSelectOptionItem(&ifta.Jurisdiction{
		ID:           pulid.MustNew("ifj_"),
		CountryCode:  "US",
		Code:         "DC",
		Name:         "District of Columbia",
		IsIftaMember: false,
	})

	require.NotNil(t, item.option.Description)
	assert.Equal(t, "District of Columbia · Not an IFTA member", *item.option.Description)
	assert.Equal(t, false, item.option.Meta["isIftaMember"])
}
