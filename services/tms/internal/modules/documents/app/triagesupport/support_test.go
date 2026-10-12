package triagesupport

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/testutil/mocks/triageportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type answer struct {
	Value int `json:"value"`
}

var tenantInfo = pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"}

func request() *Request {
	return &Request{
		Section:    triageports.CacheBriefing,
		TenantInfo: tenantInfo,
		Variant:    "v",
		TTL:        time.Minute,
	}
}

func counting(calls *int) func(context.Context) (*answer, error) {
	return func(context.Context) (*answer, error) {
		*calls++

		return &answer{Value: 42}, nil
	}
}

func TestGetOrComputeWithoutACacheComputes(t *testing.T) {
	t.Parallel()

	calls := 0
	got, err := GetOrCompute(t.Context(), nil, zap.NewNop(), request(), counting(&calls))

	require.NoError(t, err)
	assert.Equal(t, 42, got.Value)
	assert.Equal(t, 1, calls)
}

func TestGetOrComputeServesAHitUnderTheCurrentEpoch(t *testing.T) {
	t.Parallel()

	cache := triageportsmock.NewMockTriageCache(t)
	cache.EXPECT().QueueEpoch(mock.Anything, tenantInfo).Return(int64(3), nil).Once()
	cache.EXPECT().
		Get(mock.Anything, mock.MatchedBy(func(key *triageports.CacheKey) bool {
			return key.Epoch == 3 && key.Variant == "v" && key.Section == triageports.CacheBriefing
		}), mock.Anything).
		Run(func(_ context.Context, _ *triageports.CacheKey, dest any) {
			dest.(*answer).Value = 7
		}).
		Return(true, nil).Once()

	calls := 0
	got, err := GetOrCompute(t.Context(), cache, zap.NewNop(), request(), counting(&calls))

	require.NoError(t, err)
	assert.Equal(t, 7, got.Value)
	assert.Zero(t, calls)
}

func TestGetOrComputeStoresAMiss(t *testing.T) {
	t.Parallel()

	cache := triageportsmock.NewMockTriageCache(t)
	cache.EXPECT().QueueEpoch(mock.Anything, tenantInfo).Return(int64(0), nil).Once()
	cache.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Once()
	cache.EXPECT().
		Set(mock.Anything, mock.Anything, &answer{Value: 42}, time.Minute).
		Return(errors.New("redis down")).Once()

	calls := 0
	got, err := GetOrCompute(t.Context(), cache, zap.NewNop(), request(), counting(&calls))

	require.NoError(t, err)
	assert.Equal(t, 42, got.Value)
	assert.Equal(t, 1, calls)
}

func TestGetOrComputeFallsBackWhenTheEpochCannotBeRead(t *testing.T) {
	t.Parallel()

	cache := triageportsmock.NewMockTriageCache(t)
	cache.EXPECT().QueueEpoch(mock.Anything, tenantInfo).Return(int64(0), errors.New("down")).Once()

	calls := 0
	got, err := GetOrCompute(t.Context(), cache, zap.NewNop(), request(), counting(&calls))

	require.NoError(t, err)
	assert.Equal(t, 42, got.Value)
	assert.Equal(t, 1, calls)
}

func TestGetOrComputeReturnsTheComputeError(t *testing.T) {
	t.Parallel()

	cache := triageportsmock.NewMockTriageCache(t)
	cache.EXPECT().QueueEpoch(mock.Anything, tenantInfo).Return(int64(1), nil).Once()
	cache.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Once()

	_, err := GetOrCompute(t.Context(), cache, zap.NewNop(), request(),
		func(context.Context) (*answer, error) { return nil, errors.New("query failed") })

	require.EqualError(t, err, "query failed")
}

func TestInvalidateBumpsTheEpoch(t *testing.T) {
	t.Parallel()

	cache := triageportsmock.NewMockTriageCache(t)
	cache.EXPECT().BumpQueueEpoch(mock.Anything, tenantInfo).Return(errors.New("down")).Once()

	Invalidate(t.Context(), cache, zap.NewNop(), tenantInfo)
	Invalidate(t.Context(), nil, zap.NewNop(), tenantInfo)
}

func TestSampleOfFallsBackToTheSubjectsReference(t *testing.T) {
	t.Parallel()

	withReference := SampleOf(&triageports.MessageRow{ID: "imsg_1", Reference: "PRO-1"})
	assert.Equal(t, "PRO-1", withReference.Reference)

	fromSubject := SampleOf(&triageports.MessageRow{ID: "imsg_2", Subject: "POD for load 4471823"})
	assert.Equal(t, ReferenceIn("POD for load 4471823"), fromSubject.Reference)

	assert.Len(t, SamplesOf([]*triageports.MessageRow{{ID: "imsg_1"}, {ID: "imsg_2"}}), 2)
}

func TestClockCountsDaysBack(t *testing.T) {
	t.Parallel()

	clock := NewClock(nil, func() int64 { return 10 * DaySeconds })

	assert.Equal(t, int64(10*DaySeconds), clock.Now())
	assert.Equal(t, int64(7*DaySeconds), clock.DaysAgo(3))

	loc, err := clock.Location(t.Context(), tenantInfo)
	require.NoError(t, err)
	assert.Equal(t, time.UTC, loc)
}
