package aianalyticsservice

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type memoryCache struct {
	epoch         int64
	entries       map[int64]*aianalyticsports.Analytics
	keys          []aianalyticsports.CacheKey
	bumps         int
	palette       []aianalyticsports.DeploymentKey
	paletteWrites int
}

func (c *memoryCache) Epoch(context.Context, pagination.TenantInfo) (int64, error) {
	return c.epoch, nil
}

func (c *memoryCache) BumpEpoch(context.Context, pagination.TenantInfo) error {
	c.epoch++
	c.bumps++
	return nil
}

func (c *memoryCache) Get(
	_ context.Context,
	key *aianalyticsports.CacheKey,
	dest *aianalyticsports.Analytics,
) (bool, error) {
	c.keys = append(c.keys, *key)
	cached, ok := c.entries[key.Epoch]
	if !ok {
		return false, nil
	}
	*dest = *cached
	return true, nil
}

func (c *memoryCache) Palette(
	context.Context,
	pagination.TenantInfo,
	int64,
) ([]aianalyticsports.DeploymentKey, bool, error) {
	if c.palette == nil {
		return nil, false, nil
	}
	return c.palette, true, nil
}

func (c *memoryCache) SetPalette(
	_ context.Context,
	_ pagination.TenantInfo,
	_ int64,
	keys []aianalyticsports.DeploymentKey,
) error {
	c.palette = keys
	c.paletteWrites++
	return nil
}

func (c *memoryCache) Set(
	_ context.Context,
	key *aianalyticsports.CacheKey,
	value *aianalyticsports.Analytics,
) error {
	c.entries[key.Epoch] = value
	return nil
}

type refreshRepo struct {
	aianalyticsports.AnalyticsRepository
	state    *aianalytics.RefreshState
	noState  bool
	idle     bool
	units    int
	hoursErr error
	chunks   []aianalyticsports.RefreshRequest
	saved    []aianalytics.RefreshState
	prunes   int
}

func (r *refreshRepo) RefreshState(
	context.Context,
	pagination.TenantInfo,
) (*aianalytics.RefreshState, error) {
	if r.noState {
		return nil, nil
	}
	if r.state != nil {
		return r.state, nil
	}
	return &aianalytics.RefreshState{RefreshedThrough: 1_790_000_000 - 60}, nil
}

func (r *refreshRepo) HasActivity(context.Context, aianalyticsports.RefreshRequest) (bool, error) {
	return !r.idle, nil
}

func (r *refreshRepo) RefreshUnits(
	_ context.Context,
	req aianalyticsports.RefreshRequest,
) (int, error) {
	r.chunks = append(r.chunks, req)
	return r.units, nil
}

func (r *refreshRepo) RefreshProviderHours(
	context.Context,
	aianalyticsports.RefreshRequest,
) (int, error) {
	return 0, r.hoursErr
}

func (r *refreshRepo) UnkeyedModels(
	context.Context,
	aianalyticsports.ModelKeyRequest,
) ([]string, error) {
	return nil, nil
}

func (r *refreshRepo) SaveRefreshState(_ context.Context, state *aianalytics.RefreshState) error {
	r.saved = append(r.saved, *state)
	return nil
}

func (r *refreshRepo) Prune(context.Context, aianalyticsports.PruneRequest) error {
	r.prunes++
	return nil
}

func serviceWith(repo aianalyticsports.AnalyticsRepository, cache *memoryCache) *Service {
	return &Service{
		l:         zap.NewNop(),
		reader:    repo,
		refresher: repo,
		cache:     cache,
		clock:     func() time.Time { return time.Unix(1_790_000_000, 0) },
	}
}

func TestAnalyticsAreServedFromTheCacheForTheTenantsCurrentEpoch(t *testing.T) {
	t.Parallel()

	cached := &aianalyticsports.Analytics{Window: aianalytics.WindowWeek, Since: 42}
	cache := &memoryCache{epoch: 3, entries: map[int64]*aianalyticsports.Analytics{3: cached}}
	svc := serviceWith(nil, cache)

	got, err := svc.Analytics(t.Context(), &aianalyticsports.AnalyticsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		Window:     aianalytics.WindowWeek,
		CompareBy:  aianalytics.CompareByModel,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(42), got.Since)
	require.Len(t, cache.keys, 1)
	assert.Equal(t, int64(3), cache.keys[0].Epoch)
	assert.Equal(
		t,
		aianalytics.BucketsDigest(
			aianalytics.BucketStarts(aianalytics.WindowWeek, time.UTC, 1_790_000_000),
		),
		cache.keys[0].Buckets,
		"an absent timezone buckets in UTC",
	)
}

func TestZonesThatShareBucketBoundariesShareACacheEntry(t *testing.T) {
	t.Parallel()

	keyFor := func(zone string) uint64 {
		cache := &memoryCache{entries: map[int64]*aianalyticsports.Analytics{
			0: {Window: aianalytics.WindowDay},
		}}
		_, err := serviceWith(nil, cache).Analytics(t.Context(), &aianalyticsports.AnalyticsRequest{
			TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
			Window:     aianalytics.WindowDay,
			CompareBy:  aianalytics.CompareByModel,
			Timezone:   zone,
		})
		require.NoError(t, err)
		require.Len(t, cache.keys, 1)
		return cache.keys[0].Buckets
	}

	assert.Equal(t, keyFor("UTC"), keyFor("Africa/Abidjan"), "both sit at UTC+0 all year")
	assert.NotEqual(t, keyFor("UTC"), keyFor("Asia/Kolkata"), "a half-hour zone has other hours")
}

func TestAnOverlongTimezoneIsRefusedBeforeAnythingIsRead(t *testing.T) {
	t.Parallel()

	_, err := serviceWith(
		nil,
		&memoryCache{},
	).Analytics(t.Context(), &aianalyticsports.AnalyticsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		Window:     aianalytics.WindowDay,
		CompareBy:  aianalytics.CompareByModel,
		Timezone:   strings.Repeat("Z", 200),
	})

	require.Error(t, err)
}

type paletteRepo struct {
	aianalyticsports.AnalyticsRepository
	reads int
}

func (r *paletteRepo) Palette(
	context.Context,
	pagination.TenantInfo,
	int64,
) ([]aianalyticsports.DeploymentKey, error) {
	r.reads++
	return []aianalyticsports.DeploymentKey{
		{ModelKey: "llama-3.3-70b", ProviderID: "aip_groq"},
	}, nil
}

func TestThePaletteIsReadOncePerEpochWhateverTheScope(t *testing.T) {
	t.Parallel()

	repo := &paletteRepo{}
	cache := &memoryCache{entries: map[int64]*aianalyticsports.Analytics{}}
	svc := serviceWith(repo, cache)
	tenant := pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"}
	key := &aianalyticsports.CacheKey{Epoch: 4}

	first, err := svc.palette(t.Context(), tenant, key, 1_790_000_000)
	require.NoError(t, err)
	second, err := svc.palette(t.Context(), tenant, key, 1_790_000_300)
	require.NoError(t, err)

	assert.Equal(t, 1, repo.reads)
	assert.Equal(t, 1, cache.paletteWrites)
	assert.Equal(t, first, second)

	_, err = svc.palette(t.Context(), tenant, nil, 1_790_000_600)
	require.NoError(t, err)
	assert.Equal(t, 2, repo.reads, "without a cache the palette is read from Postgres")
}

func TestAnalyticsAreCachedPerLanguageBecauseTheSentencesAreTranslated(t *testing.T) {
	t.Parallel()

	cache := &memoryCache{epoch: 1, entries: map[int64]*aianalyticsports.Analytics{
		1: {Window: aianalytics.WindowWeek},
	}}
	svc := serviceWith(nil, cache)

	_, err := svc.Analytics(
		i18n.WithLocale(t.Context(), i18n.ES),
		&aianalyticsports.AnalyticsRequest{
			TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
			Window:     aianalytics.WindowWeek,
			CompareBy:  aianalytics.CompareByModel,
		},
	)

	require.NoError(t, err)
	require.Len(t, cache.keys, 1)
	assert.Equal(t, "es", cache.keys[0].Locale)
}

func TestRefreshThatWroteRowsMovesTheTenantToANewEpoch(t *testing.T) {
	t.Parallel()

	cache := &memoryCache{entries: map[int64]*aianalyticsports.Analytics{}}
	tenant := pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"}

	_, err := serviceWith(&refreshRepo{units: 4}, cache).Refresh(t.Context(), tenant, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, cache.bumps)

	_, err = serviceWith(&refreshRepo{}, cache).Refresh(t.Context(), tenant, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, cache.bumps, "a refresh that found nothing keeps what is cached")
}

func TestRefreshThatFailsAfterSavingRunsStillMovesTheTenantToANewEpoch(t *testing.T) {
	t.Parallel()

	cache := &memoryCache{entries: map[int64]*aianalyticsports.Analytics{}}
	tenant := pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"}
	repo := &refreshRepo{units: 3, hoursErr: errors.New("connection reset")}

	_, err := serviceWith(repo, cache).Refresh(t.Context(), tenant, nil)

	require.Error(t, err)
	assert.Equal(t, 1, cache.bumps, "the runs already saved must not hide behind a cached answer")
}

func TestAQuietTenantIsMarkedCaughtUpWithoutRebuildingAnything(t *testing.T) {
	t.Parallel()

	cache := &memoryCache{entries: map[int64]*aianalyticsports.Analytics{}}
	tenant := pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"}
	repo := &refreshRepo{idle: true}

	result, err := serviceWith(repo, cache).Refresh(t.Context(), tenant, nil)

	require.NoError(t, err)
	assert.Empty(t, repo.chunks)
	assert.Equal(t, []aianalytics.RefreshState{{
		OrganizationID:   "org_1",
		BusinessUnitID:   "bu_1",
		RefreshedThrough: 1_790_000_000,
		RefreshedAt:      1_790_000_000,
	}}, repo.saved)
	assert.Equal(t, int64(1_790_000_000), result.RefreshedThrough)
	assert.True(t, result.CaughtUp)
	assert.Zero(t, cache.bumps)
}

func TestRetentionIsPrunedOncePerHour(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"}
	now := int64(1_790_000_000)
	hourStart := now - now%3600

	tests := []struct {
		name   string
		state  *aianalytics.RefreshState
		prunes int
	}{
		{name: "first refresh", prunes: 1},
		{
			name:   "already refreshed this hour",
			state:  &aianalytics.RefreshState{RefreshedThrough: now - 60, RefreshedAt: hourStart},
			prunes: 0,
		},
		{
			name: "last refreshed in the previous hour",
			state: &aianalytics.RefreshState{
				RefreshedThrough: now - 60,
				RefreshedAt:      hourStart - 1,
			},
			prunes: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &refreshRepo{state: tt.state, noState: tt.state == nil, idle: true}
			cache := &memoryCache{entries: map[int64]*aianalyticsports.Analytics{}}

			_, err := serviceWith(repo, cache).Refresh(t.Context(), tenant, nil)

			require.NoError(t, err)
			assert.Equal(t, tt.prunes, repo.prunes)
		})
	}
}

func TestRefreshReportsProgressAfterEveryChunk(t *testing.T) {
	t.Parallel()

	now := int64(1_790_000_000)
	repo := &refreshRepo{
		state: &aianalytics.RefreshState{
			RefreshedThrough: now - 2*ChunkSeconds + OverlapSeconds,
			RefreshedAt:      now,
		},
		units: 1,
	}
	cache := &memoryCache{entries: map[int64]*aianalyticsports.Analytics{}}
	var reported []int64

	result, err := serviceWith(repo, cache).Refresh(
		t.Context(),
		pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		func(through int64) { reported = append(reported, through) },
	)

	require.NoError(t, err)
	assert.Equal(t, []int64{now - ChunkSeconds, now}, reported)
	assert.Len(t, repo.chunks, 2)
	assert.True(t, result.CaughtUp)
	assert.Equal(t, 1, cache.bumps)
}
