package resolver_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoize_WalksMovesOnceForEveryStopVariable(t *testing.T) {
	t.Parallel()

	r := laneResolver()
	entity := &LaneShipment{
		Moves: []LaneMove{{Stops: []LaneStop{
			{Location: laneLocation("Atlanta", "30301", "GA")},
			{Location: laneLocation("Miami", "33101", "FL")},
		}}},
	}
	memoized := resolver.Memoize(entity)

	origin, err := r.ResolveComputed(memoized, "computeOriginCity")
	require.NoError(t, err)
	assert.Equal(t, "Atlanta", origin)

	// A second move appears after the first walk. The memoized view is a
	// snapshot for this evaluation and must not see it; the raw entity does.
	entity.Moves = append(entity.Moves, LaneMove{Stops: []LaneStop{
		{Location: laneLocation("Tampa", "33601", "FL")},
	}})

	destination, err := r.ResolveComputed(memoized, "computeDestinationCity")
	require.NoError(t, err)
	assert.Equal(t, "Miami", destination, "the memo walked Moves once")

	stops, err := r.ResolveComputed(memoized, "computeTotalStops")
	require.NoError(t, err)
	assert.EqualValues(t, 2, stops)

	fresh, err := r.ResolveComputed(entity, "computeDestinationCity")
	require.NoError(t, err)
	assert.Equal(t, "Tampa", fresh, "without the memo every call walks again")
}

func TestMemoize_IsTransparentToFieldReads(t *testing.T) {
	t.Parallel()

	r := laneResolver()
	entity := &CollectionShipment{
		Commodities: []CollectionShipmentCommodity{{Weight: 500, Pieces: 2}},
	}

	raw, err := r.ResolveComputed(entity, "computeTotalWeight")
	require.NoError(t, err)
	memoized, err := r.ResolveComputed(resolver.Memoize(entity), "computeTotalWeight")
	require.NoError(t, err)
	assert.Equal(t, raw, memoized)

	assert.Same(t, entity, resolver.Unwrap(resolver.Memoize(entity)))
	assert.Same(t, entity, resolver.Unwrap(entity), "unwrapping a plain entity is a no-op")
}
