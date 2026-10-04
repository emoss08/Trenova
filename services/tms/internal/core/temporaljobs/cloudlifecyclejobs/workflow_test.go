package cloudlifecyclejobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/cloudlifecycleservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func ref(org, bu string) cloudlifecycleservice.TenantRef {
	return cloudlifecycleservice.TenantRef{
		OrganizationID: pulid.ID(org),
		BusinessUnitID: pulid.ID(bu),
	}
}

func TestSweepWorkflowStartsAPurgeForEveryExpiredOrganization(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})
	env.RegisterWorkflow(CloudTenantPurgeWorkflow)

	var a *Activities
	env.OnActivity(a.SweepCloudSubscriptionsActivity, mock.Anything, mock.Anything).
		Return(&cloudlifecycleservice.SweepResult{
			Examined:     3,
			ReadOnly:     1,
			Expired:      1,
			Failures:     []string{},
			PurgeTargets: []cloudlifecycleservice.TenantRef{ref("org_a", "bu_a"), ref("org_b", "bu_b")},
		}, nil).
		Once()
	env.OnActivity(a.CheckCloudTenantPurgeActivity, mock.Anything, mock.Anything).
		Return(&PurgeEligibility{Members: []*repositories.TenantMember{}}, nil)

	env.ExecuteWorkflow(CloudSubscriptionSweepWorkflow, &SweepInput{})

	require.NoError(t, env.GetWorkflowError())
	var result SweepResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 2, result.PurgesStarted)
	assert.Equal(t, 1, result.ReadOnly)
	assert.Equal(t, 1, result.Expired)
}

func TestSweepWorkflowFailsWhenTheSweepActivityFails(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})
	env.RegisterWorkflow(CloudTenantPurgeWorkflow)

	var a *Activities
	env.OnActivity(a.SweepCloudSubscriptionsActivity, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	env.ExecuteWorkflow(CloudSubscriptionSweepWorkflow, &SweepInput{Now: 100})

	require.Error(t, env.GetWorkflowError())
}

func TestPurgeWorkflowSkipsAnOrganizationThatIsNotExpired(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.CheckCloudTenantPurgeActivity, mock.Anything, mock.Anything).
		Return(&PurgeEligibility{Members: []*repositories.TenantMember{}}, nil).
		Once()

	env.ExecuteWorkflow(CloudTenantPurgeWorkflow, &PurgePayload{
		OrganizationID: "org_a",
		BusinessUnitID: "bu_a",
	})

	require.NoError(t, env.GetWorkflowError())
	var result PurgeResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.True(t, result.Skipped)
	assert.Nil(t, result.Rows)
}

func TestPurgeWorkflowRunsEveryStepInOrder(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	steps := make([]string, 0, 5)
	member := pulid.ID("usr_owner")
	var a *Activities
	env.OnActivity(a.CheckCloudTenantPurgeActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *PurgePayload) (*PurgeEligibility, error) {
			steps = append(steps, "check")
			return &PurgeEligibility{
				Eligible: true,
				Members:  []*repositories.TenantMember{{UserID: member}, nil},
			}, nil
		})
	env.OnActivity(a.PurgeCloudTenantRowsActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *PurgePayload) (*PurgeRowsResult, error) {
			steps = append(steps, "rows")
			return &PurgeRowsResult{Deleted: 42, Complete: true}, nil
		})
	env.OnActivity(a.PurgeCloudTenantStorageActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *PurgePayload) (*PurgeStorageResult, error) {
			steps = append(steps, "storage")
			return &PurgeStorageResult{Deleted: 3}, nil
		})
	env.OnActivity(a.PurgeCloudTenantUsersActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *PurgeUsersInput) (*cloudlifecycleservice.PurgeUsersResult, error) {
			steps = append(steps, "users")
			assert.Equal(t, []pulid.ID{member}, input.UserIDs)
			return &cloudlifecycleservice.PurgeUsersResult{Deleted: 1}, nil
		})
	env.OnActivity(a.FinalizeCloudTenantPurgeActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *PurgePayload) (*repositories.DeleteTenantResult, error) {
			steps = append(steps, "finalize")
			return &repositories.DeleteTenantResult{OrganizationDeleted: true}, nil
		})

	env.ExecuteWorkflow(CloudTenantPurgeWorkflow, &PurgePayload{
		OrganizationID: "org_a",
		BusinessUnitID: "bu_a",
	})

	require.NoError(t, env.GetWorkflowError())
	var result PurgeResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, []string{"check", "rows", "storage", "users", "finalize"}, steps)
	assert.Equal(t, int64(42), result.Rows.Deleted)
	assert.True(t, result.Tenant.OrganizationDeleted)
	assert.Equal(t, 1, result.Users.Deleted)
}

func TestPurgeWorkflowIDIsStablePerOrganizationAndDay(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "cloud-tenant-purge/org_a/20261004", PurgeWorkflowID("org_a", "20261004"))
}

func TestScheduleProviderRegistersTheSweepOnlyInCloudMode(t *testing.T) {
	t.Parallel()

	assert.Empty(t, (&ScheduleProvider{}).GetSchedules())

	schedules := (&ScheduleProvider{cloud: true}).GetSchedules()
	require.Len(t, schedules, 1)
	assert.Equal(t, sweepScheduleID, schedules[0].ID)
	assert.Equal(t, "7 * * * *", schedules[0].Spec.Cron)
}
