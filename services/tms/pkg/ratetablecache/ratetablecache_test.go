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

func (s *source) Build(_ context.Context, stamp string) (*ratetablecache.Entry, error) {
	s.builds.Add(1)
	return &ratetablecache.Entry{
		Tables: []rules.TariffTable{{Code: "ZONES_" + stamp}},
		Index:  []ratetablecache.Table{{MatrixID: pulid.MustNew("rmx_"), Code: "ZONES_" + stamp}},
	}, nil
}

func tenant() (pulid.ID, pulid.ID) {
	return pulid.MustNew("org_"), pulid.MustNew("bu_")
}

func TestTablesBuildsOncePerStamp(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	src := newSource("a")

	stamp, entry, err := ratetablecache.Tables(t.Context(), org, bu, src.Stamp, src.Build)
	require.NoError(t, err)
	assert.Equal(t, "a", stamp)
	require.Len(t, entry.Tables, 1)

	_, again, err := ratetablecache.Tables(t.Context(), org, bu, src.Stamp, src.Build)
	require.NoError(t, err)
	assert.Same(t, entry, again)
	assert.EqualValues(t, 1, src.builds.Load())

	src.stamp.Store("b")
	stamp, rebuilt, err := ratetablecache.Tables(t.Context(), org, bu, src.Stamp, src.Build)
	require.NoError(t, err)
	assert.Equal(t, "b", stamp)
	assert.Equal(t, "ZONES_b", rebuilt.Tables[0].Code)
	assert.EqualValues(t, 2, src.builds.Load())
}

func TestTablesReadsTheStampOncePerRequest(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	src := newSource("a")
	ctx := ratetablecache.With(t.Context())

	for range 3 {
		_, _, err := ratetablecache.Tables(ctx, org, bu, src.Stamp, src.Build)
		require.NoError(t, err)
	}
	assert.EqualValues(t, 1, src.stamps.Load())
	assert.Same(t, ctx, ratetablecache.With(ctx))
}

func TestTablesReturnsTheStampError(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	src := newSource("a")
	src.err = errors.New("stamp failed")

	_, _, err := ratetablecache.Tables(t.Context(), org, bu, src.Stamp, src.Build)
	require.ErrorIs(t, err, src.err)
	assert.Zero(t, src.builds.Load())
}

func TestTablesNeverCachesAnEmptyStamp(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	src := newSource("")

	for range 2 {
		_, _, err := ratetablecache.Tables(t.Context(), org, bu, src.Stamp, src.Build)
		require.NoError(t, err)
	}
	assert.EqualValues(t, 2, src.builds.Load())
}

func TestEntryFindsATableByCode(t *testing.T) {
	t.Parallel()

	entry := &ratetablecache.Entry{Index: []ratetablecache.Table{{Code: "A"}, {Code: "B"}}}
	table, index, ok := entry.Table("B")
	require.True(t, ok)
	assert.Equal(t, 1, index)
	assert.Equal(t, "B", table.Code)

	_, index, ok = entry.Table("C")
	assert.False(t, ok)
	assert.Equal(t, -1, index)
}

func TestCellsAreKeptPerStampAndDroppedOnInvalidate(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	keys := []string{"SE", "MW"}
	cells := []rules.TariffCell{{Value: decimal.RequireFromString("2.35")}}

	_, ok := ratetablecache.Cells(org, bu, "a", "ZONES", keys)
	assert.False(t, ok)

	ratetablecache.StoreCells(org, bu, "a", "ZONES", keys, cells)
	got, ok := ratetablecache.Cells(org, bu, "a", "ZONES", keys)
	require.True(t, ok)
	assert.Equal(t, cells, got)

	_, ok = ratetablecache.Cells(org, bu, "b", "ZONES", keys)
	assert.False(t, ok, "a new stamp misses the old cells")

	ratetablecache.Invalidate(org, bu)
	_, ok = ratetablecache.Cells(org, bu, "a", "ZONES", keys)
	assert.False(t, ok, "invalidation drops the tenant's cells")
}

func TestCellsIgnoreAnEmptyStamp(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	ratetablecache.StoreCells(org, bu, "", "ZONES", nil, []rules.TariffCell{{}})
	_, ok := ratetablecache.Cells(org, bu, "", "ZONES", nil)
	assert.False(t, ok)
}

func TestInvalidateForcesARebuild(t *testing.T) {
	t.Parallel()

	org, bu := tenant()
	src := newSource("a")

	_, _, err := ratetablecache.Tables(t.Context(), org, bu, src.Stamp, src.Build)
	require.NoError(t, err)
	ratetablecache.Invalidate(org, bu)
	_, _, err = ratetablecache.Tables(t.Context(), org, bu, src.Stamp, src.Build)
	require.NoError(t, err)
	assert.EqualValues(t, 2, src.builds.Load())
}
