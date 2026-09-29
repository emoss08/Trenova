package aicorrectionjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

func TestExtractionAccuracyDriftWorkflow_ChecksEachOrganization(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.ListAICorrectionOrganizationsActivity, mock.Anything, mock.Anything).
		Return(&temporaljobs.TenantPage{Tenants: []temporaljobs.TenantWorkItem{
			workItem("org_a"), workItem("org_b"), workItem("org_c"),
		}}, nil).
		Once()

	instants := make(map[int64]struct{})
	env.OnActivity(a.CheckOrganizationExtractionDriftActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *OrganizationDriftInput,
		) (*OrganizationDriftResult, error) {
			instants[input.Now] = struct{}{}
			switch input.OrganizationID {
			case "org_b":
				return nil, temporal.NewNonRetryableApplicationError(
					"broken",
					"Broken",
					errors.New("broken"),
				)
			case "org_c":
				return &OrganizationDriftResult{Notified: 2}, nil
			default:
				return &OrganizationDriftResult{}, nil
			}
		})

	env.ExecuteWorkflow(ExtractionAccuracyDriftWorkflow, (*ExtractionAccuracyDriftInput)(nil))

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ExtractionAccuracyDriftResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 2, result.OrganizationsProcessed)
	assert.Equal(t, 2, result.Notified)
	assert.Equal(t, []string{"org_b"}, result.FailedOrganizations)
	assert.Len(t, instants, 1, "every organization is checked as of one instant")
}

type fakeDriftChecker struct {
	notified int
	err      error
	got      *services.ExtractionDriftCheckRequest
}

func (f *fakeDriftChecker) CheckDrift(
	_ context.Context,
	req *services.ExtractionDriftCheckRequest,
) (int, error) {
	f.got = req
	return f.notified, f.err
}

func TestCheckOrganizationExtractionDriftActivity(t *testing.T) {
	t.Parallel()

	checker := &fakeDriftChecker{notified: 1}
	activities := &Activities{drift: checker, l: zap.NewNop()}
	input := &OrganizationDriftInput{TenantWorkItem: workItem("org_a"), Now: 1_790_000_000}

	result, err := activities.CheckOrganizationExtractionDriftActivity(t.Context(), input)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Notified)
	require.NotNil(t, checker.got)
	assert.Equal(t, input.TenantInfo(), checker.got.TenantInfo)
	assert.Equal(t, input.Now, checker.got.Now)

	checker.err = errors.New("database unavailable")
	_, err = activities.CheckOrganizationExtractionDriftActivity(t.Context(), input)
	require.ErrorIs(t, err, checker.err)

	unwired := &Activities{l: zap.NewNop()}
	result, err = unwired.CheckOrganizationExtractionDriftActivity(t.Context(), input)
	require.NoError(t, err)
	assert.Zero(t, result.Notified)
}
