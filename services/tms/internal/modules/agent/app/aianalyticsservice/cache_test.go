package aianalyticsservice

import (
	"context"
	"errors"
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
	epoch   int64
	entries map[int64]*aianalyticsports.Analytics
	keys    []aianalyticsports.CacheKey
	bumps   int
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
	units    int
	hoursErr error
}

func (r *refreshRepo) RefreshState(
	context.Context,
	pagination.TenantInfo,
) (*aianalytics.RefreshState, error) {
	return &aianalytics.RefreshState{RefreshedThrough: 1_790_000_000 - 60}, nil
}

func (r *refreshRepo) RefreshUnits(context.Context, aianalyticsports.RefreshRequest) (int, error) {
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

func (r *refreshRepo) SaveRefreshState(context.Context, *aianalytics.RefreshState) error {
	return nil
}

func (r *refreshRepo) Prune(context.Context, aianalyticsports.PruneRequest) error {
	return nil
}

func serviceWith(repo aianalyticsports.AnalyticsRepository, cache *memoryCache) *Service {
	return &Service{
		l:     zap.NewNop(),
		repo:  repo,
		cache: cache,
		clock: func() time.Time { return time.Unix(1_790_000_000, 0) },
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
	assert.Equal(t, "UTC", cache.keys[0].Request.Timezone, "an absent timezone is keyed as UTC")
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

	_, err := serviceWith(&refreshRepo{units: 4}, cache).Refresh(t.Context(), tenant)
	require.NoError(t, err)
	assert.Equal(t, 1, cache.bumps)

	_, err = serviceWith(&refreshRepo{}, cache).Refresh(t.Context(), tenant)
	require.NoError(t, err)
	assert.Equal(t, 1, cache.bumps, "a refresh that found nothing keeps what is cached")
}

func TestRefreshThatFailsAfterSavingRunsStillMovesTheTenantToANewEpoch(t *testing.T) {
	t.Parallel()

	cache := &memoryCache{entries: map[int64]*aianalyticsports.Analytics{}}
	tenant := pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"}
	repo := &refreshRepo{units: 3, hoursErr: errors.New("connection reset")}

	_, err := serviceWith(repo, cache).Refresh(t.Context(), tenant)

	require.Error(t, err)
	assert.Equal(t, 1, cache.bumps, "the runs already saved must not hide behind a cached answer")
}
