package shipmentboardcache

import (
	"context"
	"errors"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type memoryCache struct {
	epoch    int64
	values   map[shipmentboardports.ShipmentBoardCacheKey][]byte
	epochErr error
}

func (m *memoryCache) Epoch(context.Context, pagination.TenantInfo) (int64, error) {
	return m.epoch, m.epochErr
}

func (m *memoryCache) Get(
	_ context.Context,
	key *shipmentboardports.ShipmentBoardCacheKey,
	dest any,
) (bool, error) {
	raw, ok := m.values[*key]
	if !ok {
		return false, nil
	}
	return true, sonic.Unmarshal(raw, dest)
}

func (m *memoryCache) Set(
	_ context.Context,
	key *shipmentboardports.ShipmentBoardCacheKey,
	value any,
) error {
	raw, err := sonic.Marshal(value)
	if err != nil {
		return err
	}
	m.values[*key] = raw
	return nil
}

type payload struct {
	Value int `json:"value"`
}

func TestGetOrComputeCachesPerEpoch(t *testing.T) {
	t.Parallel()

	cache := &memoryCache{values: map[shipmentboardports.ShipmentBoardCacheKey][]byte{}}
	calls := 0
	compute := func(context.Context) (*payload, error) {
		calls++
		return &payload{Value: calls}, nil
	}
	req := &Request{Section: shipmentboardports.ShipmentBoardSectionWatchlist, Timezone: "UTC"}

	first, err := GetOrCompute(t.Context(), cache, zap.NewNop(), req, compute)
	require.NoError(t, err)
	second, err := GetOrCompute(t.Context(), cache, zap.NewNop(), req, compute)
	require.NoError(t, err)
	assert.Equal(t, 1, first.Value)
	assert.Equal(t, 1, second.Value)

	cache.epoch++
	third, err := GetOrCompute(t.Context(), cache, zap.NewNop(), req, compute)
	require.NoError(t, err)
	assert.Equal(t, 2, third.Value)
}

func TestGetOrComputeFallsBackWithoutCache(t *testing.T) {
	t.Parallel()

	compute := func(context.Context) (*payload, error) { return &payload{Value: 7}, nil }
	value, err := GetOrCompute(t.Context(), nil, zap.NewNop(), &Request{}, compute)
	require.NoError(t, err)
	assert.Equal(t, 7, value.Value)

	broken := &memoryCache{epochErr: errors.New("down")}
	value, err = GetOrCompute(t.Context(), broken, zap.NewNop(), &Request{}, compute)
	require.NoError(t, err)
	assert.Equal(t, 7, value.Value)

	failing := func(context.Context) (*payload, error) { return nil, errors.New("boom") }
	_, err = GetOrCompute(t.Context(), nil, zap.NewNop(), &Request{}, failing)
	require.Error(t, err)
}

func scoped(ctx context.Context, grants readscope.Grants, err error) context.Context {
	return readscope.With(ctx, func(context.Context) (readscope.Grants, error) {
		return grants, err
	})
}

func TestGetOrComputeNeverCachesForAnOwnScopedReader(t *testing.T) {
	t.Parallel()

	cache := &memoryCache{values: map[shipmentboardports.ShipmentBoardCacheKey][]byte{}}
	req := &Request{Section: shipmentboardports.ShipmentBoardSectionWatchlist, Timezone: "UTC"}
	tenantWide := func(context.Context) (*payload, error) { return &payload{Value: 100}, nil }
	own := func(context.Context) (*payload, error) { return &payload{Value: 1}, nil }

	seeded, err := GetOrCompute(t.Context(), cache, zap.NewNop(), req, tenantWide)
	require.NoError(t, err)
	require.Equal(t, 100, seeded.Value)
	require.Len(t, cache.values, 1)

	ctx := scoped(
		t.Context(),
		readscope.Grants{"shipment": {Scope: readscope.ScopeOwn}},
		nil,
	)
	narrowed, err := GetOrCompute(ctx, cache, zap.NewNop(), req, own)
	require.NoError(t, err)
	assert.Equal(t, 1, narrowed.Value, "an own-scoped reader is not answered from the tenant's cache")

	again, err := GetOrCompute(t.Context(), cache, zap.NewNop(), req, own)
	require.NoError(t, err)
	assert.Equal(t, 100, again.Value, "an own-scoped result is never stored for the tenant")
}

func TestGetOrComputeLeavesTheCacheAloneWhenTheScopeCannotBeResolved(t *testing.T) {
	t.Parallel()

	cache := &memoryCache{values: map[shipmentboardports.ShipmentBoardCacheKey][]byte{}}
	req := &Request{Section: shipmentboardports.ShipmentBoardSectionWatchlist, Timezone: "UTC"}
	computed := func(context.Context) (*payload, error) { return &payload{Value: 7}, nil }

	ctx := scoped(t.Context(), nil, errors.New("role lookup failed"))
	value, err := GetOrCompute(ctx, cache, zap.NewNop(), req, computed)

	require.NoError(t, err)
	assert.Equal(t, 7, value.Value)
	assert.Empty(t, cache.values)
}

func TestGetOrComputeStillCachesWhenAnotherKindIsNarrowed(t *testing.T) {
	t.Parallel()

	cache := &memoryCache{values: map[shipmentboardports.ShipmentBoardCacheKey][]byte{}}
	req := &Request{Section: shipmentboardports.ShipmentBoardSectionWatchlist, Timezone: "UTC"}
	computed := func(context.Context) (*payload, error) { return &payload{Value: 7}, nil }

	ctx := scoped(
		t.Context(),
		readscope.Grants{"invoice": {Scope: readscope.ScopeOwn}},
		nil,
	)
	_, err := GetOrCompute(ctx, cache, zap.NewNop(), req, computed)

	require.NoError(t, err)
	assert.Len(t, cache.values, 1)
}
