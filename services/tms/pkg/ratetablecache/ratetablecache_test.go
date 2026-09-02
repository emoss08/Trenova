package ratetablecache_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLookup struct{ formulatemplatetypes.RateTableLookup }

type stampedSource struct {
	stamp  atomic.Value
	stamps atomic.Int32
	builds atomic.Int32
	err    error
}

func newStampedSource(stamp string) *stampedSource {
	s := &stampedSource{}
	s.stamp.Store(stamp)
	return s
}

func (s *stampedSource) Stamp(context.Context) (string, error) {
	s.stamps.Add(1)
	if s.err != nil {
		return "", s.err
	}
	return s.stamp.Load().(string), nil //nolint:errcheck // always a string
}

func (s *stampedSource) Build(context.Context) (formulatemplatetypes.RateTableLookup, error) {
	s.builds.Add(1)
	return &fakeLookup{}, nil
}

func TestGetStamped_ReusesAcrossContextsWhileTheStampHolds(t *testing.T) {
	t.Parallel()

	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")
	source := newStampedSource("3:7:1700000000")

	first, err := ratetablecache.GetStamped(t.Context(), org, bu, source.Stamp, source.Build)
	require.NoError(t, err)
	second, err := ratetablecache.GetStamped(t.Context(), org, bu, source.Stamp, source.Build)
	require.NoError(t, err)

	assert.Same(t, first, second, "a second request in a new context reuses the built lookup")
	assert.EqualValues(t, 1, source.builds.Load(), "the expensive read happened once")
	assert.EqualValues(t, 2, source.stamps.Load(), "each new context asks the cheap stamp")
}

func TestGetStamped_RebuildsWhenTheStampMoves(t *testing.T) {
	t.Parallel()

	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")
	source := newStampedSource("3:7:1700000000")

	first, err := ratetablecache.GetStamped(t.Context(), org, bu, source.Stamp, source.Build)
	require.NoError(t, err)

	source.stamp.Store("3:8:1700000500")
	second, err := ratetablecache.GetStamped(t.Context(), org, bu, source.Stamp, source.Build)
	require.NoError(t, err)

	assert.NotSame(t, first, second, "a changed matrix means a fresh lookup")
	assert.EqualValues(t, 2, source.builds.Load())
}

func TestGetStamped_InvalidateForcesARebuild(t *testing.T) {
	t.Parallel()

	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")
	source := newStampedSource("stable")

	_, err := ratetablecache.GetStamped(t.Context(), org, bu, source.Stamp, source.Build)
	require.NoError(t, err)

	ratetablecache.Invalidate(org, bu)

	_, err = ratetablecache.GetStamped(t.Context(), org, bu, source.Stamp, source.Build)
	require.NoError(t, err)
	assert.EqualValues(t, 2, source.builds.Load(), "the write path can evict without a stamp change")
}

func TestGetStamped_PerContextMemoWinsOverTheStamp(t *testing.T) {
	t.Parallel()

	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")
	source := newStampedSource("stable")
	ctx := ratetablecache.With(t.Context())

	_, err := ratetablecache.GetStamped(ctx, org, bu, source.Stamp, source.Build)
	require.NoError(t, err)
	_, err = ratetablecache.GetStamped(ctx, org, bu, source.Stamp, source.Build)
	require.NoError(t, err)

	assert.EqualValues(t, 1, source.stamps.Load(), "within one batch even the stamp is asked once")
	assert.EqualValues(t, 1, source.builds.Load())
}

func TestGetStamped_EmptyOrFailingStampSkipsTheProcessCache(t *testing.T) {
	t.Parallel()

	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")

	unstamped := newStampedSource("")
	_, err := ratetablecache.GetStamped(t.Context(), org, bu, unstamped.Stamp, unstamped.Build)
	require.NoError(t, err)
	_, err = ratetablecache.GetStamped(t.Context(), org, bu, unstamped.Stamp, unstamped.Build)
	require.NoError(t, err)
	assert.EqualValues(t, 2, unstamped.builds.Load(), "no stamp means no sharing across requests")

	failing := newStampedSource("x")
	failing.err = errors.New("stamp query failed")
	_, err = ratetablecache.GetStamped(t.Context(), org, bu, failing.Stamp, failing.Build)
	require.NoError(t, err, "a failing stamp degrades to a plain build")
	assert.EqualValues(t, 1, failing.builds.Load())
}
