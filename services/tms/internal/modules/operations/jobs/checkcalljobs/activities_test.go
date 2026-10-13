package checkcalljobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type fakeLinks struct {
	checkcallports.TrackingLinkRepository

	deadAt  []int64
	befores []int64
	limits  []int
	err     error
}

func (f *fakeLinks) PurgeDead(_ context.Context, before int64, limit int) (int, error) {
	f.befores = append(f.befores, before)
	f.limits = append(f.limits, limit)
	if f.err != nil {
		return 0, f.err
	}
	kept := make([]int64, 0, len(f.deadAt))
	purged := 0
	for _, at := range f.deadAt {
		if purged < limit && at < before {
			purged++
			continue
		}
		kept = append(kept, at)
	}
	f.deadAt = kept
	return purged, nil
}

func TestPurgeDeadTrackingLinksSparesRecentlyDeadLinks(t *testing.T) {
	t.Parallel()

	now := timeutils.NowUnix()
	links := &fakeLinks{deadAt: []int64{
		now - 90*secondsPerDay,
		now - 31*secondsPerDay,
		now - secondsPerDay,
		now + 7*secondsPerDay,
	}}
	a := &Activities{links: links, l: zap.NewNop()}

	result, err := a.PurgeDeadTrackingLinksActivity(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(2), result.LinksPurged)
	assert.Len(t, links.deadAt, 2)
	require.Len(t, links.limits, 1)
	assert.Equal(t, linkPurgeBatchSize, links.limits[0])
	assert.LessOrEqual(t, links.befores[0], now-linkRetentionDays*secondsPerDay+5)
}

func TestPurgeDeadTrackingLinksLoopsUntilAShortBatch(t *testing.T) {
	t.Parallel()

	now := timeutils.NowUnix()
	dead := make([]int64, 0, linkPurgeBatchSize+3)
	for i := range linkPurgeBatchSize + 3 {
		dead = append(dead, now-90*secondsPerDay-int64(i))
	}
	links := &fakeLinks{deadAt: dead}
	a := &Activities{links: links, l: zap.NewNop()}

	result, err := a.PurgeDeadTrackingLinksActivity(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(linkPurgeBatchSize+3), result.LinksPurged)
	assert.Len(t, links.limits, 2)
	assert.Empty(t, links.deadAt)
}

func TestPurgeDeadTrackingLinksPropagatesFailure(t *testing.T) {
	t.Parallel()

	a := &Activities{links: &fakeLinks{err: errors.New("connection reset")}, l: zap.NewNop()}

	result, err := a.PurgeDeadTrackingLinksActivity(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestPurgeTrackingLinksWorkflowRunsThePurge(t *testing.T) {
	t.Parallel()

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(PurgeTrackingLinksWorkflow)

	var a *Activities
	env.OnActivity(a.PurgeDeadTrackingLinksActivity, mock.Anything).
		Return(&PurgeTrackingLinksResult{LinksPurged: 9}, nil).
		Once()

	env.ExecuteWorkflow(PurgeTrackingLinksWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result PurgeTrackingLinksResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, int64(9), result.LinksPurged)
	env.AssertExpectations(t)
}
