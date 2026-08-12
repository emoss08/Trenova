package tenderjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

// tokenRow mirrors the three timestamps the purge predicate coalesces over, so
// the fake stores answer exactly what the SQL does:
// COALESCE(used_at, revoked_at, expires_at) < deadBefore.
type tokenRow struct {
	name      string
	usedAt    *int64
	revokedAt *int64
	expiresAt int64
}

func (r tokenRow) deadAt() int64 {
	switch {
	case r.usedAt != nil:
		return *r.usedAt
	case r.revokedAt != nil:
		return *r.revokedAt
	default:
		return r.expiresAt
	}
}

type tokenStore struct {
	rows     []tokenRow
	requests []repositories.PurgeDeadTokensRequest
	err      error
}

func (s *tokenStore) purge(
	_ context.Context,
	req repositories.PurgeDeadTokensRequest,
) (int64, error) {
	s.requests = append(s.requests, req)
	if s.err != nil {
		return 0, s.err
	}

	kept := make([]tokenRow, 0, len(s.rows))
	var purged int64
	for _, row := range s.rows {
		if purged < int64(req.Limit) && row.deadAt() < req.DeadBefore {
			purged++
			continue
		}
		kept = append(kept, row)
	}
	s.rows = kept
	return purged, nil
}

func (s *tokenStore) names() []string {
	names := make([]string, 0, len(s.rows))
	for _, row := range s.rows {
		names = append(names, row.name)
	}
	return names
}

type fakeTokenTenderRepo struct {
	repositories.TenderRepository

	store *tokenStore
}

func (f *fakeTokenTenderRepo) PurgeDeadOfferTokens(
	ctx context.Context,
	req repositories.PurgeDeadTokensRequest,
) (int64, error) {
	return f.store.purge(ctx, req)
}

type fakeTokenRateConRepo struct {
	repositories.RateConfirmationRepository

	store *tokenStore
}

func (f *fakeTokenRateConRepo) PurgeDeadSignTokens(
	ctx context.Context,
	req repositories.PurgeDeadTokensRequest,
) (int64, error) {
	return f.store.purge(ctx, req)
}

func ptrUnix(v int64) *int64 { return &v }

func retentionFixture() []tokenRow {
	now := timeutils.NowUnix()
	longAgo := now - 90*secondsPerDay
	return []tokenRow{
		{name: "used-long-ago", usedAt: ptrUnix(longAgo), expiresAt: now + 3600},
		{name: "revoked-long-ago", revokedAt: ptrUnix(longAgo), expiresAt: now + 3600},
		{name: "expired-long-ago", expiresAt: longAgo},
		{name: "live", expiresAt: now + 3600},
		{name: "used-yesterday", usedAt: ptrUnix(now - secondsPerDay), expiresAt: now + 3600},
		{name: "expired-yesterday", expiresAt: now - secondsPerDay},
	}
}

func newTokenRetentionActivities(
	offerStore, signStore *tokenStore,
) *Activities {
	return &Activities{
		repo:        &fakeTokenTenderRepo{store: offerStore},
		rateConRepo: &fakeTokenRateConRepo{store: signStore},
		l:           zap.NewNop(),
	}
}

func TestPurgeDeadTokensActivity_PurgesEveryDeadClassAndSparesLiveTokens(t *testing.T) {
	offerStore := &tokenStore{rows: retentionFixture()}
	signStore := &tokenStore{rows: retentionFixture()}
	a := newTokenRetentionActivities(offerStore, signStore)

	result, err := a.PurgeDeadTokensActivity(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(3), result.OfferTokensPurged)
	assert.Equal(t, int64(3), result.SignTokensPurged)

	survivors := []string{"live", "used-yesterday", "expired-yesterday"}
	assert.ElementsMatch(t, survivors, offerStore.names())
	assert.ElementsMatch(t, survivors, signStore.names())
}

func TestPurgeDeadTokensActivity_UsesTheThirtyDayWindowAndBatchSize(t *testing.T) {
	offerStore := &tokenStore{}
	signStore := &tokenStore{}
	a := newTokenRetentionActivities(offerStore, signStore)

	before := timeutils.NowUnix()
	_, err := a.PurgeDeadTokensActivity(t.Context())
	require.NoError(t, err)

	require.Len(t, offerStore.requests, 1)
	require.Len(t, signStore.requests, 1)
	assert.Equal(t, tokenPurgeBatchSize, offerStore.requests[0].Limit)
	assert.Equal(t, tokenPurgeBatchSize, signStore.requests[0].Limit)
	assert.GreaterOrEqual(
		t, offerStore.requests[0].DeadBefore, before-tokenRetentionDays*secondsPerDay,
	)
	assert.Less(t, offerStore.requests[0].DeadBefore, before)
}

func TestPurgeDeadTokensActivity_LoopsUntilAShortBatch(t *testing.T) {
	now := timeutils.NowUnix()
	rows := make([]tokenRow, 0, tokenPurgeBatchSize+7)
	for i := range tokenPurgeBatchSize + 7 {
		rows = append(rows, tokenRow{name: "dead", expiresAt: now - int64(i) - 90*secondsPerDay})
	}
	offerStore := &tokenStore{rows: rows}
	signStore := &tokenStore{}
	a := newTokenRetentionActivities(offerStore, signStore)

	result, err := a.PurgeDeadTokensActivity(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(tokenPurgeBatchSize+7), result.OfferTokensPurged)
	assert.Len(t, offerStore.requests, 2)
	assert.Empty(t, offerStore.rows)
}

func TestPurgeDeadTokensActivity_PropagatesRepositoryFailure(t *testing.T) {
	offerStore := &tokenStore{err: errors.New("connection reset")}
	a := newTokenRetentionActivities(offerStore, &tokenStore{})

	result, err := a.PurgeDeadTokensActivity(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestPurgeTenderTokensWorkflow_RunsThePurgeActivity(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(PurgeTenderTokensWorkflow)

	var a *Activities
	env.OnActivity(a.PurgeDeadTokensActivity, mock.Anything).
		Return(&PurgeTenderTokensResult{OfferTokensPurged: 12, SignTokensPurged: 4}, nil).
		Once()

	env.ExecuteWorkflow(PurgeTenderTokensWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	result := new(PurgeTenderTokensResult)
	require.NoError(t, env.GetWorkflowResult(result))
	assert.Equal(t, int64(12), result.OfferTokensPurged)
	assert.Equal(t, int64(4), result.SignTokensPurged)
	env.AssertExpectations(t)
}
