package billingqueuejobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func testPayload() *ApprovalRunPayload {
	return &ApprovalRunPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.ID("org_test"),
			BusinessUnitID: pulid.ID("bu_test"),
			UserID:         pulid.ID("usr_test"),
		},
		RunID: pulid.ID("bqar_test"),
	}
}

func newEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(BulkApprovalWorkflow)

	var a *Activities
	env.RegisterActivity(a.StartApprovalRunActivity)
	env.RegisterActivity(a.ListApprovalRunItemsActivity)
	env.RegisterActivity(a.ApproveQueueItemActivity)
	env.RegisterActivity(a.RecordApprovalItemFailureActivity)
	env.RegisterActivity(a.FinalizeApprovalRunActivity)

	return env
}

func captureFinalize(env *testsuite.TestWorkflowEnvironment) *FinalizeApprovalRunPayload {
	finalized := &FinalizeApprovalRunPayload{}
	env.OnActivity("FinalizeApprovalRunActivity", mock.Anything, mock.Anything).
		Return(func(_ context.Context, p *FinalizeApprovalRunPayload) (*ApprovalRunResult, error) {
			*finalized = *p
			return &ApprovalRunResult{RunID: p.RunID, Status: p.Status}, nil
		})

	return finalized
}

// Nothing is approved until the undo window has passed, and then every item
// is approved in turn.
func TestBulkApprovalCommitsAfterTheUndoWindow(t *testing.T) {
	env := newEnv(t)
	items := []pulid.ID{"bqi_1", "bqi_2", "bqi_3"}

	var startedAt time.Time
	env.OnActivity("StartApprovalRunActivity", mock.Anything, mock.Anything).
		Return(func(context.Context, *ApprovalRunPayload) (bool, error) {
			startedAt = env.Now()
			return true, nil
		})
	env.OnActivity("ListApprovalRunItemsActivity", mock.Anything, mock.Anything).Return(items, nil)
	approved := make([]pulid.ID, 0, len(items))
	env.OnActivity("ApproveQueueItemActivity", mock.Anything, mock.Anything).
		Return(func(_ context.Context, p *ApproveItemPayload) (*ApproveItemResult, error) {
			approved = append(approved, p.ItemID)
			return &ApproveItemResult{Status: billingqueue.ApprovalItemApproved}, nil
		})
	finalized := captureFinalize(env)

	began := env.Now()
	env.ExecuteWorkflow(BulkApprovalWorkflow, testPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.GreaterOrEqual(t, startedAt.Sub(began), billingqueue.ApprovalUndoWindowSeconds*time.Second)
	assert.Equal(t, items, approved)
	assert.Equal(t, billingqueue.ApprovalRunCompleted, finalized.Status)
}

// An undo inside the window ends the run without touching a single item.
func TestBulkApprovalUndoneBeforeItStartsWritesNothing(t *testing.T) {
	env := newEnv(t)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(UndoSignalName, nil)
	}, 2*time.Second)

	finalized := captureFinalize(env)

	env.ExecuteWorkflow(BulkApprovalWorkflow, testPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, billingqueue.ApprovalRunUndone, finalized.Status)
	assert.Equal(t, billingqueue.ApprovalFailureUndone, finalized.LeftoverCode)
	env.AssertNotCalled(t, "StartApprovalRunActivity", mock.Anything, mock.Anything)
	env.AssertNotCalled(t, "ApproveQueueItemActivity", mock.Anything, mock.Anything)
}

// The undo endpoint writes the row before it signals. When the signal is lost
// the row still wins: the run does not start.
func TestBulkApprovalObeysAnUndoWrittenWithoutASignal(t *testing.T) {
	env := newEnv(t)
	env.OnActivity("StartApprovalRunActivity", mock.Anything, mock.Anything).Return(false, nil)
	finalized := captureFinalize(env)

	env.ExecuteWorkflow(BulkApprovalWorkflow, testPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, billingqueue.ApprovalRunUndone, finalized.Status)
	env.AssertNotCalled(t, "ApproveQueueItemActivity", mock.Anything, mock.Anything)
}

// One item failing past its retries is recorded against that item and the
// run carries on with the rest.
func TestBulkApprovalRecordsAPartialFailureAndCarriesOn(t *testing.T) {
	env := newEnv(t)
	items := []pulid.ID{"bqi_1", "bqi_2", "bqi_3"}

	env.OnActivity("StartApprovalRunActivity", mock.Anything, mock.Anything).Return(true, nil)
	env.OnActivity("ListApprovalRunItemsActivity", mock.Anything, mock.Anything).Return(items, nil)
	attempts := map[pulid.ID]int{}
	env.OnActivity("ApproveQueueItemActivity", mock.Anything, mock.Anything).
		Return(func(_ context.Context, p *ApproveItemPayload) (*ApproveItemResult, error) {
			attempts[p.ItemID]++
			if p.ItemID == "bqi_2" {
				return nil, errors.New("database went away")
			}
			return &ApproveItemResult{Status: billingqueue.ApprovalItemApproved}, nil
		})
	var failed *RecordItemFailurePayload
	env.OnActivity("RecordApprovalItemFailureActivity", mock.Anything, mock.Anything).
		Return(func(_ context.Context, p *RecordItemFailurePayload) error {
			failed = p
			return nil
		})
	finalized := captureFinalize(env)

	env.ExecuteWorkflow(BulkApprovalWorkflow, testPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, 3, attempts["bqi_2"], "a fault is retried before it is given up on")
	assert.Equal(t, 1, attempts["bqi_3"], "the run went on past the failure")
	require.NotNil(t, failed)
	assert.Equal(t, pulid.ID("bqi_2"), failed.ItemID)
	assert.Equal(t, billingqueue.ApprovalRunCompleted, finalized.Status)
}

func TestBulkApprovalFinalizesAsFailedWhenItCannotStart(t *testing.T) {
	env := newEnv(t)
	env.OnActivity("StartApprovalRunActivity", mock.Anything, mock.Anything).
		Return(false, errors.New("no database"))
	finalized := captureFinalize(env)

	env.ExecuteWorkflow(BulkApprovalWorkflow, testPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	assert.Equal(t, billingqueue.ApprovalRunFailed, finalized.Status)
}
