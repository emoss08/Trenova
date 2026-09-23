package insightjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func workItem(id string) temporaljobs.TenantWorkItem {
	return temporaljobs.TenantWorkItem{
		OrganizationID: pulid.ID(id),
		BusinessUnitID: pulid.ID("bu_" + id),
	}
}

// Every organization is refreshed in a child of its own, as of one instant,
// and one organization's failure is that organization's alone.
func TestInsightRefreshWorkflow_RefreshesEachOrganizationInItsOwnChild(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})
	env.RegisterWorkflow(RefreshOrganizationInsightsWorkflow)

	var a *Activities
	env.OnActivity(a.ListOrganizationsActivity, mock.Anything, mock.Anything).
		Return(&temporaljobs.TenantPage{Tenants: []temporaljobs.TenantWorkItem{
			workItem("org_a"), workItem("org_b"), workItem("org_c"),
		}}, nil).
		Once()

	instants := make(map[int64]struct{})
	env.OnActivity(a.RefreshOrganizationInsightsActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *OrganizationInsightsInput) (*OrganizationInsightsResult, error) {
			instants[input.Now] = struct{}{}
			switch input.OrganizationID {
			case "org_b":
				return nil, temporal.NewNonRetryableApplicationError(
					"corrupt table",
					"Broken",
					errors.New("corrupt"),
				)
			case "org_c":
				return &OrganizationInsightsResult{
					Created:  2,
					Narrated: 1,
					Failed:   []string{"late_loads"},
				}, nil
			default:
				return &OrganizationInsightsResult{
					Created:  1,
					Resolved: 1,
					Failed:   []string{"late_loads"},
				}, nil
			}
		})

	env.ExecuteWorkflow(InsightRefreshWorkflow, (*InsightRefreshInput)(nil))

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result InsightRefreshResult
	require.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, 2, result.OrganizationsProcessed)
	assert.Equal(t, 3, result.InsightsCreated)
	assert.Equal(t, 1, result.InsightsResolved)
	assert.Equal(t, 1, result.InsightsNarrated)
	assert.Equal(t, []string{"org_b"}, result.FailedOrganizations)
	assert.Equal(t, []string{"late_loads"}, result.FailedDetectors,
		"a detector broken everywhere is named once")
	assert.Len(t, instants, 1, "every organization describes the same instant")
}

// A sweep that pages carries its instant into the next run, so the tenants on
// either side of continue-as-new describe the same window.
func TestInsightRefreshWorkflow_CarriesItsInstantAcrossPages(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})
	env.RegisterWorkflow(RefreshOrganizationInsightsWorkflow)

	var a *Activities
	pages := 0
	env.OnActivity(a.ListOrganizationsActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *ListOrganizationsInput) (*temporaljobs.TenantPage, error) {
			pages++
			return &temporaljobs.TenantPage{
				Tenants: []temporaljobs.TenantWorkItem{workItem(pulid.MustNew("org_").String())},
				HasMore: true,
			}, nil
		})
	env.OnActivity(a.RefreshOrganizationInsightsActivity, mock.Anything, mock.Anything).
		Return(&OrganizationInsightsResult{}, nil)

	env.ExecuteWorkflow(InsightRefreshWorkflow, &InsightRefreshInput{Now: 1_700_000_000})

	var continued *workflow.ContinueAsNewError
	require.ErrorAs(t, env.GetWorkflowError(), &continued,
		"a sweep with more pages continues as new")
	assert.Positive(t, pages)

	var next InsightRefreshInput
	require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(continued.Input, &next))
	assert.Equal(t, int64(1_700_000_000), next.Now)
	require.NotNil(t, next.After, "the next run starts after the last organization refreshed")
}
