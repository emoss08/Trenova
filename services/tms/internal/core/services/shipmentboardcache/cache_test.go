package shipmentboardcache

import (
	"context"
	"errors"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type memoryCache struct {
	epoch    int64
	values   map[repositories.ShipmentBoardCacheKey][]byte
	epochErr error
}

func (m *memoryCache) Epoch(context.Context, pagination.TenantInfo) (int64, error) {
	return m.epoch, m.epochErr
}

func (m *memoryCache) Get(
	_ context.Context,
	key *repositories.ShipmentBoardCacheKey,
	dest any,
) (bool, error) {
	raw, ok := m.values[*key]
	if !ok {
		return false, nil
	}
	return true, sonic.Unmarshal(raw, dest)
}

func (m *memoryCache) Set(_ context.Context, key *repositories.ShipmentBoardCacheKey, value any) error {
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

	cache := &memoryCache{values: map[repositories.ShipmentBoardCacheKey][]byte{}}
	calls := 0
	compute := func(context.Context) (*payload, error) {
		calls++
		return &payload{Value: calls}, nil
	}
	req := Request{Section: repositories.ShipmentBoardSectionWatchlist, Timezone: "UTC"}

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
	value, err := GetOrCompute(t.Context(), nil, zap.NewNop(), Request{}, compute)
	require.NoError(t, err)
	assert.Equal(t, 7, value.Value)

	broken := &memoryCache{epochErr: errors.New("down")}
	value, err = GetOrCompute(t.Context(), broken, zap.NewNop(), Request{}, compute)
	require.NoError(t, err)
	assert.Equal(t, 7, value.Value)

	failing := func(context.Context) (*payload, error) { return nil, errors.New("boom") }
	_, err = GetOrCompute(t.Context(), nil, zap.NewNop(), Request{}, failing)
	require.Error(t, err)
}
