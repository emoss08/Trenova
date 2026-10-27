package analysisrunservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/ports/analysisports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRuns struct {
	run    *analysis.Run
	listed []*analysis.Run
	asked  *analysisports.ListRunsRequest
}

func (f *fakeRuns) Create(_ context.Context, run *analysis.Run) (*analysis.Run, error) {
	run.ID = pulid.MustNew(analysis.RunIDPrefix)
	f.run = run

	return run, nil
}

func (f *fakeRuns) Get(_ context.Context, _ pagination.TenantInfo, id pulid.ID) (*analysis.Run, error) {
	if f.run == nil || f.run.ID != id {
		return nil, errortypes.NewNotFoundError("Analysis not found")
	}

	return f.run, nil
}

func (f *fakeRuns) List(
	_ context.Context,
	req *analysisports.ListRunsRequest,
) ([]*analysis.Run, error) {
	f.asked = req

	return f.listed[:min(len(f.listed), req.Limit)], nil
}

func newService(repo *fakeRuns) *Service {
	return New(Params{Repo: repo, Logger: zap.NewNop()})
}

func TestAnAnalysisIsReadOnlyByThePersonItRanForOrSomeoneWhoOverseesAgents(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	run := &analysis.Run{ID: pulid.MustNew(analysis.RunIDPrefix), RequestedByID: owner}
	svc := newService(&fakeRuns{run: run})
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	ownerTenant := tenant
	ownerTenant.UserID = owner
	read, err := svc.Run(t.Context(), &analysisports.RunRequest{TenantInfo: ownerTenant, RunID: run.ID})
	require.NoError(t, err)
	assert.Equal(t, run.ID, read.ID)

	other := tenant
	other.UserID = pulid.MustNew("usr_")
	_, err = svc.Run(t.Context(), &analysisports.RunRequest{TenantInfo: other, RunID: run.ID})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "another person is told it does not exist")

	read, err = svc.Run(t.Context(), &analysisports.RunRequest{
		TenantInfo: other,
		RunID:      run.ID,
		Oversees:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, run.ID, read.ID)

	_, err = svc.Run(t.Context(), &analysisports.RunRequest{
		TenantInfo: ownerTenant,
		RunID:      pulid.MustNew(analysis.RunIDPrefix),
		Oversees:   true,
	})
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestTheHistoryPagesFromTheLastRunsTimeAndID(t *testing.T) {
	t.Parallel()

	runs := make([]*analysis.Run, 0, 3)
	for idx := range 3 {
		runs = append(runs, &analysis.Run{
			ID:        pulid.MustNew(analysis.RunIDPrefix),
			StartedAt: int64(1_800_000_000 - idx),
		})
	}
	repo := &fakeRuns{listed: runs}
	svc := newService(repo)

	page, err := svc.ListRuns(t.Context(), &analysisports.ListRunsRequest{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page.Items, 2)
	assert.Equal(t, runs[1].StartedAt, page.NextBefore)
	assert.Equal(t, runs[1].ID, page.NextBeforeID)

	last, err := svc.ListRuns(t.Context(), &analysisports.ListRunsRequest{Limit: 5})
	require.NoError(t, err)
	assert.Len(t, last.Items, 3)
	assert.Zero(t, last.NextBefore, "a page shorter than asked for is the last")
	assert.True(t, last.NextBeforeID.IsNil())

	_, err = svc.ListRuns(t.Context(), &analysisports.ListRunsRequest{Limit: 1000})
	require.NoError(t, err)
	assert.Equal(t, runListLimit, repo.asked.Limit, "a page is never larger than 200")
}

func TestRecordingAnAnalysisBoundsItFirst(t *testing.T) {
	t.Parallel()

	repo := &fakeRuns{}
	long := make([]rune, analysis.MaxOutputRunes+500)
	for idx := range long {
		long[idx] = 'a'
	}

	recorded, err := newService(repo).Record(t.Context(), &analysis.Run{
		Status: analysis.StatusSucceeded,
		Output: string(long),
	})
	require.NoError(t, err)
	assert.LessOrEqual(t, len([]rune(recorded.Output)), analysis.MaxOutputRunes)
}
