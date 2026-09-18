package billingtransferjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func testPayload() *TransferRunPayload {
	return &TransferRunPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.ID("org_test"),
			BusinessUnitID: pulid.ID("bu_test"),
			UserID:         pulid.ID("usr_test"),
		},
		RunID: pulid.ID("btr_test"),
	}
}

func testPrepared(totalCount int) *PreparedRun {
	return &PreparedRun{
		RunID:         pulid.ID("btr_test"),
		RequestedByID: pulid.ID("usr_test"),
		TotalCount:    totalCount,
		BatchSize:     BatchSize,
	}
}

func newEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(BulkBillingTransferWorkflow)

	var a *Activities
	env.RegisterActivity(a.PrepareRunActivity)
	env.RegisterActivity(a.ProcessBatchActivity)
	env.RegisterActivity(a.FinalizeRunActivity)

	return env
}

func finalizedResult(status billingtransfer.RunStatus) *TransferRunResult {
	return &TransferRunResult{RunID: pulid.ID("btr_test"), Status: status}
}

// A run keeps asking for batches until one answers for nothing. That, and not a
// count the workflow carries, is what decides when it is done — so a batch that
// transfers fewer shipments than it claimed cannot strand the rest.
func TestWorkflowRunsUntilABatchAnswersForNothing(t *testing.T) {
	env := newEnv(t)

	env.OnActivity("PrepareRunActivity", mock.Anything, mock.Anything).
		Return(testPrepared(60), nil)

	batches := 0
	env.OnActivity("ProcessBatchActivity", mock.Anything, mock.Anything).
		Return(func(context.Context, *ProcessBatchPayload) (*ProcessBatchResult, error) {
			batches++
			if batches > 2 {
				return &ProcessBatchResult{}, nil
			}
			return &ProcessBatchResult{Processed: 25, Transferred: 20, NotTransferred: 5}, nil
		})

	var finalized *FinalizeRunPayload
	env.OnActivity("FinalizeRunActivity", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			finalized = args.Get(1).(*FinalizeRunPayload)
		}).
		Return(finalizedResult(billingtransfer.RunStatusCompleted), nil)

	env.ExecuteWorkflow(BulkBillingTransferWorkflow, testPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, 3, batches, "the empty batch is what ends the loop")

	require.NotNil(t, finalized)
	assert.Equal(t, billingtransfer.RunStatusCompleted, finalized.Status)
}

// The cancel flag the mutation writes reaches the workflow through the batch
// result, so a stop works even when the signal never arrives.
func TestWorkflowStopsWhenABatchReportsCancelRequested(t *testing.T) {
	env := newEnv(t)

	env.OnActivity("PrepareRunActivity", mock.Anything, mock.Anything).
		Return(testPrepared(100), nil)

	batches := 0
	env.OnActivity("ProcessBatchActivity", mock.Anything, mock.Anything).
		Return(func(context.Context, *ProcessBatchPayload) (*ProcessBatchResult, error) {
			batches++
			return &ProcessBatchResult{
				Processed:       25,
				Transferred:     25,
				CancelRequested: batches == 2,
			}, nil
		})

	var finalized *FinalizeRunPayload
	env.OnActivity("FinalizeRunActivity", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			finalized = args.Get(1).(*FinalizeRunPayload)
		}).
		Return(finalizedResult(billingtransfer.RunStatusCanceled), nil)

	env.ExecuteWorkflow(BulkBillingTransferWorkflow, testPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, 2, batches, "the batch in flight finishes; the next one never starts")

	require.NotNil(t, finalized)
	assert.Equal(t, billingtransfer.RunStatusCanceled, finalized.Status)
}

// The signal is the fast path for the same stop. It is read between batches
// only: a batch that has started has already moved shipments into the queue,
// and abandoning it would leave those shipments out of the run's own report.
func TestWorkflowStopsOnCancelSignalBetweenBatches(t *testing.T) {
	env := newEnv(t)

	env.OnActivity("PrepareRunActivity", mock.Anything, mock.Anything).
		Return(testPrepared(100), nil)

	batches := 0
	env.OnActivity("ProcessBatchActivity", mock.Anything, mock.Anything).
		Return(func(context.Context, *ProcessBatchPayload) (*ProcessBatchResult, error) {
			batches++
			return &ProcessBatchResult{Processed: 25, Transferred: 25}, nil
		})

	var finalized *FinalizeRunPayload
	env.OnActivity("FinalizeRunActivity", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			finalized = args.Get(1).(*FinalizeRunPayload)
		}).
		Return(finalizedResult(billingtransfer.RunStatusCanceled), nil)

	// Delivered before the loop's first turn, so the run stops without touching
	// a single shipment — which is the whole point of reading it up there.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CancelSignalName, nil)
	}, 0)

	env.ExecuteWorkflow(BulkBillingTransferWorkflow, testPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, 0, batches, "a cancel that arrives first transfers nothing")

	require.NotNil(t, finalized)
	assert.Equal(t, billingtransfer.RunStatusCanceled, finalized.Status)
}

// A run must never be left sitting in Running with nobody coming back for it,
// so every failing path still finalizes.
func TestWorkflowFinalizesWhenPrepareFails(t *testing.T) {
	env := newEnv(t)

	env.OnActivity("PrepareRunActivity", mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError(
			"the transfer has already finished", ErrTypeTransferValidation, nil,
		))

	var finalized *FinalizeRunPayload
	env.OnActivity("FinalizeRunActivity", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			finalized = args.Get(1).(*FinalizeRunPayload)
		}).
		Return(finalizedResult(billingtransfer.RunStatusFailed), nil)

	env.ExecuteWorkflow(BulkBillingTransferWorkflow, testPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())

	require.NotNil(t, finalized)
	assert.Equal(t, billingtransfer.RunStatusFailed, finalized.Status)
	assert.Equal(t, "the transfer has already finished", finalized.FailureMessage)
	env.AssertNotCalled(t, "ProcessBatchActivity")
}

func TestWorkflowFinalizesWhenABatchFails(t *testing.T) {
	env := newEnv(t)

	env.OnActivity("PrepareRunActivity", mock.Anything, mock.Anything).
		Return(testPrepared(50), nil)
	env.OnActivity("ProcessBatchActivity", mock.Anything, mock.Anything).
		Return(nil, errors.New("the database went away"))

	var finalized *FinalizeRunPayload
	env.OnActivity("FinalizeRunActivity", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			finalized = args.Get(1).(*FinalizeRunPayload)
		}).
		Return(finalizedResult(billingtransfer.RunStatusFailed), nil)

	env.ExecuteWorkflow(BulkBillingTransferWorkflow, testPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())

	require.NotNil(t, finalized)
	assert.Equal(t, billingtransfer.RunStatusFailed, finalized.Status)
	assert.NotEmpty(t, finalized.FailureMessage)
}

// A scope-AllMatching run whose search matched nothing has nothing to do. It
// must still finish cleanly rather than asking for a batch that cannot exist.
func TestWorkflowCompletesWithoutBatchesWhenNothingMatched(t *testing.T) {
	env := newEnv(t)

	env.OnActivity("PrepareRunActivity", mock.Anything, mock.Anything).
		Return(testPrepared(0), nil)

	batches := 0
	env.OnActivity("ProcessBatchActivity", mock.Anything, mock.Anything).
		Return(func(context.Context, *ProcessBatchPayload) (*ProcessBatchResult, error) {
			batches++
			return &ProcessBatchResult{}, nil
		})

	var finalized *FinalizeRunPayload
	env.OnActivity("FinalizeRunActivity", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			finalized = args.Get(1).(*FinalizeRunPayload)
		}).
		Return(finalizedResult(billingtransfer.RunStatusCompleted), nil)

	env.ExecuteWorkflow(BulkBillingTransferWorkflow, testPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.LessOrEqual(t, batches, 2, "an empty run does not grind through batches")

	require.NotNil(t, finalized)
	assert.Equal(t, billingtransfer.RunStatusCompleted, finalized.Status)
}

// A schedule whose args do not match its workflow signature fails the whole
// worker at startup, taking every other domain's jobs down with it.
func TestSchedulesAreValid(t *testing.T) {
	t.Parallel()

	schedules := NewScheduleProvider().GetSchedules()
	require.NotEmpty(t, schedules)

	for _, s := range schedules {
		require.NoErrorf(t, s.Validate(), "schedule %q", s.ID)
	}
}
