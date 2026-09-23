package briefingjobs

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

// The hour's due organizations are each written in a child of their own, as
// of the one instant the hour was decided at. One whose hour could not be
// decided, and one whose write failed, are named; the others get their
// morning.
func TestDailyBriefingWorkflow_WritesEachDueOrganizationInItsOwnChild(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})
	env.RegisterWorkflow(WriteOrganizationBriefingWorkflow)

	var a *Activities
	var decidedAt int64
	env.OnActivity(a.ListDueOrganizationsActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *DueOrganizationsInput) (*DueOrganizations, error) {
			decidedAt = input.Now

			return &DueOrganizations{
				Due:    []temporaljobs.TenantWorkItem{workItem("org_a"), workItem("org_b")},
				Failed: []string{"org_c"},
			}, nil
		}).
		Once()

	written := make(map[pulid.ID]int64)
	env.OnActivity(a.WriteOrganizationBriefingActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *OrganizationBriefingInput) (*OrganizationBriefingResult, error) {
			written[input.OrganizationID] = input.Now
			if input.OrganizationID == "org_b" {
				return nil, temporal.NewNonRetryableApplicationError(
					"unreadable",
					"Broken",
					errors.New("x"),
				)
			}

			return &OrganizationBriefingResult{Written: 5, Narrated: 4}, nil
		})

	env.ExecuteWorkflow(DailyBriefingWorkflow)

	require.NoError(t, env.GetWorkflowError())
	var result DailyBriefingResult
	require.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, 2, result.OrganizationsDue)
	assert.Equal(t, 5, result.BriefingsWritten)
	assert.Equal(t, 4, result.BriefingsNarrated)
	assert.ElementsMatch(t, []string{"org_b", "org_c"}, result.FailedOrganizations)
	require.Len(t, written, 2)
	for _, now := range written {
		assert.Equal(t, decidedAt, now, "every page describes the instant the hour was decided at")
	}
}

// An hour nobody is due in writes nothing and starts nothing.
func TestDailyBriefingWorkflow_AnHourWithNobodyDue(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})
	env.RegisterWorkflow(WriteOrganizationBriefingWorkflow)

	var a *Activities
	env.OnActivity(a.ListDueOrganizationsActivity, mock.Anything, mock.Anything).
		Return(&DueOrganizations{Due: []temporaljobs.TenantWorkItem{}}, nil).
		Once()

	env.ExecuteWorkflow(DailyBriefingWorkflow)

	require.NoError(t, env.GetWorkflowError())
	var result DailyBriefingResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Zero(t, result.OrganizationsDue)
	assert.Empty(t, result.FailedOrganizations)
}
