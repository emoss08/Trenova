package billingqueuejobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/jobs/billingqueuejobs/billingqueuecontract"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func tenantPayload() *billingqueuecontract.TenantPayload {
	return &billingqueuecontract.TenantPayload{
		OrganizationID: pulid.ID("org_test"),
		BusinessUnitID: pulid.ID("bu_test"),
	}
}

func newTenantEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(BillingQueueTenantWorkflow)

	var a *Activities
	env.RegisterActivity(a.DrainRechecksActivity)
	env.RegisterActivity(a.RefreshBriefActivity)

	return env
}

// pages answers the drain with one page after another, then empty pages.
func pages(env *testsuite.TestWorkflowEnvironment, results ...DrainRechecksResult) *int {
	calls := 0
	env.OnActivity("DrainRechecksActivity", mock.Anything, mock.Anything).
		Return(func(context.Context, *billingqueuecontract.TenantPayload) (*DrainRechecksResult, error) {
			calls++
			if calls <= len(results) {
				page := results[calls-1]
				return &page, nil
			}
			return &DrainRechecksResult{}, nil
		})

	return &calls
}

func briefs(env *testsuite.TestWorkflowEnvironment) *int {
	calls := 0
	env.OnActivity("RefreshBriefActivity", mock.Anything, mock.Anything).
		Return(func(context.Context, *billingqueuecontract.TenantPayload) error {
			calls++
			return nil
		})

	return &calls
}

// A burst of changes is one round of checks and one brief, however many
// items it marked due, and the workflow ends once nothing else arrives.
func TestTenantWorkflowChecksABurstOnceAndWritesOneBrief(t *testing.T) {
	t.Parallel()

	env := newTenantEnv(t)
	drains := pages(env, DrainRechecksResult{Checked: 7})
	written := briefs(env)

	env.ExecuteWorkflow(BillingQueueTenantWorkflow, tenantPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result TenantWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 7, result.Checked)
	assert.Equal(t, 1, *drains, "a page that is not full ends the round")
	assert.Equal(t, 1, *written)
}

// A full page means more items are due, so the drain keeps paging within the
// round instead of waiting out another debounce.
func TestTenantWorkflowPagesThroughABacklog(t *testing.T) {
	t.Parallel()

	env := newTenantEnv(t)
	drains := pages(env,
		DrainRechecksResult{Checked: drainPageSize, Full: true},
		DrainRechecksResult{Checked: drainPageSize, Full: true},
		DrainRechecksResult{Checked: 3},
	)
	written := briefs(env)

	env.ExecuteWorkflow(BillingQueueTenantWorkflow, tenantPayload())

	require.NoError(t, env.GetWorkflowError())
	var result TenantWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 2*drainPageSize+3, result.Checked)
	assert.Equal(t, 3, *drains)
	assert.Equal(t, 1, *written)
}

// A brief asked for with nothing due is still written; nothing checked and
// nothing asked for writes none.
func TestTenantWorkflowWritesABriefOnlyWhenOneIsOwed(t *testing.T) {
	t.Parallel()

	t.Run("asked for", func(t *testing.T) {
		t.Parallel()

		env := newTenantEnv(t)
		pages(env)
		written := briefs(env)
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(billingqueuecontract.TenantSignalName,
				billingqueuecontract.TenantSignal{Brief: true})
		}, time.Second)

		env.ExecuteWorkflow(BillingQueueTenantWorkflow, tenantPayload())

		require.NoError(t, env.GetWorkflowError())
		assert.Equal(t, 1, *written)
	})

	t.Run("nothing owed", func(t *testing.T) {
		t.Parallel()

		env := newTenantEnv(t)
		pages(env)
		written := briefs(env)

		env.ExecuteWorkflow(BillingQueueTenantWorkflow, tenantPayload())

		require.NoError(t, env.GetWorkflowError())
		assert.Zero(t, *written)
	})
}

// A change arriving while a round's checks run starts another round, and the brief keeps its own pace: the second waits out briefEvery
// rather than being written on every round.
func TestTenantWorkflowKeepsTheBriefToItsOwnPace(t *testing.T) {
	t.Parallel()

	env := newTenantEnv(t)
	drains := 0
	env.OnActivity("DrainRechecksActivity", mock.Anything, mock.Anything).
		After(5 * time.Second).
		Return(func(context.Context, *billingqueuecontract.TenantPayload) (*DrainRechecksResult, error) {
			drains++
			return &DrainRechecksResult{Checked: 1}, nil
		})
	var writtenAt []time.Time
	env.OnActivity("RefreshBriefActivity", mock.Anything, mock.Anything).
		Return(func(context.Context, *billingqueuecontract.TenantPayload) error {
			writtenAt = append(writtenAt, env.Now())
			return nil
		})
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(billingqueuecontract.TenantSignalName, billingqueuecontract.TenantSignal{})
	}, tenantDebounce+time.Second)

	env.ExecuteWorkflow(BillingQueueTenantWorkflow, tenantPayload())

	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, 2, drains, "the change that arrived during the drain ran a second round")
	require.Len(t, writtenAt, 2)
	assert.GreaterOrEqual(t, writtenAt[1].Sub(writtenAt[0]), briefEvery)
}

// A drain that cannot read the due items fails the workflow, visibly; the
// items stay due and the sweep wakes the tenant again.
func TestTenantWorkflowFailsWhenTheDrainCannotRun(t *testing.T) {
	t.Parallel()

	env := newTenantEnv(t)
	env.OnActivity("DrainRechecksActivity", mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError("database unavailable", "Outage", errors.New("down")))
	briefs(env)

	env.ExecuteWorkflow(BillingQueueTenantWorkflow, tenantPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}
