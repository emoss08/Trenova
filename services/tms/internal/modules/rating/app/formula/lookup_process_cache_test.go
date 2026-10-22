package formula_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stampedMatrixRepo struct {
	ratematrixports.RateMatrixRepository
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
	*ratematrixports.GetRateMatrixLookupDataRequest,
) ([]*ratematrixports.RateMatrixLookupData, error) {
	r.reads.Add(1)
	return []*ratematrixports.RateMatrixLookupData{
		exactLookupMatrix("fsc", map[string]string{"DIESEL": "0.35"}),
	}, nil
}

func (r *stampedMatrixRepo) GetLookupStamp(
	context.Context,
	pagination.TenantInfo,
) (string, error) {
	return r.stamp.Load().(string), nil //nolint:errcheck // always a string
}

func TestRateTables_EncodedOncePerStampAndSharedAcrossRequests(t *testing.T) {
	t.Parallel()

	repo := newStampedMatrixRepo("2:5:1700000000")
	svc := setupCountingService(t, repo)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	first, err := svc.RateTables(t.Context(), tenant)
	require.NoError(t, err)
	second, err := svc.RateTables(t.Context(), tenant)
	require.NoError(t, err)
	assert.EqualValues(t, 1, repo.reads.Load(), "two requests, one full read")
	assert.Same(t, first, second, "the second request reuses the encoded bytes")
	require.Len(t, first.Tables, 1)
	assert.Equal(t, "2:5:1700000000", first.Stamp)
	assert.Contains(t, string(first.Tables[0].Encoded), `"value":"0.35"`)

	repo.stamp.Store("2:6:1700000900")
	third, err := svc.RateTables(t.Context(), tenant)
	require.NoError(t, err)
	assert.EqualValues(t, 2, repo.reads.Load(), "an edited matrix is read again")
	assert.NotSame(t, first, third)
	assert.Equal(t, "2:6:1700000900", third.Stamp)

	ratetablecache.Invalidate(tenant.OrgID, tenant.BuID)
	_, err = svc.RateTables(t.Context(), tenant)
	require.NoError(t, err)
	assert.EqualValues(t, 3, repo.reads.Load(), "the write path can evict directly")
}
