package billingqueuejobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/billingqueuejobs/billingqueuecontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func novaEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	return novaEnvLeaving(t, false)
}

// novaEnvLeaving is a generation's test environment where finishing leaves
// items unreviewed, or not.
func novaEnvLeaving(t *testing.T, left bool) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(BillingQueueNovaGenerationWorkflow)

	var a *Activities
	env.RegisterActivity(a.SubmitNovaReviewsActivity)
	env.RegisterActivity(a.NovaBatchEndedActivity)
	env.RegisterActivity(a.CancelNovaReviewsActivity)
	env.RegisterActivity(a.ApplyNovaReviewsActivity)
	env.RegisterActivity(a.PublishNovaProgressActivity)
	env.RegisterActivity(a.NovaLeftUnreviewedActivity)
	env.RegisterActivity(a.AgentReviewActivity)
	env.RegisterActivity(a.AutoApproveActivity)
	env.RegisterActivity(a.RefreshBriefActivity)
	env.OnActivity(a.PublishNovaProgressActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.NovaLeftUnreviewedActivity, mock.Anything, mock.Anything).
		Return(left, nil).
		Maybe()
	env.OnActivity(a.RefreshBriefActivity, mock.Anything, mock.Anything).Return(nil).Once()

	return env
}

func novaPayload() *billingqueuecontract.NovaGenerationPayload {
	return &billingqueuecontract.NovaGenerationPayload{
		OrganizationID: pulid.ID("org_test"),
		BusinessUnitID: pulid.ID("bu_test"),
	}
}

func progressOf(
	t *testing.T,
	env *testsuite.TestWorkflowEnvironment,
) billingqueueports.NovaGeneration {
	t.Helper()
	encoded, err := env.QueryWorkflow(billingqueuecontract.NovaGenerationProgressQuery)
	require.NoError(t, err)
	var progress billingqueuecontract.NovaGenerationProgress
	require.NoError(t, encoded.Get(&progress))

	return progress.Generation
}

func TestNovaGeneration_WaitsForTheBatchRecordsItThenWritesTheBrief(t *testing.T) {
	t.Parallel()

	env := novaEnv(t)
	var a *Activities
	submission := &billingqueueports.ReviewSubmission{
		Handle: &billingqueueports.ReviewBatch{BatchID: "batch_1"},
		Tickets: []billingqueueports.ReviewTicket{
			{ItemID: pulid.ID("bqi_1"), Hash: "h1"},
			{ItemID: pulid.ID("bqi_2"), Hash: "h2"},
		},
	}
	env.OnActivity(a.SubmitNovaReviewsActivity, mock.Anything, mock.MatchedBy(func(in *SubmitNovaReviewsInput) bool {
		return !in.Inline
	})).
		Return(submission, nil).
		Once()
	env.OnActivity(a.NovaBatchEndedActivity, mock.Anything, mock.Anything).Return(false, nil).Once()
	env.OnActivity(a.NovaBatchEndedActivity, mock.Anything, mock.Anything).Return(true, nil).Once()
	env.OnActivity(a.ApplyNovaReviewsActivity, mock.Anything, mock.MatchedBy(func(in *NovaBatchInput) bool {
		return in.Submission.Handle.BatchID == "batch_1" && len(in.Submission.Tickets) == 2
	})).
		Return(2, nil).
		Once()

	env.ExecuteWorkflow(BillingQueueNovaGenerationWorkflow, novaPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
	assert.Equal(t, billingqueueports.NovaGeneration{
		Phase: billingqueueports.NovaPhaseDone,
		Total: 2,
		Done:  2,
	}, progressOf(t, env))
}

func batchOfTwo() *billingqueueports.ReviewSubmission {
	return &billingqueueports.ReviewSubmission{
		Handle: &billingqueueports.ReviewBatch{BatchID: "batch_1"},
		Tickets: []billingqueueports.ReviewTicket{
			{ItemID: pulid.ID("bqi_1"), Hash: "h1"},
			{ItemID: pulid.ID("bqi_2"), Hash: "h2"},
		},
	}
}

func signalNova(env *testsuite.TestWorkflowEnvironment, trigger billingqueueports.NovaTrigger) {
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(
			billingqueuecontract.NovaGenerationSignalName,
			billingqueuecontract.NovaGenerationSignal{Trigger: trigger},
		)
	}, 10*time.Second)
}

func TestNovaGeneration_SomeoneOpeningTheQueueCancelsTheBatchAndReviewsTheRestAtOnce(t *testing.T) {
	t.Parallel()

	env := novaEnv(t)
	var a *Activities
	env.OnActivity(a.SubmitNovaReviewsActivity, mock.Anything, mock.MatchedBy(func(in *SubmitNovaReviewsInput) bool {
		return !in.Inline
	})).
		Return(batchOfTwo(), nil).
		Once()
	env.OnActivity(a.CancelNovaReviewsActivity, mock.Anything, mock.MatchedBy(func(in *NovaBatchInput) bool {
		return in.Submission.Handle.BatchID == "batch_1"
	})).
		Return(nil).
		Once()
	env.OnActivity(a.NovaBatchEndedActivity, mock.Anything, mock.Anything).Return(true, nil).Once()
	env.OnActivity(a.ApplyNovaReviewsActivity, mock.Anything, mock.Anything).Return(1, nil).Once()
	env.OnActivity(a.SubmitNovaReviewsActivity, mock.Anything, mock.MatchedBy(func(in *SubmitNovaReviewsInput) bool {
		return in.Inline
	})).
		Return(&billingqueueports.ReviewSubmission{Inline: []pulid.ID{"bqi_2"}}, nil).
		Once()
	env.OnActivity(a.AgentReviewActivity, mock.Anything, mock.Anything).
		Return(&billingqueueports.AgentReviewOutcome{Reviewed: true}, nil).Once()
	signalNova(env, billingqueueports.NovaTriggerQueueOpened)
	started := env.Now()

	env.ExecuteWorkflow(BillingQueueNovaGenerationWorkflow, novaPayload())

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
	assert.Less(
		t,
		env.Now().Sub(started),
		novaBatchFirstCheck,
		"it stopped waiting the moment the queue was opened",
	)
	assert.Equal(t, billingqueueports.NovaGeneration{
		Phase: billingqueueports.NovaPhaseDone,
		Total: 1,
		Done:  1,
	}, progressOf(t, env), "the brief is written once, after the rest are reviewed")
}

func TestNovaGeneration_TheScheduleAskingAgainLeavesTheBatchToEnd(t *testing.T) {
	t.Parallel()

	env := novaEnv(t)
	var a *Activities
	env.OnActivity(a.SubmitNovaReviewsActivity, mock.Anything, mock.Anything).
		Return(batchOfTwo(), nil).
		Once()
	env.OnActivity(a.NovaBatchEndedActivity, mock.Anything, mock.Anything).Return(true, nil).Once()
	env.OnActivity(a.ApplyNovaReviewsActivity, mock.Anything, mock.Anything).Return(2, nil).Once()
	env.OnActivity(a.SubmitNovaReviewsActivity, mock.Anything, mock.MatchedBy(func(in *SubmitNovaReviewsInput) bool {
		return !in.Inline
	})).
		Return(&billingqueueports.ReviewSubmission{}, nil).
		Once()
	env.OnActivity(a.RefreshBriefActivity, mock.Anything, mock.Anything).Return(nil)
	signalNova(env, billingqueueports.NovaTriggerSchedule)

	env.ExecuteWorkflow(BillingQueueNovaGenerationWorkflow, novaPayload())

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
	env.AssertNotCalled(t, "CancelNovaReviewsActivity", mock.Anything, mock.Anything)
	env.AssertNumberOfCalls(t, "SubmitNovaReviewsActivity", 2)
}

func TestNovaGeneration_ReportsWhenItEndsWithItemsItCouldNotReview(t *testing.T) {
	t.Parallel()

	env := novaEnvLeaving(t, true)
	var a *Activities
	env.OnActivity(a.SubmitNovaReviewsActivity, mock.Anything, mock.Anything).Return(
		&billingqueueports.ReviewSubmission{Inline: []pulid.ID{"bqi_1"}}, nil,
	).Once()
	env.OnActivity(a.AgentReviewActivity, mock.Anything, mock.Anything).
		Return((*billingqueueports.AgentReviewOutcome)(nil),
			temporal.NewNonRetryableApplicationError("provider unavailable", "Unavailable", nil),
		).
		Once()

	env.ExecuteWorkflow(BillingQueueNovaGenerationWorkflow, novaPayload())

	require.NoError(t, env.GetWorkflowError())
	var result billingqueuecontract.NovaGenerationProgress
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.True(
		t,
		result.LeftUnreviewed,
		"opening the queue does not send what failed round again at once",
	)
	assert.Equal(t, billingqueueports.NovaPhaseDone, result.Generation.Phase)
}

func TestNovaGeneration_ReviewsOneAtATimeWhenTheProviderCannotBatch(t *testing.T) {
	t.Parallel()

	env := novaEnv(t)
	var a *Activities
	env.OnActivity(a.SubmitNovaReviewsActivity, mock.Anything, mock.Anything).Return(
		&billingqueueports.ReviewSubmission{Inline: []pulid.ID{"bqi_1", "bqi_2"}}, nil,
	).Once()
	env.OnActivity(a.AgentReviewActivity, mock.Anything, mock.Anything).
		Return(&billingqueueports.AgentReviewOutcome{Reviewed: true}, nil).Twice()

	env.ExecuteWorkflow(BillingQueueNovaGenerationWorkflow, novaPayload())

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
	env.AssertNotCalled(t, "NovaBatchEndedActivity", mock.Anything, mock.Anything)
	assert.Equal(t, 2, progressOf(t, env).Done)
}

func TestNovaGeneration_ReviewsAtOnceForSomeoneWaitingAndOffersWhatNovaAgreesWithToAutoApprove(
	t *testing.T,
) {
	t.Parallel()

	env := novaEnv(t)
	var a *Activities
	agreed, questioned := pulid.ID("bqi_agreed"), pulid.ID("bqi_questioned")
	env.OnActivity(a.SubmitNovaReviewsActivity, mock.Anything, mock.MatchedBy(func(in *SubmitNovaReviewsInput) bool {
		return in.Inline
	})).
		Return(&billingqueueports.ReviewSubmission{Inline: []pulid.ID{agreed, questioned}}, nil).
		Once()
	env.OnActivity(a.AgentReviewActivity, mock.Anything, mock.MatchedBy(func(in *billingqueuecontract.ItemPayload) bool {
		return in.ItemID == agreed
	})).
		Return(&billingqueueports.AgentReviewOutcome{Reviewed: true, Verdict: billingqueue.AgentVerdictAgrees}, nil).
		Once()
	env.OnActivity(a.AgentReviewActivity, mock.Anything, mock.MatchedBy(func(in *billingqueuecontract.ItemPayload) bool {
		return in.ItemID == questioned
	})).
		Return(&billingqueueports.AgentReviewOutcome{Reviewed: true, Verdict: billingqueue.AgentVerdictConcern}, nil).
		Once()
	env.OnActivity(a.AutoApproveActivity, mock.Anything, mock.MatchedBy(func(in *billingqueuecontract.ItemPayload) bool {
		return in.ItemID == agreed
	})).
		Return(&billingqueueports.AutoApproveOutcome{}, nil).
		Once()

	payload := novaPayload()
	payload.Watched = true
	env.ExecuteWorkflow(BillingQueueNovaGenerationWorkflow, payload)

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
	env.AssertNumberOfCalls(t, "AutoApproveActivity", 1)
	env.AssertNotCalled(t, "NovaBatchEndedActivity", mock.Anything, mock.Anything)
	assert.Equal(t, 2, progressOf(t, env).Done)
}

func TestNovaGeneration_ReviewsSeveralItemsAtATime(t *testing.T) {
	t.Parallel()

	env := novaEnv(t)
	var a *Activities
	items := make([]pulid.ID, novaInlineWave+1)
	for idx := range items {
		items[idx] = pulid.MustNew("bqi_")
	}
	env.OnActivity(a.SubmitNovaReviewsActivity, mock.Anything, mock.Anything).
		Return(&billingqueueports.ReviewSubmission{Inline: items}, nil).Once()
	env.OnActivity(a.AgentReviewActivity, mock.Anything, mock.Anything).
		Return(&billingqueueports.AgentReviewOutcome{Reviewed: true}, nil).Times(len(items))

	payload := novaPayload()
	payload.Watched = true
	env.ExecuteWorkflow(BillingQueueNovaGenerationWorkflow, payload)

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
	// One publish as the page is counted, one per wave, one each for the brief and the end.
	env.AssertNumberOfCalls(t, "PublishNovaProgressActivity", 1+2+2)
}

func TestNovaGeneration_WritesOnlyTheBriefWhenEveryItemIsReviewed(t *testing.T) {
	t.Parallel()

	env := novaEnv(t)
	var a *Activities
	env.OnActivity(a.SubmitNovaReviewsActivity, mock.Anything, mock.Anything).
		Return(&billingqueueports.ReviewSubmission{}, nil).Once()

	env.ExecuteWorkflow(BillingQueueNovaGenerationWorkflow, novaPayload())

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
	assert.Equal(
		t,
		billingqueueports.NovaGeneration{Phase: billingqueueports.NovaPhaseDone},
		progressOf(t, env),
	)
}

func TestNovaGeneration_TakesAnotherPassWhenItemsWereLeftOver(t *testing.T) {
	t.Parallel()

	env := novaEnv(t)
	var a *Activities
	page := func(next *billingqueueports.ReviewCursor) *billingqueueports.ReviewSubmission {
		return &billingqueueports.ReviewSubmission{Inline: []pulid.ID{"bqi_1"}, Next: next}
	}
	cursor := &billingqueueports.ReviewCursor{CreatedAt: 1, ID: "bqi_1"}
	env.OnActivity(a.SubmitNovaReviewsActivity, mock.Anything, mock.Anything).
		Return(page(cursor), nil).Times(novaPagesPerPass)
	env.OnActivity(a.SubmitNovaReviewsActivity, mock.Anything, mock.Anything).
		Return(page(nil), nil).Once()
	env.OnActivity(a.AgentReviewActivity, mock.Anything, mock.Anything).
		Return(&billingqueueports.AgentReviewOutcome{}, nil)
	env.OnActivity(a.RefreshBriefActivity, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(BillingQueueNovaGenerationWorkflow, novaPayload())

	require.NoError(t, env.GetWorkflowError())
	env.AssertNumberOfCalls(t, "SubmitNovaReviewsActivity", novaPagesPerPass+1)
	env.AssertNumberOfCalls(t, "RefreshBriefActivity", 2)
}
