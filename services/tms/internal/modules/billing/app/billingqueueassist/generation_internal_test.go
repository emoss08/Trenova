package billingqueueassist

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type servesEverything struct{ llmports.TaskRunner }

func (servesEverything) Available(context.Context, aitask.Task, pagination.TenantInfo) (bool, error) {
	return true, nil
}

type reviewingControls struct {
	tenantports.BillingControlRepository
}

func (reviewingControls) GetByOrgID(context.Context, pulid.ID) (*billingcontrol.BillingControl, error) {
	return &billingcontrol.BillingControl{QueueAgentReviewEnabled: true}, nil
}

type recordingGenerator struct {
	state     billingqueueports.NovaGenerationState
	requested int
	trigger   billingqueueports.NovaTrigger
}

func (g *recordingGenerator) RequestGeneration(
	_ context.Context,
	_ pagination.TenantInfo,
	trigger billingqueueports.NovaTrigger,
) {
	g.requested++
	g.trigger = trigger
}

// emptyBatcher is a provider with a batch API; the tests using it never get
// as far as sending a batch.
type emptyBatcher struct{ llmports.TaskBatcher }

func (g *recordingGenerator) Generation(
	context.Context,
	pagination.TenantInfo,
) (billingqueueports.NovaGenerationState, error) {
	return g.state, nil
}

func generationService(
	t *testing.T,
	generator *recordingGenerator,
) (*Service, *billingqueueportsmock.MockBillingQueueRepository) {
	t.Helper()
	repo := billingqueueportsmock.NewMockBillingQueueRepository(t)

	return &Service{
		l:         zap.NewNop(),
		repo:      repo,
		controls:  reviewingControls{},
		nova:      servesEverything{},
		generator: generator,
	}, repo
}

func TestNovaGeneration_ShowsTheGenerationAlreadyRunningWithoutStartingAnother(t *testing.T) {
	t.Parallel()

	running := &billingqueueports.NovaGeneration{Phase: billingqueueports.NovaPhaseReviewing, Total: 40, Done: 12}
	generator := &recordingGenerator{state: billingqueueports.NovaGenerationState{Running: running}}
	s, _ := generationService(t, generator)

	got, err := s.NovaGeneration(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Equal(t, running, got)
	assert.Zero(t, generator.requested)
}

func TestNovaGeneration_StartsOneForAQueueWithItemsNovaHasNotReviewed(t *testing.T) {
	t.Parallel()

	generator := &recordingGenerator{}
	s, repo := generationService(t, generator)
	repo.EXPECT().ListUnreviewedItems(mock.Anything, mock.Anything).
		Return([]*billingqueueports.UnreviewedItem{{ID: pulid.MustNew("bqi_")}}, nil)

	got, err := s.NovaGeneration(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Equal(t, &billingqueueports.NovaGeneration{Phase: billingqueueports.NovaPhaseReviewing}, got,
		"the reader sees it start at once, before the workflow first answers")
	assert.Equal(t, 1, generator.requested)
	assert.Equal(t, billingqueueports.NovaTriggerQueueOpened, generator.trigger,
		"someone opened the queue, so it reviews at once rather than waiting on a batch")
}

func TestNovaGeneration_SaysWhatTheLastGenerationLeftAndWhenItTriesAgain(t *testing.T) {
	t.Parallel()

	endedAt := timeutils.NowUnix() - 30
	generator := &recordingGenerator{state: billingqueueports.NovaGenerationState{
		EndedAt:        endedAt,
		LeftUnreviewed: true,
	}}
	s, repo := generationService(t, generator)
	repo.EXPECT().ListUnreviewedItems(mock.Anything, mock.Anything).
		Return([]*billingqueueports.UnreviewedItem{{ID: pulid.MustNew("bqi_")}}, nil)

	got, err := s.NovaGeneration(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Equal(t, &billingqueueports.NovaGeneration{
		Phase:   billingqueueports.NovaPhaseLeft,
		RetryAt: endedAt + generationRest,
	}, got)
	assert.Zero(t, generator.requested, "items the last generation just failed on wait for the retry")
}

func TestNovaGeneration_ShowsNothingLeftOnceWhatWasLeftHasBeenReviewed(t *testing.T) {
	t.Parallel()

	generator := &recordingGenerator{state: billingqueueports.NovaGenerationState{
		EndedAt:        timeutils.NowUnix() - 30,
		LeftUnreviewed: true,
	}}
	s, repo := generationService(t, generator)
	repo.EXPECT().ListUnreviewedItems(mock.Anything, mock.Anything).Return(nil, nil)
	repo.EXPECT().GetBrief(mock.Anything, mock.Anything).
		Return(&billingqueue.QueueBrief{Text: "All clear."}, nil)

	got, err := s.NovaGeneration(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Nil(t, got)
	assert.Zero(t, generator.requested)
}

func TestNovaGeneration_StartsOneRightAfterAGenerationThatFinishedEverything(t *testing.T) {
	t.Parallel()

	generator := &recordingGenerator{state: billingqueueports.NovaGenerationState{
		EndedAt: timeutils.NowUnix() - 30,
	}}
	s, repo := generationService(t, generator)
	repo.EXPECT().ListUnreviewedItems(mock.Anything, mock.Anything).
		Return([]*billingqueueports.UnreviewedItem{{ID: pulid.MustNew("bqi_")}}, nil)

	_, err := s.NovaGeneration(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Equal(t, 1, generator.requested, "what is unreviewed now is new, so nothing is waited out")
}

func TestNovaGeneration_StartsAnotherOnceTheLastHasRested(t *testing.T) {
	t.Parallel()

	generator := &recordingGenerator{state: billingqueueports.NovaGenerationState{
		EndedAt:        timeutils.NowUnix() - generationRest - 1,
		LeftUnreviewed: true,
	}}
	s, repo := generationService(t, generator)
	repo.EXPECT().ListUnreviewedItems(mock.Anything, mock.Anything).
		Return([]*billingqueueports.UnreviewedItem{{ID: pulid.MustNew("bqi_")}}, nil)

	_, err := s.NovaGeneration(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Equal(t, 1, generator.requested)
}

func TestNovaGeneration_StartsOneForAQueueWithNoBrief(t *testing.T) {
	t.Parallel()

	generator := &recordingGenerator{}
	s, repo := generationService(t, generator)
	repo.EXPECT().ListUnreviewedItems(mock.Anything, mock.Anything).Return(nil, nil)
	repo.EXPECT().GetBrief(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("no brief"))

	got, err := s.NovaGeneration(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)

	assert.NotNil(t, got)
	assert.Equal(t, 1, generator.requested)
}

func TestNovaGeneration_LeavesAQueueWithEveryReviewAndABriefAlone(t *testing.T) {
	t.Parallel()

	generator := &recordingGenerator{}
	s, repo := generationService(t, generator)
	repo.EXPECT().ListUnreviewedItems(mock.Anything, mock.Anything).Return(nil, nil)
	repo.EXPECT().GetBrief(mock.Anything, mock.Anything).
		Return(&billingqueue.QueueBrief{Text: "All clear."}, nil)

	got, err := s.NovaGeneration(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Nil(t, got)
	assert.Zero(t, generator.requested)
}

func TestSubmitReviews_EndsOnAShortPage(t *testing.T) {
	t.Parallel()

	s, repo := generationService(t, &recordingGenerator{})
	repo.EXPECT().ListUnreviewedItems(mock.Anything, mock.MatchedBy(
		func(req *billingqueueports.ListUnreviewedItemsRequest) bool { return req.Limit == generationPage },
	)).Return(nil, nil)

	submission, err := s.SubmitReviews(t.Context(), &billingqueueports.SubmitReviewsRequest{})
	require.NoError(t, err)

	assert.Zero(t, submission.Count())
	assert.Nil(t, submission.Next, "a short page is the last")
}

// unreadableQueue fails to read one item and finds the rest gone.
type unreadableQueue struct {
	billingqueueports.BillingQueueService
	broken pulid.ID
	read   []pulid.ID
}

func (q *unreadableQueue) GetByID(
	_ context.Context,
	req *billingqueueports.GetBillingQueueItemByIDRequest,
) (*billingqueue.BillingQueueItem, error) {
	q.read = append(q.read, req.ItemID)
	if req.ItemID == q.broken {
		return nil, errors.New("scan failed")
	}

	return nil, errortypes.NewNotFoundError("gone")
}

func TestSubmitReviews_SkipsAnItemWhoseEvidenceCannotBeReadAndReadsTheRest(t *testing.T) {
	t.Parallel()

	s, repo := generationService(t, &recordingGenerator{})
	broken, after := pulid.MustNew("bqi_"), pulid.MustNew("bqi_")
	queue := &unreadableQueue{broken: broken}
	s.queue = queue
	s.batches = emptyBatcher{}
	repo.EXPECT().ListUnreviewedItems(mock.Anything, mock.Anything).Return(
		[]*billingqueueports.UnreviewedItem{{ID: broken}, {ID: after}}, nil,
	)

	submission, err := s.SubmitReviews(t.Context(), &billingqueueports.SubmitReviewsRequest{})

	require.NoError(t, err, "one item that cannot be read does not stop the page")
	assert.Equal(t, []pulid.ID{broken, after}, queue.read)
	assert.Zero(t, submission.Count())
}

func TestSubmitReviews_LeavesAPageSomeoneIsWaitingOnToBeReviewedOneItemAtATime(t *testing.T) {
	t.Parallel()

	s, repo := generationService(t, &recordingGenerator{})
	queue := &unreadableQueue{}
	s.queue = queue
	s.batches = emptyBatcher{}
	first, second := pulid.MustNew("bqi_"), pulid.MustNew("bqi_")
	repo.EXPECT().ListUnreviewedItems(mock.Anything, mock.Anything).Return(
		[]*billingqueueports.UnreviewedItem{{ID: first}, {ID: second}}, nil,
	)

	submission, err := s.SubmitReviews(t.Context(), &billingqueueports.SubmitReviewsRequest{Inline: true})
	require.NoError(t, err)

	assert.Equal(t, []pulid.ID{first, second}, submission.Inline)
	assert.Nil(t, submission.Handle)
	assert.Empty(t, queue.read, "each review gathers its own evidence when it runs")
}

type cancelingBatcher struct {
	llmports.TaskBatcher
	canceled []*llmports.BatchCancelRequest
}

func (b *cancelingBatcher) CancelBatch(_ context.Context, req *llmports.BatchCancelRequest) error {
	b.canceled = append(b.canceled, req)

	return nil
}

func TestCancelReviews_CancelsTheBatchTheSubmissionWentOutAs(t *testing.T) {
	t.Parallel()

	s, _ := generationService(t, &recordingGenerator{})
	batches := &cancelingBatcher{}
	s.batches = batches
	providerID := pulid.MustNew("aiprv_")
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	require.NoError(t, s.CancelReviews(t.Context(), tenant, &billingqueueports.ReviewSubmission{
		Handle: &llmports.BatchHandle{ProviderID: providerID, BatchID: "msgbatch_1"},
	}))
	require.NoError(t, s.CancelReviews(t.Context(), tenant, &billingqueueports.ReviewSubmission{
		Inline: []pulid.ID{pulid.MustNew("bqi_")},
	}), "a page reviewed one at a time has no batch to cancel")

	assert.Equal(t, []*llmports.BatchCancelRequest{
		{TenantInfo: tenant, ProviderID: providerID, BatchID: "msgbatch_1"},
	}, batches.canceled)
}

func TestApplyReviews_RecordsNothingForASubmissionThatWasNotBatched(t *testing.T) {
	t.Parallel()

	s, _ := generationService(t, &recordingGenerator{})

	applied, err := s.ApplyReviews(t.Context(), &billingqueueports.ApplyReviewsRequest{
		Submission: &billingqueueports.ReviewSubmission{Inline: []pulid.ID{pulid.MustNew("bqi_")}},
	})
	require.NoError(t, err)

	assert.Zero(t, applied)
}
