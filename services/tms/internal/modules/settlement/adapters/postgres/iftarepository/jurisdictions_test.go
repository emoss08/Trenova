package iftarepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jurisdiction(
	code, country string,
	sortOrder int,
	member bool,
	status ifta.JurisdictionStatus,
) *ifta.Jurisdiction {
	return &ifta.Jurisdiction{
		ID:           pulid.MustNew("ifj_"),
		CountryCode:  country,
		Code:         code,
		Name:         country + "-" + code,
		IsIftaMember: member,
		SortOrder:    sortOrder,
		Status:       status,
	}
}

func catalog() []*ifta.Jurisdiction {
	return []*ifta.Jurisdiction{
		jurisdiction("TX", "US", 1, true, ifta.JurisdictionStatusActive),
		jurisdiction("OK", "US", 2, true, ifta.JurisdictionStatusActive),
		jurisdiction("OR", "US", 3, false, ifta.JurisdictionStatusActive),
		jurisdiction("AB", "CA", 4, true, ifta.JurisdictionStatusActive),
		jurisdiction("YT", "CA", 5, true, ifta.JurisdictionStatusInactive),
	}
}

func TestFilterJurisdictions(t *testing.T) {
	t.Parallel()

	t.Run("members only drops non-members", func(t *testing.T) {
		t.Parallel()

		out := filterJurisdictions(catalog(), &iftaports.ListJurisdictionsRequest{
			MembersOnly: true,
		})

		assert.Equal(t, []string{"TX", "OK", "AB", "YT"}, codesOf(out))
	})

	t.Run("country code is matched case and space insensitively", func(t *testing.T) {
		t.Parallel()

		out := filterJurisdictions(catalog(), &iftaports.ListJurisdictionsRequest{
			CountryCode: "  ca ",
		})

		assert.Equal(t, []string{"AB", "YT"}, codesOf(out))
	})

	t.Run("statuses narrow to the listed ones", func(t *testing.T) {
		t.Parallel()

		out := filterJurisdictions(catalog(), &iftaports.ListJurisdictionsRequest{
			Statuses: []ifta.JurisdictionStatus{ifta.JurisdictionStatusInactive},
		})

		assert.Equal(t, []string{"YT"}, codesOf(out))
	})

	t.Run("filters combine", func(t *testing.T) {
		t.Parallel()

		out := filterJurisdictions(catalog(), &iftaports.ListJurisdictionsRequest{
			MembersOnly: true,
			CountryCode: "US",
			Statuses:    []ifta.JurisdictionStatus{ifta.JurisdictionStatusActive},
		})

		assert.Equal(t, []string{"TX", "OK"}, codesOf(out))
	})

	t.Run("the load order survives filtering", func(t *testing.T) {
		t.Parallel()

		out := filterJurisdictions(catalog(), &iftaports.ListJurisdictionsRequest{})

		assert.Equal(t, []string{"TX", "OK", "OR", "AB", "YT"}, codesOf(out))
	})
}

func TestJurisdictionLookups(t *testing.T) {
	t.Parallel()

	t.Run("by id", func(t *testing.T) {
		t.Parallel()
		all := catalog()

		got, err := jurisdictionByID(all, all[2].ID)
		require.NoError(t, err)
		assert.Equal(t, "OR", got.Code)
	})

	t.Run("by id, missing", func(t *testing.T) {
		t.Parallel()

		_, err := jurisdictionByID(catalog(), pulid.MustNew("ifj_"))
		require.Error(t, err)
		assert.True(t, errortypes.IsNotFoundError(err), "expected a not-found error, got %v", err)
	})

	t.Run("by ids keeps load order and drops unknown ids", func(t *testing.T) {
		t.Parallel()
		all := catalog()

		got := jurisdictionsByIDs(all, []pulid.ID{all[3].ID, all[0].ID, pulid.MustNew("ifj_")})

		assert.Equal(t, []string{"TX", "AB"}, codesOf(got))
	})

	t.Run("by code ignores case and spaces", func(t *testing.T) {
		t.Parallel()

		got, err := jurisdictionByCode(catalog(), "us", " tx ")
		require.NoError(t, err)
		assert.Equal(t, "TX", got.Code)
	})

	t.Run("by code, missing", func(t *testing.T) {
		t.Parallel()

		_, err := jurisdictionByCode(catalog(), "US", "ZZ")
		require.Error(t, err)
		assert.True(t, errortypes.IsNotFoundError(err), "expected a not-found error, got %v", err)
	})
}

func TestJurisdictionsByKeys_MatchesExactPairs(t *testing.T) {
	t.Parallel()

	out := jurisdictionsByKeys(catalog(), normaliseJurisdictionKeys([]string{
		"US_TX", "CA-AB", "tx", "CA_TX", "", "US_ZZ",
	}))

	require.Contains(t, out, "US_TX")
	assert.Equal(t, "TX", out["US_TX"].Code)

	require.Contains(t, out, "CA-AB")
	assert.Equal(t, "AB", out["CA-AB"].Code)

	require.Contains(t, out, "tx")
	assert.Equal(t, "US", out["tx"].CountryCode)

	assert.NotContains(t, out, "CA_TX", "TX is a US jurisdiction, not a Canadian one")
	assert.NotContains(t, out, "US_ZZ")
	assert.NotContains(t, out, "")
}

func TestNormaliseJurisdictionKeys_DropsKeysThatNameNothing(t *testing.T) {
	t.Parallel()

	assert.Empty(t, normaliseJurisdictionKeys(nil))
	assert.Empty(t, normaliseJurisdictionKeys([]string{"", "  ", "US_"}))
}

func codesOf(entries []*ifta.Jurisdiction) []string {
	codes := make([]string, 0, len(entries))
	for _, entry := range entries {
		codes = append(codes, entry.Code)
	}

	return codes
}
