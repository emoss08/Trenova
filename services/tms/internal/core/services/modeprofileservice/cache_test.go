package modeprofileservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errCacheMiss = errors.New("cache miss")

type fakeProfileRepo struct {
	profiles []*modeprofile.Profile
	err      error
	calls    int
}

func (f *fakeProfileRepo) GetActiveProfiles(
	_ context.Context,
	_ pagination.TenantInfo,
) ([]*modeprofile.Profile, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.profiles, nil
}

func (f *fakeProfileRepo) List(
	_ context.Context,
	_ *repositories.ListModeProfilesRequest,
) (*pagination.ListResult[*modeprofile.Profile], error) {
	return nil, nil
}

func (f *fakeProfileRepo) GetByID(
	_ context.Context,
	_ *repositories.GetModeProfileByIDRequest,
) (*modeprofile.Profile, error) {
	return nil, nil
}

func (f *fakeProfileRepo) Create(
	_ context.Context,
	entity *modeprofile.Profile,
) (*modeprofile.Profile, error) {
	return entity, nil
}

func (f *fakeProfileRepo) Update(
	_ context.Context,
	entity *modeprofile.Profile,
) (*modeprofile.Profile, error) {
	return entity, nil
}

type fakeCacheRepo struct {
	profiles      []*modeprofile.Profile
	getErr        error
	setErr        error
	getCalls      int
	setCalls      int
	invalidations int
	lastSet       []*modeprofile.Profile
}

func (f *fakeCacheRepo) GetActiveProfiles(
	_ context.Context,
	_ pagination.TenantInfo,
) ([]*modeprofile.Profile, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.profiles, nil
}

func (f *fakeCacheRepo) SetActiveProfiles(
	_ context.Context,
	_ pagination.TenantInfo,
	profiles []*modeprofile.Profile,
) error {
	f.setCalls++
	f.lastSet = profiles
	return f.setErr
}

func (f *fakeCacheRepo) Invalidate(_ context.Context, _ pagination.TenantInfo) error {
	f.invalidations++
	return nil
}

func newCachingService(
	repo *fakeProfileRepo,
	cache *fakeCacheRepo,
) *service {
	return &service{
		profileRepo: repo,
		cacheRepo:   cache,
		l:           zap.NewNop(),
	}
}

func TestResolve_CacheHitSkipsDatabase(t *testing.T) {
	repo := &fakeProfileRepo{profiles: []*modeprofile.Profile{coreProfile("DB", true)}}
	cache := &fakeCacheRepo{profiles: []*modeprofile.Profile{coreProfile("CACHED", true)}}

	svc := newCachingService(repo, cache)

	policy, err := svc.Resolve(t.Context(), &services.ResolveModeProfileRequest{})

	require.NoError(t, err)
	require.NotNil(t, policy)
	assert.Equal(t, "CACHED", policy.ProfileCode)
	assert.Equal(t, 0, repo.calls)
	assert.Equal(t, 0, cache.setCalls)
}

func TestResolve_CacheMissLoadsFromDatabaseAndPopulatesCache(t *testing.T) {
	profiles := []*modeprofile.Profile{coreProfile("OTR", true)}
	repo := &fakeProfileRepo{profiles: profiles}
	cache := &fakeCacheRepo{getErr: errCacheMiss}

	svc := newCachingService(repo, cache)

	policy, err := svc.Resolve(t.Context(), &services.ResolveModeProfileRequest{})

	require.NoError(t, err)
	require.NotNil(t, policy)
	assert.Equal(t, "OTR", policy.ProfileCode)
	assert.Equal(t, 1, repo.calls)
	assert.Equal(t, 1, cache.setCalls)
	assert.Equal(t, profiles, cache.lastSet)
}

func TestResolve_CacheWriteFailureStillResolves(t *testing.T) {
	repo := &fakeProfileRepo{profiles: []*modeprofile.Profile{coreProfile("OTR", true)}}
	cache := &fakeCacheRepo{getErr: errCacheMiss, setErr: errors.New("redis down")}

	svc := newCachingService(repo, cache)

	policy, err := svc.Resolve(t.Context(), &services.ResolveModeProfileRequest{})

	require.NoError(t, err)
	require.NotNil(t, policy)
	assert.Equal(t, "OTR", policy.ProfileCode)
}

func TestResolve_DatabaseErrorPropagates(t *testing.T) {
	dbErr := errors.New("connection refused")
	repo := &fakeProfileRepo{err: dbErr}
	cache := &fakeCacheRepo{getErr: errCacheMiss}

	svc := newCachingService(repo, cache)

	policy, err := svc.Resolve(t.Context(), &services.ResolveModeProfileRequest{})

	require.ErrorIs(t, err, dbErr)
	assert.Nil(t, policy)
}

func TestResolve_WithoutCacheRepositoryFallsBackToDatabase(t *testing.T) {
	repo := &fakeProfileRepo{profiles: []*modeprofile.Profile{coreProfile("OTR", true)}}

	svc := &service{profileRepo: repo, l: zap.NewNop()}

	policy, err := svc.Resolve(t.Context(), &services.ResolveModeProfileRequest{})

	require.NoError(t, err)
	require.NotNil(t, policy)
	assert.Equal(t, 1, repo.calls)
}

func TestResolve_EmptyProfileSetResolvesToNil(t *testing.T) {
	repo := &fakeProfileRepo{profiles: nil}
	cache := &fakeCacheRepo{getErr: errCacheMiss}

	svc := newCachingService(repo, cache)

	policy, err := svc.Resolve(t.Context(), &services.ResolveModeProfileRequest{})

	require.NoError(t, err)
	assert.Nil(t, policy)
}

func TestSortCandidates_OrdersByPriorityThenSpecificityThenAge(t *testing.T) {
	customerID := pulid.MustNew("cus_")

	lowPriority := coreProfile("LOW", true)
	lowPriority.Priority = 1
	lowPriority.CreatedAt = 10

	highPriority := coreProfile("HIGH", false)
	highPriority.Priority = 9
	highPriority.CustomerID = &customerID
	highPriority.CreatedAt = 20

	samePriorityMoreSpecific := coreProfile("SPECIFIC", false)
	samePriorityMoreSpecific.Priority = 1
	samePriorityMoreSpecific.SpecificityScore = 16
	samePriorityMoreSpecific.CustomerID = &customerID
	samePriorityMoreSpecific.CreatedAt = 30

	sameEverythingOlder := coreProfile("OLDER", true)
	sameEverythingOlder.Priority = 1
	sameEverythingOlder.CreatedAt = 5

	candidates := []*modeprofile.Profile{
		lowPriority,
		sameEverythingOlder,
		highPriority,
		samePriorityMoreSpecific,
	}

	sortCandidates(candidates)

	assert.Equal(t, []string{"HIGH", "SPECIFIC", "OLDER", "LOW"}, []string{
		candidates[0].Code,
		candidates[1].Code,
		candidates[2].Code,
		candidates[3].Code,
	})
}

func TestResolve_ReordersCachedProfilesBeforeSelecting(t *testing.T) {
	customerID := pulid.MustNew("cus_")

	orgDefault := coreProfile("OTR", true)
	orgDefault.Priority = 0

	scoped := coreProfile("OTR-ACME", false)
	scoped.CustomerID = &customerID
	scoped.SpecificityScore = modeprofile.SpecificityWeightCustomer

	cache := &fakeCacheRepo{profiles: []*modeprofile.Profile{orgDefault, scoped}}
	svc := newCachingService(&fakeProfileRepo{}, cache)

	policy, err := svc.Resolve(t.Context(), &services.ResolveModeProfileRequest{
		CustomerID: customerID,
	})

	require.NoError(t, err)
	require.NotNil(t, policy)
	assert.Equal(t, "OTR-ACME", policy.ProfileCode)
}

func TestInvalidateCache(t *testing.T) {
	cache := &fakeCacheRepo{}
	svc := newCachingService(&fakeProfileRepo{}, cache)

	require.NoError(t, svc.InvalidateCache(t.Context(), pagination.TenantInfo{}))
	assert.Equal(t, 1, cache.invalidations)

	noCache := &service{profileRepo: &fakeProfileRepo{}, l: zap.NewNop()}
	require.NoError(t, noCache.InvalidateCache(t.Context(), pagination.TenantInfo{}))
}
