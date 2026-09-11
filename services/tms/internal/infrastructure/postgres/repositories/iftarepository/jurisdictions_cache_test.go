package iftarepository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/iftarepository"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

type FakeJurisdictionCache struct {
	entries  []*ifta.Jurisdiction
	getCalls int
	setCalls int
	getErr   error
	setErr   error
}

func (c *FakeJurisdictionCache) GetAll(_ context.Context) ([]*ifta.Jurisdiction, error) {
	c.getCalls++
	if c.getErr != nil {
		return nil, c.getErr
	}
	if len(c.entries) == 0 {
		return nil, errortypes.NewNotFoundError("ifta jurisdictions not cached")
	}

	return c.entries, nil
}

func (c *FakeJurisdictionCache) Set(_ context.Context, entries []*ifta.Jurisdiction) error {
	c.setCalls++
	if c.setErr != nil {
		return c.setErr
	}
	c.entries = entries

	return nil
}

func (c *FakeJurisdictionCache) Invalidate(_ context.Context) error {
	c.entries = nil

	return nil
}

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

func warmCache() *FakeJurisdictionCache {
	return &FakeJurisdictionCache{entries: []*ifta.Jurisdiction{
		jurisdiction("TX", "US", 1, true, ifta.JurisdictionStatusActive),
		jurisdiction("OK", "US", 2, true, ifta.JurisdictionStatusActive),
		jurisdiction("OR", "US", 3, false, ifta.JurisdictionStatusActive),
		jurisdiction("AB", "CA", 4, true, ifta.JurisdictionStatusActive),
		jurisdiction("YT", "CA", 5, true, ifta.JurisdictionStatusInactive),
	}}
}

func newCacheTestRepository(
	t *testing.T,
	cache repositories.IFTAJurisdictionCacheRepository,
) (repositories.IFTARepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
	})

	repo := iftarepository.New(iftarepository.Params{
		DB:                postgres.NewTestConnection(bun.NewDB(db, pgdialect.New())),
		Logger:            zap.NewNop(),
		JurisdictionCache: cache,
	})

	return repo, mock
}

func TestListJurisdictions_WarmCacheIssuesNoQuery(t *testing.T) {
	t.Parallel()

	cache := warmCache()
	repo, _ := newCacheTestRepository(t, cache)

	out, err := repo.ListJurisdictions(t.Context(), &repositories.ListJurisdictionsRequest{})
	require.NoError(t, err)

	assert.Len(t, out, 5)
	assert.Equal(t, 1, cache.getCalls)
	assert.Equal(t, 0, cache.setCalls, "a warm cache must not be rewritten")
}

func TestListJurisdictions_FiltersTheSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("members only drops non-members", func(t *testing.T) {
		t.Parallel()
		repo, _ := newCacheTestRepository(t, warmCache())

		out, err := repo.ListJurisdictions(t.Context(), &repositories.ListJurisdictionsRequest{
			MembersOnly: true,
		})
		require.NoError(t, err)

		codes := codesOf(out)
		assert.Equal(t, []string{"TX", "OK", "AB", "YT"}, codes)
	})

	t.Run("country code is matched case and space insensitively", func(t *testing.T) {
		t.Parallel()
		repo, _ := newCacheTestRepository(t, warmCache())

		out, err := repo.ListJurisdictions(t.Context(), &repositories.ListJurisdictionsRequest{
			CountryCode: "  ca ",
		})
		require.NoError(t, err)

		assert.Equal(t, []string{"AB", "YT"}, codesOf(out))
	})

	t.Run("statuses narrow to the listed ones", func(t *testing.T) {
		t.Parallel()
		repo, _ := newCacheTestRepository(t, warmCache())

		out, err := repo.ListJurisdictions(t.Context(), &repositories.ListJurisdictionsRequest{
			Statuses: []ifta.JurisdictionStatus{ifta.JurisdictionStatusInactive},
		})
		require.NoError(t, err)

		assert.Equal(t, []string{"YT"}, codesOf(out))
	})

	t.Run("filters combine", func(t *testing.T) {
		t.Parallel()
		repo, _ := newCacheTestRepository(t, warmCache())

		out, err := repo.ListJurisdictions(t.Context(), &repositories.ListJurisdictionsRequest{
			MembersOnly: true,
			CountryCode: "US",
			Statuses:    []ifta.JurisdictionStatus{ifta.JurisdictionStatusActive},
		})
		require.NoError(t, err)

		assert.Equal(t, []string{"TX", "OK"}, codesOf(out))
	})

	t.Run("the load order survives filtering", func(t *testing.T) {
		t.Parallel()
		repo, _ := newCacheTestRepository(t, warmCache())

		out, err := repo.ListJurisdictions(t.Context(), &repositories.ListJurisdictionsRequest{})
		require.NoError(t, err)

		assert.Equal(t, []string{"TX", "OK", "OR", "AB", "YT"}, codesOf(out))
	})
}

func TestListJurisdictions_ColdCacheLoadsOnceThenServesFromCache(t *testing.T) {
	t.Parallel()

	cache := &FakeJurisdictionCache{}
	repo, mock := newCacheTestRepository(t, cache)

	mock.ExpectQuery(`FROM "ifta_jurisdictions"`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "country_code", "code", "name", "is_ifta_member", "sort_order", "status"}).
			AddRow(pulid.MustNew("ifj_").String(), "US", "TX", "US-TX", true, 1, "Active").
			AddRow(pulid.MustNew("ifj_").String(), "CA", "AB", "CA-AB", true, 2, "Active"),
	)

	first, err := repo.ListJurisdictions(t.Context(), &repositories.ListJurisdictionsRequest{})
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.Equal(t, 1, cache.setCalls, "a cold read must populate the cache")

	second, err := repo.GetJurisdictionByCode(t.Context(), "us", " tx ")
	require.NoError(t, err)
	assert.Equal(t, "TX", second.Code)
}

func TestListJurisdictions_CacheFailureFallsThroughToPostgres(t *testing.T) {
	t.Parallel()

	cache := &FakeJurisdictionCache{
		getErr: errors.New("redis is down"),
		setErr: errors.New("redis is down"),
	}
	repo, mock := newCacheTestRepository(t, cache)

	mock.ExpectQuery(`FROM "ifta_jurisdictions"`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "country_code", "code", "name", "is_ifta_member", "sort_order", "status"}).
			AddRow(pulid.MustNew("ifj_").String(), "US", "TX", "US-TX", true, 1, "Active"),
	)

	out, err := repo.ListJurisdictions(t.Context(), &repositories.ListJurisdictionsRequest{})
	require.NoError(t, err)

	assert.Len(t, out, 1)
	assert.Equal(t, 1, cache.setCalls, "a failed Set must not fail the read")
}

func TestJurisdictionLookups_AgainstTheSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("by id", func(t *testing.T) {
		t.Parallel()
		cache := warmCache()
		repo, _ := newCacheTestRepository(t, cache)
		wanted := cache.entries[2]

		got, err := repo.GetJurisdictionByID(t.Context(), wanted.ID)
		require.NoError(t, err)
		assert.Equal(t, "OR", got.Code)
	})

	t.Run("by id, missing", func(t *testing.T) {
		t.Parallel()
		repo, _ := newCacheTestRepository(t, warmCache())

		_, err := repo.GetJurisdictionByID(t.Context(), pulid.MustNew("ifj_"))
		require.Error(t, err)
		assert.True(t, errortypes.IsNotFoundError(err), "expected a not-found error, got %v", err)
	})

	t.Run("by ids keeps load order and drops unknown ids", func(t *testing.T) {
		t.Parallel()
		cache := warmCache()
		repo, _ := newCacheTestRepository(t, cache)

		got, err := repo.GetJurisdictionsByIDs(t.Context(), []pulid.ID{
			cache.entries[3].ID, cache.entries[0].ID, pulid.MustNew("ifj_"),
		})
		require.NoError(t, err)

		assert.Equal(t, []string{"TX", "AB"}, codesOf(got))
	})

	t.Run("by ids, empty input short circuits", func(t *testing.T) {
		t.Parallel()
		cache := warmCache()
		repo, _ := newCacheTestRepository(t, cache)

		got, err := repo.GetJurisdictionsByIDs(t.Context(), nil)
		require.NoError(t, err)

		assert.Empty(t, got)
		assert.Equal(t, 0, cache.getCalls, "no ids means nothing to look up")
	})

	t.Run("by code, missing", func(t *testing.T) {
		t.Parallel()
		repo, _ := newCacheTestRepository(t, warmCache())

		_, err := repo.GetJurisdictionByCode(t.Context(), "US", "ZZ")
		require.Error(t, err)
		assert.True(t, errortypes.IsNotFoundError(err), "expected a not-found error, got %v", err)
	})
}

func TestFindJurisdictionsByCodes_MatchesExactPairs(t *testing.T) {
	t.Parallel()

	cache := warmCache()
	repo, _ := newCacheTestRepository(t, cache)

	out, err := repo.FindJurisdictionsByCodes(t.Context(), []string{
		"US_TX", "CA-AB", "tx", "CA_TX", "", "US_ZZ",
	})
	require.NoError(t, err)

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

func TestFindJurisdictionsByCodes_EmptyInputSkipsTheCache(t *testing.T) {
	t.Parallel()

	cache := warmCache()
	repo, _ := newCacheTestRepository(t, cache)

	out, err := repo.FindJurisdictionsByCodes(t.Context(), nil)
	require.NoError(t, err)

	assert.Empty(t, out)
	assert.Equal(t, 0, cache.getCalls)
}

func codesOf(entries []*ifta.Jurisdiction) []string {
	codes := make([]string, 0, len(entries))
	for _, entry := range entries {
		codes = append(codes, entry.Code)
	}

	return codes
}
