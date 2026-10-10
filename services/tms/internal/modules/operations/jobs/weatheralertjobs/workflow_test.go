package weatheralertjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestPollNWSAlertsWorkflow_PollsOnceForEveryTenantThenExpires(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	polled := &PollNWSAlertsResult{AlertsInFeed: 12}
	polled.TenantsScanned = 3
	polled.AddTenantResult(4, 8)
	polled.AddTenantResult(0, 12)
	polled.AddFailure(temporaljobs.TenantWorkItem{OrganizationID: pulid.ID("org_b")}, errors.New("db"))

	var a *Activities
	env.OnActivity(a.PollNWSAlertsActivity, mock.Anything).Return(polled, nil).Once()
	env.OnActivity(a.ExpireStaleWeatherAlertsActivity, mock.Anything).Return(nil).Once()

	env.ExecuteWorkflow(PollNWSAlertsWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
	env.AssertNotCalled(t, "ListWeatherAlertTenantsActivity", mock.Anything, mock.Anything)
	env.AssertNotCalled(t, "PollNWSAlertsForTenantActivity", mock.Anything, mock.Anything)

	var result PollNWSAlertsResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 3, result.TenantsScanned)
	assert.Equal(t, 2, result.TenantsProcessed)
	assert.Equal(t, 4, result.RecordsProcessed)
	assert.Equal(t, 20, result.SkippedCount)
	assert.Equal(t, 1, result.FailureCount)
	assert.Equal(t, 12, result.AlertsInFeed)
}

func TestPollNWSAlertsWorkflow_StillExpiresWhenTheFeedIsUnchanged(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	unchanged := &PollNWSAlertsResult{FeedUnchanged: true}
	unchanged.TenantsScanned = 7

	var a *Activities
	env.OnActivity(a.PollNWSAlertsActivity, mock.Anything).Return(unchanged, nil).Once()
	env.OnActivity(a.ExpireStaleWeatherAlertsActivity, mock.Anything).Return(nil).Once()

	env.ExecuteWorkflow(PollNWSAlertsWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)

	var result PollNWSAlertsResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.True(t, result.FeedUnchanged)
	assert.Equal(t, 7, result.TenantsScanned)
}

func TestPollNWSAlertsWorkflow_FailsWhenThePollFails(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.PollNWSAlertsActivity, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError("feed down", "NWS", nil))

	env.ExecuteWorkflow(PollNWSAlertsWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "ExpireStaleWeatherAlertsActivity", mock.Anything)
}

func TestPollNWSAlertsWorkflow_RunsStartedBeforeFetchOnceKeepPollingPerTenant(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})
	env.OnGetVersion(fetchOnceChange, workflow.DefaultVersion, 1).Return(workflow.DefaultVersion)

	tenants := []temporaljobs.TenantWorkItem{
		{OrganizationID: pulid.ID("org_a"), BusinessUnitID: pulid.ID("bu_a")},
		{OrganizationID: pulid.ID("org_b"), BusinessUnitID: pulid.ID("bu_b")},
	}

	var a *Activities
	env.OnActivity(a.ListWeatherAlertTenantsActivity, mock.Anything, mock.Anything).
		Return(&ListWeatherAlertTenantsResult{Tenants: tenants}, nil).
		Once()
	env.OnActivity(a.PollNWSAlertsForTenantActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, payload *PollNWSAlertsTenantPayload) error {
			if payload.OrganizationID == "org_b" {
				return temporal.NewNonRetryableApplicationError("db", "Broken", nil)
			}
			return nil
		})
	env.OnActivity(a.ExpireStaleWeatherAlertsActivity, mock.Anything).Return(nil).Once()

	env.ExecuteWorkflow(PollNWSAlertsWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
	env.AssertNotCalled(t, "PollNWSAlertsActivity", mock.Anything)

	var result PollNWSAlertsResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 2, result.TenantsScanned)
	assert.Equal(t, 1, result.TenantsProcessed)
	assert.Equal(t, 1, result.FailureCount)
	require.Len(t, result.PartialFailures, 1)
	assert.Equal(t, pulid.ID("org_b"), result.PartialFailures[0].OrganizationID)
}
