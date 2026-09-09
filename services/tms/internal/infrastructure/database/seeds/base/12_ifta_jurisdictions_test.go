package base

import (
	"regexp"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var iftaJurisdictionIDPattern = regexp.MustCompile(`^ifj_[0-7][0-9A-HJKMNP-TV-Z]{25}$`)

func loadIFTAJurisdictionFixture(t *testing.T) []iftaJurisdictionRow {
	t.Helper()

	rows, err := loadIFTAJurisdictionRows(seedhelpers.NewDataLoader("data"))
	require.NoError(t, err)
	return rows
}

func TestIFTAJurisdictionsSeed_RegistersAfterUSStates(t *testing.T) {
	t.Parallel()

	seed := NewIFTAJurisdictionsSeed()

	require.Equal(t, "IFTAJurisdictions", seed.Name())
	assert.Contains(t, seed.Environments(), common.EnvProduction)
	assert.Contains(t, seed.Environments(), common.EnvTest)
	assert.Contains(t, seed.DependsOn(), string(seedhelpers.SeedUSStates),
		"us_state_id is resolved from us_states and must not run before it")
}

func TestIFTAJurisdictionsSeed_FixtureShape(t *testing.T) {
	t.Parallel()

	rows := loadIFTAJurisdictionFixture(t)
	require.Len(t, rows, 64, "48 contiguous states + AK, HI, DC + 10 provinces + YT, NT, NU")

	members := 0
	surcharge := make([]string, 0, 3)
	nonMembers := make([]string, 0, 6)
	ids := make(map[string]struct{}, len(rows))
	keys := make(map[string]struct{}, len(rows))
	usRows := 0
	caRows := 0

	for _, row := range rows {
		assert.Regexp(t, iftaJurisdictionIDPattern, row.ID, "%s id must be a pulid", row.Code)
		_, dupID := ids[row.ID]
		assert.False(t, dupID, "%s reuses id %s", row.Code, row.ID)
		ids[row.ID] = struct{}{}

		key := row.CountryCode + "_" + row.Code
		_, dupKey := keys[key]
		assert.False(t, dupKey, "%s listed twice", key)
		keys[key] = struct{}{}

		assert.NotEmpty(t, row.Name, "%s has no name", key)
		assert.Positive(t, row.SortOrder, "%s has no sort order", key)

		switch row.CountryCode {
		case "US":
			usRows++
			assert.Equal(t, row.Code, row.UsStateAbbreviation,
				"%s must resolve to its us_states row", key)
		case "CA":
			caRows++
			assert.Empty(t, row.UsStateAbbreviation, "%s is not a US state", key)
		default:
			t.Errorf("%s: unexpected country %q", key, row.CountryCode)
		}

		if row.IsIftaMember {
			members++
		} else {
			nonMembers = append(nonMembers, key)
		}
		if row.HasSurcharge {
			surcharge = append(surcharge, key)
		}
	}

	assert.Equal(t, 51, usRows)
	assert.Equal(t, 13, caRows)
	assert.Equal(t, 58, members)
	assert.ElementsMatch(t,
		[]string{"US_AK", "US_HI", "US_DC", "CA_YT", "CA_NT", "CA_NU"},
		nonMembers,
	)
	assert.ElementsMatch(t, []string{"US_IN", "US_KY", "US_VA"}, surcharge,
		"only Indiana, Kentucky and Virginia levy a surcharge")
}

func TestBuildIFTAJurisdictions_ResolvesStatesAndValidates(t *testing.T) {
	t.Parallel()

	rows := loadIFTAJurisdictionFixture(t)

	texasID := pulid.MustNew("us_")
	stateIDs := map[string]pulid.ID{"TX": texasID}

	jurisdictions, err := buildIFTAJurisdictions(rows, stateIDs)
	require.NoError(t, err)
	require.Len(t, jurisdictions, len(rows))

	found := false
	for i := range jurisdictions {
		j := &jurisdictions[i]
		assert.False(t, j.ID.IsNil())
		assert.Equal(t, "Active", j.Status.String())

		switch j.Key() {
		case "US_TX":
			found = true
			require.NotNil(t, j.UsStateID)
			assert.Equal(t, texasID, *j.UsStateID)
		case "US_OK":
			assert.Nil(t, j.UsStateID, "an unknown abbreviation leaves the link empty for the next run")
		case "CA_ON":
			assert.Nil(t, j.UsStateID)
			assert.True(t, j.IsMember())
		}
	}
	assert.True(t, found, "Texas missing from the fixture")
}

func TestBuildIFTAJurisdictions_RejectsBadRows(t *testing.T) {
	t.Parallel()

	base := iftaJurisdictionRow{
		ID:           "ifj_01KDVDNA00YREPMM0K4QT2TNCE",
		CountryCode:  "US",
		Code:         "TX",
		Name:         "Texas",
		IsIftaMember: true,
		SortOrder:    1,
	}

	t.Run("duplicate id", func(t *testing.T) {
		t.Parallel()
		other := base
		other.Code = "OK"
		other.Name = "Oklahoma"
		_, err := buildIFTAJurisdictions([]iftaJurisdictionRow{base, other}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate id")
	})

	t.Run("duplicate code", func(t *testing.T) {
		t.Parallel()
		other := base
		other.ID = "ifj_01KDVDNA006J5BZHS8NWXP3VPA"
		other.Code = "tx"
		_, err := buildIFTAJurisdictions([]iftaJurisdictionRow{base, other}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "listed twice")
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()
		bad := base
		bad.ID = ""
		_, err := buildIFTAJurisdictions([]iftaJurisdictionRow{bad}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id is required")
	})

	t.Run("invalid row", func(t *testing.T) {
		t.Parallel()
		bad := base
		bad.Name = ""
		_, err := buildIFTAJurisdictions([]iftaJurisdictionRow{bad}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is invalid")
	})
}
