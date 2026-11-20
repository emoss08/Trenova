package temporaljobs

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

const runForTenantActivity = "RunForTenantActivity"

type fanOutWorld struct {
	Entitled TenantWorkItem
	Lacking  TenantWorkItem
}

func newFanOutWorld() fanOutWorld {
	item := func() TenantWorkItem {
		return TenantWorkItem{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		}
	}
	return fanOutWorld{Entitled: item(), Lacking: item()}
}

func fanOutTestWorkflow(ctx workflow.Context, world fanOutWorld) (*TenantRunResult, error) {
	result, _, err := RunTenantFanOut(ctx, FanOutOptions{
		Label: "test",
		Need:  entitlement.For(platformcatalog.FeatureAIDesk),
		ListPage: func(workflow.Context, *TenantWorkItem) (*TenantPage, error) {
			return &TenantPage{Tenants: []TenantWorkItem{world.Entitled, world.Lacking}}, nil
		},
		RunTenant: func(wctx workflow.Context, tenant TenantWorkItem) (int, error) {
			actx := workflow.WithActivityOptions(
				wctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Minute},
			)
			return 1, workflow.ExecuteActivity(actx, runForTenantActivity, tenant).Get(actx, nil)
		},
	})
	return result, err
}

func TestRunTenantFanOutServesOnlyTheEntitledTenants(t *testing.T) {
	t.Parallel()

	world := newFanOutWorld()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(
		NewEntitledActivities(entitledOrganizations{world.Entitled.OrganizationID}),
	)
	var served []pulid.ID
	env.RegisterActivityWithOptions(func(_ context.Context, tenant TenantWorkItem) error {
		served = append(served, tenant.OrganizationID)
		return nil
	}, activity.RegisterOptions{Name: runForTenantActivity})
	env.RegisterWorkflow(fanOutTestWorkflow)

	env.ExecuteWorkflow(fanOutTestWorkflow, world)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result TenantRunResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, []pulid.ID{world.Entitled.OrganizationID}, served)
	assert.Equal(t, 2, result.TenantsScanned)
	assert.Equal(t, 1, result.TenantsProcessed)
	assert.Equal(t, 1, result.SkippedCount)
}

func keepWithoutNeedWorkflow(ctx workflow.Context, items []TenantWorkItem) error {
	_, err := KeepEntitled(ctx, entitlement.Need{}, items)
	return err
}

func TestKeepEntitledRefusesWorkThatNamesNoNeed(t *testing.T) {
	t.Parallel()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(keepWithoutNeedWorkflow)
	env.ExecuteWorkflow(keepWithoutNeedWorkflow, []TenantWorkItem{newFanOutWorld().Entitled})
	require.True(t, env.IsWorkflowCompleted())
	require.ErrorContains(t, env.GetWorkflowError(), "must name the feature")
}
