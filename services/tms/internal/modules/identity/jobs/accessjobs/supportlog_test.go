package accessjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/testutil/entitlementtest"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/supportlogservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

type scriptedLogTimeouts struct {
	calls int
}

func (s *scriptedLogTimeouts) FailOverdue(
	_ context.Context,
	now int64,
) (*supportlogservice.SweepResult, error) {
	s.calls++
	if now <= 0 {
		return nil, errors.New("the sweep needs the current time")
	}
	return &supportlogservice.SweepResult{Failed: 1}, nil
}

func TestSchedules_FailOverdueLogRequestsOnlyOnAManagedInstance(t *testing.T) {
	t.Parallel()

	ids := func(managed bool) []string {
		var out []string
		for _, s := range NewScheduleProvider(&managedconfig.Config{Enabled: managed}).GetSchedules() {
			out = append(out, s.ID)
		}
		return out
	}

	assert.NotContains(t, ids(false), SupportLogTimeoutScheduleID)
	assert.Contains(t, ids(true), SupportLogTimeoutScheduleID)
}

func TestSupportLogTimeoutWorkflow_ReportsWhatItFailed(t *testing.T) {
	t.Parallel()

	timeouts := &scriptedLogTimeouts{}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(temporaljobs.NewEntitledActivities(entitlementtest.Instance()))
	env.RegisterActivity(NewActivities(ActivitiesParams{SupportLogTimeouts: timeouts}))
	env.RegisterWorkflow(FailOverdueSupportLogRequestsWorkflow)
	env.ExecuteWorkflow(FailOverdueSupportLogRequestsWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result supportlogservice.SweepResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 1, result.Failed)
	assert.Equal(t, 1, timeouts.calls)
}
