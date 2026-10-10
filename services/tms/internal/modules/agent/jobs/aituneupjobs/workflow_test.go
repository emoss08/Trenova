package aituneupjobs

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

func TestComputeAITuneUpsWorkflow_ComputesEachOrganization(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.ListAITuneUpOrganizationsActivity, mock.Anything, mock.Anything).
		Return(&temporaljobs.TenantPage{Tenants: []temporaljobs.TenantWorkItem{
			workItem("org_a"), workItem("org_b"), workItem("org_c"),
		}}, nil).
		Once()

	env.OnActivity(a.ComputeOrganizationAITuneUpsActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *OrganizationAITuneUpsInput,
		) (*OrganizationAITuneUpsResult, error) {
			if input.OrganizationID == "org_b" {
				return nil, temporal.NewNonRetryableApplicationError("broken", "Broken", errors.New("broken"))
			}
			return &OrganizationAITuneUpsResult{Suggested: 3}, nil
		})

	env.ExecuteWorkflow(ComputeAITuneUpsWorkflow, (*ComputeAITuneUpsInput)(nil))

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ComputeAITuneUpsResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 2, result.OrganizationsProcessed)
	assert.Equal(t, 6, result.Suggested)
	assert.Equal(t, []string{"org_b"}, result.FailedOrganizations,
		"one organization failing does not stop the others")
}

func TestSchedules_MatchTheirWorkflowSignatures(t *testing.T) {
	t.Parallel()

	for _, s := range NewScheduleProvider().GetSchedules() {
		require.NoError(t, s.Validate(), s.ID)
	}
}
