package formula_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stampedMatrixRepo answers the cheap stamp question and counts the expensive
// read, so a test can tell reuse from rebuild.
type stampedMatrixRepo struct {
	repositories.RateMatrixRepository
	stamp atomic.Value
	reads atomic.Int32
}

func newStampedMatrixRepo(stamp string) *stampedMatrixRepo {
	repo := &stampedMatrixRepo{}
	repo.stamp.Store(stamp)
	return repo
}

func (r *stampedMatrixRepo) GetLookupData(
	context.Context,
	*repositories.GetRateMatrixLookupDataRequest,
) ([]*repositories.RateMatrixLookupData, error) {
	r.reads.Add(1)
	return nil, nil
}

func (r *stampedMatrixRepo) GetLookupStamp(
	context.Context,
	pagination.TenantInfo,
) (string, error) {
	return r.stamp.Load().(string), nil //nolint:errcheck // always a string
}

func TestBuildLookup_SharesAcrossRequestsUntilAMatrixChanges(t *testing.T) {
	t.Parallel()

	repo := newStampedMatrixRepo("2:5:1700000000")
	svc := setupCountingService(t, repo)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	_, err := svc.BuildLookup(t.Context(), tenant)
	require.NoError(t, err)
	_, err = svc.BuildLookup(t.Context(), tenant)
	require.NoError(t, err)
	assert.EqualValues(t, 1, repo.reads.Load(), "two requests, one full read")

	repo.stamp.Store("2:6:1700000900")
	_, err = svc.BuildLookup(t.Context(), tenant)
	require.NoError(t, err)
	assert.EqualValues(t, 2, repo.reads.Load(), "an edited matrix is read again")

	ratetablecache.Invalidate(tenant.OrgID, tenant.BuID)
	_, err = svc.BuildLookup(t.Context(), tenant)
	require.NoError(t, err)
	assert.EqualValues(t, 3, repo.reads.Load(), "the write path can evict directly")
}
