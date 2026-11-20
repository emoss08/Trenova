package accessjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/testutil/entitlementtest"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/supportaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

type scriptedSweeper struct {
	calls  int
	result *userservice.OrganizationAccessSweepResult
	err    error
}

func (s *scriptedSweeper) SweepOrphanedOrganizationAccess(
	_ context.Context,
	now int64,
) (*userservice.OrganizationAccessSweepResult, error) {
	s.calls++
	if now <= 0 {
		return nil, errors.New("the sweep needs the current time")
	}
	return s.result, s.err
}

func runSweep(t *testing.T, sweeper *scriptedSweeper) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(temporaljobs.NewEntitledActivities(entitlementtest.Instance()))
	env.RegisterActivity(NewActivities(ActivitiesParams{Sweeper: sweeper}))
	env.RegisterWorkflow(SweepOrphanedOrganizationAccessWorkflow)
	env.ExecuteWorkflow(SweepOrphanedOrganizationAccessWorkflow)

	return env
}

func TestSchedules_MatchTheirWorkflowSignatures(t *testing.T) {
	t.Parallel()

	for _, managed := range []bool{false, true} {
		for _, s := range NewScheduleProvider(&managedconfig.Config{Enabled: managed}).GetSchedules() {
			require.NoError(t, s.Validate(), s.ID)
		}
	}
}

func TestSchedules_RecordSupportAccessExpiriesOnlyOnAManagedInstance(t *testing.T) {
	t.Parallel()

	ids := func(managed bool) []string {
		var out []string
		for _, s := range NewScheduleProvider(&managedconfig.Config{Enabled: managed}).GetSchedules() {
			out = append(out, s.ID)
		}
		return out
	}

	assert.NotContains(t, ids(false), SupportExpiryScheduleID)
	assert.Contains(t, ids(true), SupportExpiryScheduleID)
}

type scriptedExpiries struct {
	calls int
}

func (s *scriptedExpiries) RecordExpiries(
	_ context.Context,
	now int64,
) (*supportaccessservice.ExpirySweepResult, error) {
	s.calls++
	if now <= 0 {
		return nil, errors.New("the record needs the current time")
	}
	return &supportaccessservice.ExpirySweepResult{Recorded: 2}, nil
}

func TestSupportExpiryWorkflow_ReportsWhatItRecorded(t *testing.T) {
	t.Parallel()

	expiries := &scriptedExpiries{}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(temporaljobs.NewEntitledActivities(entitlementtest.Instance()))
	env.RegisterActivity(NewActivities(ActivitiesParams{SupportExpiries: expiries}))
	env.RegisterWorkflow(RecordSupportAccessExpiriesWorkflow)
	env.ExecuteWorkflow(RecordSupportAccessExpiriesWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result supportaccessservice.ExpirySweepResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 2, result.Recorded)
	assert.Equal(t, 1, expiries.calls)
}

func TestSweepWorkflow_ReportsWhatItRevoked(t *testing.T) {
	t.Parallel()

	sweeper := &scriptedSweeper{result: &userservice.OrganizationAccessSweepResult{Revoked: 3}}
	env := runSweep(t, sweeper)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result userservice.OrganizationAccessSweepResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 3, result.Revoked)
	assert.Equal(t, 1, sweeper.calls)
}

func TestSweepWorkflow_RetriesAFailedSweep(t *testing.T) {
	t.Parallel()

	sweeper := &scriptedSweeper{err: errors.New("database unavailable")}
	env := runSweep(t, sweeper)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	assert.Equal(t, int(sweepOptions.RetryPolicy.MaximumAttempts), sweeper.calls)
}
