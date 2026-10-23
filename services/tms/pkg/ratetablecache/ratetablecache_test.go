package ratetablecache_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type source struct {
	stamp  atomic.Value
	stamps atomic.Int32
	builds atomic.Int32
	loads  atomic.Int32
	err    error
}

func newSource(stamp string) *source {
	s := &source{}
	s.stamp.Store(stamp)
	return s
}

func (s *source) Stamp(context.Context) (string, error) {
	s.stamps.Add(1)
	if s.err != nil {
		return "", s.err
	}
	return s.stamp.Load().(string), nil //nolint:errcheck // always a string
}

func (s *source) Build(_ context.Context, stamp string) (*ratetablecache.Served, error) {
	s.builds.Add(1)
	return &ratetablecache.Served{Versions: []*ratetablecache.Version{{
		Shape: rules.TariffTable{Code: "ZONES_" + stamp},
		Index: ratetablecache.Table{Code: "ZONES_" + stamp},
	}}}, nil
}

func (s *source) Load(
	_ context.Context,
	keys []ratetablecache.VersionKey,
) ([]*ratetablecache.Version, error) {
	s.loads.Add(1)
	out := make([]*ratetablecache.Version, 0, len(keys))
	for _, key := range keys {
		out = append(out, &ratetablecache.Version{
			Shape: rules.TariffTable{Code: "ZONES", Version: key.Version},
			Index: ratetablecache.Table{Key: key, Code: "ZONES"},
		})
	}
	return out, nil
}

func tenant() (pulid.ID, pulid.ID) {
	return pulid.MustNew("org_"), pulid.MustNew("bu_")
}

func TestServedTablesBuildOncePerStamp(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	src := newSource("a")

	stamp, entry, err := ratetablecache.ServedTables(t.Context(), org, bu, src.Stamp, src.Build)
	require.NoError(t, err)
	assert.Equal(t, "a", stamp)
	require.Len(t, entry.Versions, 1)

	_, again, err := ratetablecache.ServedTables(t.Context(), org, bu, src.Stamp, src.Build)
	require.NoError(t, err)
	assert.Same(t, entry, again)
	assert.EqualValues(t, 1, src.builds.Load())

	src.stamp.Store("b")
	stamp, rebuilt, err := ratetablecache.ServedTables(t.Context(), org, bu, src.Stamp, src.Build)
	require.NoError(t, err)
	assert.Equal(t, "b", stamp)
	assert.Equal(t, "ZONES_b", rebuilt.Versions[0].Shape.Code)
	assert.EqualValues(t, 2, src.builds.Load())
}

func TestServedTablesReadTheStampOncePerRequest(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	src := newSource("a")
	ctx := ratetablecache.With(t.Context())

	for range 3 {
		_, _, err := ratetablecache.ServedTables(ctx, org, bu, src.Stamp, src.Build)
		require.NoError(t, err)
	}
	assert.EqualValues(t, 1, src.stamps.Load())
	assert.Same(t, ctx, ratetablecache.With(ctx))
}

func TestServedTablesReturnTheStampError(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	src := newSource("a")
	src.err = errors.New("stamp failed")

	_, _, err := ratetablecache.ServedTables(t.Context(), org, bu, src.Stamp, src.Build)
	require.ErrorIs(t, err, src.err)
	assert.Zero(t, src.builds.Load())
}

func TestServedTablesNeverCacheAnEmptyStamp(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	src := newSource("")

	for range 2 {
		_, _, err := ratetablecache.ServedTables(t.Context(), org, bu, src.Stamp, src.Build)
		require.NoError(t, err)
	}
	assert.EqualValues(t, 2, src.builds.Load())
}

func TestInvalidateForcesARebuildOfTheServedTables(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	src := newSource("a")

	_, _, err := ratetablecache.ServedTables(t.Context(), org, bu, src.Stamp, src.Build)
	require.NoError(t, err)
	ratetablecache.Invalidate(org, bu)
	_, _, err = ratetablecache.ServedTables(t.Context(), org, bu, src.Stamp, src.Build)
	require.NoError(t, err)
	assert.EqualValues(t, 2, src.builds.Load())
}

func TestVersionsLoadOnlyWhatTheCacheMisses(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	src := newSource("a")
	matrix := pulid.MustNew("rmx_")
	first := ratetablecache.VersionKey{MatrixID: matrix, Version: 1}
	second := ratetablecache.VersionKey{MatrixID: matrix, Version: 2}

	got, err := ratetablecache.Versions(t.Context(), org, bu,
		[]ratetablecache.VersionKey{first, first}, src.Load)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.EqualValues(t, 1, got[first].Shape.Version)

	got, err = ratetablecache.Versions(t.Context(), org, bu,
		[]ratetablecache.VersionKey{first, second}, src.Load)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.EqualValues(t, 2, src.loads.Load(), "the second call loads only version 2")

	_, err = ratetablecache.Versions(t.Context(), org, bu,
		[]ratetablecache.VersionKey{first, second}, src.Load)
	require.NoError(t, err)
	assert.EqualValues(t, 2, src.loads.Load(), "a published version never changes")

	otherOrg, otherBu := tenant()
	_, err = ratetablecache.Versions(t.Context(), otherOrg, otherBu,
		[]ratetablecache.VersionKey{first}, src.Load)
	require.NoError(t, err)
	assert.EqualValues(t, 3, src.loads.Load(), "another tenant never reads this tenant's entry")
}

func TestCellsAreKeptPerVersion(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	matrix := pulid.MustNew("rmx_")
	v1 := ratetablecache.VersionKey{MatrixID: matrix, Version: 1}
	v2 := ratetablecache.VersionKey{MatrixID: matrix, Version: 2}
	keys := []string{"SE", "MW"}
	cells := []rules.TariffCell{{Value: decimal.RequireFromString("2.35")}}

	_, ok := ratetablecache.Cells(org, bu, v1, keys)
	assert.False(t, ok)

	ratetablecache.StoreCells(org, bu, v1, keys, cells)
	got, ok := ratetablecache.Cells(org, bu, v1, keys)
	require.True(t, ok)
	assert.Equal(t, cells, got)

	_, ok = ratetablecache.Cells(org, bu, v2, keys)
	assert.False(t, ok, "another version misses the cells")

	ratetablecache.Invalidate(org, bu)
	_, ok = ratetablecache.Cells(org, bu, v1, keys)
	assert.True(t, ok, "a published version's cells survive a new publish")
}
