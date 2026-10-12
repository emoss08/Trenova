package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		key    string
		tier   Tier
		ok     bool
	}{
		{"drivers list", "GET", "/fleet/drivers", "GET /fleet/drivers", TierLevelTwo, true},
		{"drivers create", "POST", "/fleet/drivers", "POST /fleet/drivers", TierLevelOne, true},
		{
			"driver by numeric id",
			"GET",
			"/fleet/drivers/1654973",
			"GET /fleet/drivers/{id}",
			TierLegacyOne,
			true,
		},
		{
			"driver patch by external id",
			"patch",
			"/fleet/drivers/payrollId:ABC",
			"PATCH /fleet/drivers/{id}",
			TierLevelOne,
			true,
		},
		{
			"literal wins over wildcard",
			"GET",
			"/fleet/vehicles/stats",
			"GET /fleet/vehicles/stats",
			TierLegacyTwo,
			true,
		},
		{
			"vehicle by id",
			"GET",
			"/fleet/vehicles/281474977075805",
			"GET /fleet/vehicles/{id}",
			TierLegacyOne,
			true,
		},
		{
			"driver tachograph history",
			"GET",
			"/fleet/drivers/tachograph-files/history",
			"GET /fleet/drivers/tachograph-files/history",
			TierLevelTwo,
			true,
		},
		{"hos logs", "GET", "/fleet/hos/logs", "GET /fleet/hos/logs", TierLevelTwo, true},
		{
			"hos daily logs trailing slash",
			"GET",
			"/fleet/hos/daily-logs/",
			"GET /fleet/hos/daily-logs",
			TierLevelTwo,
			true,
		},
		{"hos clocks", "GET", "/fleet/hos/clocks", "GET /fleet/hos/clocks", TierLegacyOne, true},
		{"assets list", "GET", "/assets", "GET /assets", TierUnlisted, true},
		{"legacy messages", "POST", "/v1/fleet/messages", "POST /v1/fleet/messages", TierLevelOne, true},
		{"query ignored", "GET", "/fleet/routes?limit=5", "GET /fleet/routes", TierLevelTwo, true},
		{"unknown path", "GET", "/fleet/unknown", "", TierUnlisted, false},
		{"unknown method", "PUT", "/fleet/drivers", "", TierUnlisted, false},
		{"empty id segment", "GET", "/fleet/drivers//", "GET /fleet/drivers", TierLevelTwo, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			endpoint, ok := Classify(tt.method, tt.path)
			require.Equal(t, tt.ok, ok)
			if !ok {
				return
			}
			assert.Equal(t, tt.key, endpoint.Key)
			assert.Equal(t, tt.tier, endpoint.Tier)
		})
	}
}

func TestTierLimits(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 100.0/60.0, float64(TierLevelOne.Limit()), 1e-9)
	assert.InDelta(t, 5.0, float64(TierLevelTwo.Limit()), 1e-9)
	assert.InDelta(t, 10.0, float64(TierLevelThree.Limit()), 1e-9)
	assert.InDelta(t, 25.0, float64(TierLegacyOne.Limit()), 1e-9)
	assert.InDelta(t, 50.0, float64(TierLegacyTwo.Limit()), 1e-9)
	assert.Equal(t, rate.Inf, TierUnlisted.Limit())
}

func TestSetPacesLevelTwoEndpoint(t *testing.T) {
	t.Parallel()

	set := NewSet()
	started := time.Now()
	for range 4 {
		require.NoError(t, set.Wait(t.Context(), "GET", "/fleet/hos/logs"))
	}
	assert.GreaterOrEqual(t, time.Since(started), 590*time.Millisecond)
}

func TestSetKeepsEndpointsIndependent(t *testing.T) {
	t.Parallel()

	set := NewSet()
	require.NoError(t, set.Wait(t.Context(), "GET", "/fleet/hos/logs"))

	started := time.Now()
	require.NoError(t, set.Wait(t.Context(), "GET", "/fleet/hos/daily-logs"))
	require.NoError(t, set.Wait(t.Context(), "GET", "/fleet/drivers/123"))
	require.NoError(t, set.Wait(t.Context(), "GET", "/fleet/drivers/456"))
	assert.Less(t, time.Since(started), 150*time.Millisecond)
}

func TestSetSharesBucketAcrossIDs(t *testing.T) {
	t.Parallel()

	set := NewSet()
	started := time.Now()
	require.NoError(t, set.Wait(t.Context(), "PATCH", "/fleet/drivers/1"))
	require.NoError(t, set.Wait(t.Context(), "PATCH", "/fleet/drivers/2"))
	assert.GreaterOrEqual(t, time.Since(started), 590*time.Millisecond)
}

func TestSetWaitHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	set := NewSet()
	require.NoError(t, set.Wait(t.Context(), "POST", "/fleet/drivers"))

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	err := set.Wait(ctx, "POST", "/fleet/drivers")
	require.Error(t, err)
}

func TestSetPenalizeBlocksEndpoint(t *testing.T) {
	t.Parallel()

	set := NewSet()
	set.Penalize("GET", "/fleet/hos/violations", 300*time.Millisecond)
	set.Penalize("GET", "/fleet/hos/violations", 100*time.Millisecond)

	started := time.Now()
	require.NoError(t, set.Wait(t.Context(), "GET", "/fleet/hos/violations"))
	assert.GreaterOrEqual(t, time.Since(started), 280*time.Millisecond)

	other := time.Now()
	require.NoError(t, set.Wait(t.Context(), "GET", "/assets"))
	assert.Less(t, time.Since(other), 100*time.Millisecond)
}

func TestSetPenalizeIgnoresNonPositiveAndUnknown(t *testing.T) {
	t.Parallel()

	set := NewSet()
	set.Penalize("GET", "/fleet/drivers", 0)
	set.Penalize("GET", "/fleet/drivers", -time.Second)
	set.Penalize("GET", "/not/a/samsara/path", time.Hour)

	started := time.Now()
	require.NoError(t, set.Wait(t.Context(), "GET", "/fleet/drivers"))
	require.NoError(t, set.Wait(t.Context(), "GET", "/not/a/samsara/path"))
	assert.Less(t, time.Since(started), 100*time.Millisecond)
}

func TestSetPenaltyWaitHonorsContext(t *testing.T) {
	t.Parallel()

	set := NewSet()
	set.Penalize("GET", "/fleet/drivers", time.Hour)

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	err := set.Wait(ctx, "GET", "/fleet/drivers")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSetConcurrentWaitersStayWithinRate(t *testing.T) {
	t.Parallel()

	set := NewSet()
	var wg sync.WaitGroup
	started := time.Now()
	for range 6 {
		wg.Go(func() {
			assert.NoError(t, set.Wait(t.Context(), "GET", "/fleet/routes"))
		})
	}
	wg.Wait()
	assert.GreaterOrEqual(t, time.Since(started), 990*time.Millisecond)
}

func TestRegistrySharesSetPerCredential(t *testing.T) {
	t.Parallel()

	registry := NewRegistry(time.Hour)
	first := registry.For("https://api.samsara.com", "token-a")
	second := registry.For("https://api.samsara.com", "token-a")
	other := registry.For("https://api.samsara.com", "token-b")
	eu := registry.For("https://api.eu.samsara.com", "token-a")

	assert.Same(t, first, second)
	assert.NotSame(t, first, other)
	assert.NotSame(t, first, eu)
	assert.Equal(t, 3, registry.Len())
}

func TestRegistryEvictsIdleSets(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}
	advance := func(d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		now = now.Add(d)
	}

	registry := newRegistry(time.Minute, clock)
	stale := registry.For("https://api.samsara.com", "stale")
	advance(30 * time.Second)
	active := registry.For("https://api.samsara.com", "active")
	advance(45 * time.Second)
	active.touch()

	_ = registry.For("https://api.samsara.com", "fresh")
	assert.Equal(t, 2, registry.Len())
	assert.NotSame(t, stale, registry.For("https://api.samsara.com", "stale"))
	assert.Same(t, active, registry.For("https://api.samsara.com", "active"))
}

func TestSharedRegistryReturnsSameSet(t *testing.T) {
	t.Parallel()

	assert.Same(
		t,
		Shared("https://api.samsara.com", "shared-token"),
		Shared("https://api.samsara.com", "shared-token"),
	)
}
