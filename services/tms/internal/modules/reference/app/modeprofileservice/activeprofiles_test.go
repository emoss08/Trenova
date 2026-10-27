package modeprofileservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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
	_ *modeprofileports.ListModeProfilesRequest,
) (*pagination.ListResult[*modeprofile.Profile], error) {
	return nil, nil
}

func (f *fakeProfileRepo) GetByID(
	_ context.Context,
	_ *modeprofileports.GetModeProfileByIDRequest,
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

func newProfileService(repo *fakeProfileRepo) *service {
	return &service{profileRepo: repo, l: zap.NewNop()}
}

func TestResolve_ReadsTheActiveProfilesOnEveryResolve(t *testing.T) {
	repo := &fakeProfileRepo{profiles: []*modeprofile.Profile{coreProfile("OTR", true)}}
	svc := newProfileService(repo)

	for range 2 {
		policy, err := svc.Resolve(t.Context(), &modeprofileports.ResolveModeProfileRequest{})
		require.NoError(t, err)
		require.NotNil(t, policy)
		assert.Equal(t, "OTR", policy.ProfileCode)
	}

	assert.Equal(t, 2, repo.calls, "the repository's result cache serves repeats")
}

func TestResolve_DatabaseErrorPropagates(t *testing.T) {
	dbErr := errors.New("connection refused")
	svc := newProfileService(&fakeProfileRepo{err: dbErr})

	policy, err := svc.Resolve(t.Context(), &modeprofileports.ResolveModeProfileRequest{})

	require.ErrorIs(t, err, dbErr)
	assert.Nil(t, policy)
}

// A tenant with no profiles gets a sentinel rather than a bare nil policy, so a
// caller can tell "nothing is configured" from "the lookup failed" instead of
// treating a database outage as though no rules applied.
func TestResolve_EmptyProfileSetReportsNothingConfigured(t *testing.T) {
	svc := newProfileService(&fakeProfileRepo{profiles: nil})

	policy, err := svc.Resolve(t.Context(), &modeprofileports.ResolveModeProfileRequest{})

	require.ErrorIs(t, err, modeprofileports.ErrNoModeProfileConfigured)
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

func TestResolve_ReordersTheActiveProfilesBeforeSelecting(t *testing.T) {
	customerID := pulid.MustNew("cus_")

	orgDefault := coreProfile("OTR", true)
	orgDefault.Priority = 0

	scoped := coreProfile("OTR-ACME", false)
	scoped.CustomerID = &customerID
	scoped.SpecificityScore = modeprofile.SpecificityWeightCustomer

	svc := newProfileService(
		&fakeProfileRepo{profiles: []*modeprofile.Profile{orgDefault, scoped}},
	)

	policy, err := svc.Resolve(t.Context(), &modeprofileports.ResolveModeProfileRequest{
		CustomerID: customerID,
	})

	require.NoError(t, err)
	require.NotNil(t, policy)
	assert.Equal(t, "OTR-ACME", policy.ProfileCode)
}
