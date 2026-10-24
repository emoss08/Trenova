package reference

import (
	"io/fs"
	"regexp"
	"slices"
	"testing"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/places/domain/jurisdictionrule"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	iftaJurisdictionIDPattern = regexp.MustCompile(`^ifj_[0-7][0-9A-HJKMNP-TV-Z]{25}$`)
	hazmatIDPattern           = regexp.MustCompile(`^dhr_[0-7][0-9A-HJKMNP-TV-Z]{25}$`)
)

func loadYAML[T any](t *testing.T, path string) T {
	t.Helper()

	raw, err := fs.ReadFile(data, path)
	require.NoError(t, err)

	var out T
	require.NoError(t, yaml.Unmarshal(raw, &out))
	return out
}

func TestTheRegistryHoldsOnlyValidReferenceSeeds(t *testing.T) {
	t.Parallel()

	require.NoError(t, Seeds.Validate())
	for _, s := range Seeds.Seeds() {
		assert.Equal(t, seed.Reference, s.Kind, "%s ships to production and must be reference data", s.Name)
	}
}

func TestSeedsThatReadStatesRunAfterThem(t *testing.T) {
	t.Parallel()

	for _, s := range []*seed.Seed{IFTAJurisdictions, JurisdictionRulesBaseline} {
		assert.True(t, slices.Contains(s.After, USStates), "%s resolves us_states ids", s.Name)
	}
}

func TestPerTenantSeedsReconcileEveryTenant(t *testing.T) {
	t.Parallel()

	for _, s := range []*seed.Seed{
		ChartOfAccounts,
		DocumentTypes,
		ServiceFailureReasonCodes,
		DocumentTemplateStarters,
		SystemAgentDefinitions,
		TableChangeAlertAllowlist,
		AdministratorPermissions,
	} {
		assert.True(t, s.PerTenant, "%s writes tenant rows", s.Name)
		assert.True(t, s.Version != "" || s.Fingerprint != nil, "%s needs a fingerprint source", s.Name)
	}
}

func TestEveryHazmatReferenceHasAStableUniqueID(t *testing.T) {
	t.Parallel()

	file := loadYAML[struct {
		References []hazmatRow `yaml:"references"`
	}](t, hazmatFile)
	require.NotEmpty(t, file.References)

	seen := make(map[string]struct{}, len(file.References))
	for _, row := range file.References {
		assert.Regexp(t, hazmatIDPattern, row.ID, "%s %s", row.UnNumber, row.ProperShippingName)
		_, dup := seen[row.ID]
		assert.False(t, dup, "%s is used twice", row.ID)
		seen[row.ID] = struct{}{}
	}
}

func TestIFTAJurisdictionsFixtureShape(t *testing.T) {
	t.Parallel()

	rows := loadYAML[iftaJurisdictionFile](t, iftaFile).Jurisdictions
	require.Len(t, rows, 64, "48 contiguous states + AK, HI, DC + 10 provinces + YT, NT, NU")

	members := 0
	surcharge := make([]string, 0, 3)
	nonMembers := make([]string, 0, 6)
	ids := make(map[string]struct{}, len(rows))
	keys := make(map[string]struct{}, len(rows))
	usRows, caRows := 0, 0

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
			assert.Equal(t, row.Code, row.UsStateAbbreviation, "%s must resolve to its us_states row", key)
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
	assert.ElementsMatch(t, []string{"US_AK", "US_HI", "US_DC", "CA_YT", "CA_NT", "CA_NU"}, nonMembers)
	assert.ElementsMatch(t, []string{"US_IN", "US_KY", "US_VA"}, surcharge,
		"only Indiana, Kentucky and Virginia levy a surcharge")
}

func TestBuildIFTAJurisdictionsResolvesStatesAndValidates(t *testing.T) {
	t.Parallel()

	rows := loadYAML[iftaJurisdictionFile](t, iftaFile).Jurisdictions
	texasID := pulid.MustNew("us_")

	jurisdictions, err := buildIFTAJurisdictions(rows, map[string]pulid.ID{"TX": texasID})
	require.NoError(t, err)
	require.Len(t, jurisdictions, len(rows))

	found := false
	for _, j := range jurisdictions {
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

func TestBuildIFTAJurisdictionsRejectsBadRows(t *testing.T) {
	t.Parallel()

	base := iftaJurisdictionRow{
		ID:           "ifj_01KDVDNA00YREPMM0K4QT2TNCE",
		CountryCode:  "US",
		Code:         "TX",
		Name:         "Texas",
		IsIftaMember: true,
		SortOrder:    1,
	}

	cases := map[string]struct {
		rows []iftaJurisdictionRow
		want string
	}{
		"duplicate id": {
			rows: []iftaJurisdictionRow{base, {ID: base.ID, CountryCode: "US", Code: "OK", Name: "Oklahoma", SortOrder: 2}},
			want: "duplicate id",
		},
		"duplicate code": {
			rows: []iftaJurisdictionRow{base, {ID: "ifj_01KDVDNA006J5BZHS8NWXP3VPA", CountryCode: "US", Code: "tx", Name: "Texas", SortOrder: 2}},
			want: "listed twice",
		},
		"missing id": {
			rows: []iftaJurisdictionRow{{CountryCode: "US", Code: "TX", Name: "Texas", SortOrder: 1}},
			want: "id is required",
		},
		"invalid row": {
			rows: []iftaJurisdictionRow{{ID: base.ID, CountryCode: "US", Code: "TX", SortOrder: 1}},
			want: "is invalid",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := buildIFTAJurisdictions(tc.rows, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestJurisdictionBaselineUsesOnlyFederallyUniformLimits(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 8.5, jurisdictionrule.FederalMaxWidthFeet, 0.001)
	assert.Equal(t, 80_000, jurisdictionrule.FederalMaxWeightPounds)
}

func TestJurisdictionBaselineErrsStrictOnStateVariableLimits(t *testing.T) {
	t.Parallel()

	assert.LessOrEqual(t, jurisdictionrule.FederalMaxHeightFeet, 14.0,
		"states commonly permit 13'6\" to 14'; the baseline must not exceed the top of that range")
	assert.GreaterOrEqual(t, jurisdictionrule.FederalMaxHeightFeet, 13.5,
		"below 13'6\" every standard dry van would raise a permit requirement")
	assert.LessOrEqual(t, jurisdictionrule.FederalMaxLengthFeet, 53.0,
		"53ft is the common single-trailer maximum; above it over-length loads go unflagged")
}

func TestJurisdictionBaselineDisclosesItsLimitsAndStaysUnverified(t *testing.T) {
	t.Parallel()

	for _, phrase := range []string{
		"Federal baseline only",
		"not this state's statute",
		"NOT populated",
		"mark this row Verified",
	} {
		assert.Contains(t, baselineSourceNote, phrase)
	}
	assert.NotEmpty(t, baselineSourceURL)

	rule := baselineRule(pulid.MustNew("us_"))
	assert.Equal(t, jurisdictionrule.VerificationUnverified, rule.VerificationState)
	assert.Equal(t, jurisdictionrule.StatusActive, rule.Status)
}
