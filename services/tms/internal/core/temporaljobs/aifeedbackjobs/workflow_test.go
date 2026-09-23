package aifeedbackjobs

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
)

func workItem(id string) temporaljobs.TenantWorkItem {
	return temporaljobs.TenantWorkItem{
		OrganizationID: pulid.ID(id),
		BusinessUnitID: pulid.ID("bu_" + id),
	}
}

func TestAIFeedbackMaintenanceWorkflow_MaintainsEachOrganization(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.ListAIFeedbackOrganizationsActivity, mock.Anything, mock.Anything).
		Return(&temporaljobs.TenantPage{Tenants: []temporaljobs.TenantWorkItem{
			workItem("org_a"), workItem("org_b"),
		}}, nil).
		Once()

	instants := make(map[int64]struct{})
	env.OnActivity(a.MaintainOrganizationAIFeedbackActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *OrganizationAIFeedbackInput,
		) (*OrganizationAIFeedbackResult, error) {
			instants[input.Now] = struct{}{}
			if input.OrganizationID == "org_b" {
				return nil, temporal.NewNonRetryableApplicationError(
					"broken",
					"Broken",
					errors.New("broken"),
				)
			}

			return &OrganizationAIFeedbackResult{Purged: 4, Suggested: 2, Covered: 1}, nil
		})

	env.ExecuteWorkflow(AIFeedbackMaintenanceWorkflow, (*AIFeedbackMaintenanceInput)(nil))

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result AIFeedbackMaintenanceResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 1, result.OrganizationsProcessed)
	assert.Equal(t, int64(4), result.Purged)
	assert.Equal(t, 2, result.Suggested)
	assert.Equal(t, 1, result.Covered)
	assert.Equal(t, []string{"org_b"}, result.FailedOrganizations)
	assert.Len(t, instants, 1, "every organization is maintained as of one instant")
}

func TestSchedules_MatchTheirWorkflowSignatures(t *testing.T) {
	t.Parallel()

	for _, s := range NewScheduleProvider().GetSchedules() {
		require.NoError(t, s.Validate(), s.ID)
	}
}
