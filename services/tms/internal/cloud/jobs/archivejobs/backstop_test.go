package archivejobs

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/jobs/archivejobs/archivejobscontract"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/interceptors"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/workfeature"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type suspendedGate struct{ asked atomic.Int32 }

func (g *suspendedGate) Enabled(ctx context.Context, _ platformcatalog.FeatureKey) (bool, error) {
	if _, ok := dbscope.TenantFrom(ctx); ok {
		g.asked.Add(1)
	}
	return false, nil
}

func (g *suspendedGate) EnabledAnywhere(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, nil
}

func TestAFinalArchiveRunsForASuspendedTenant(t *testing.T) {
	t.Parallel()

	gate := &suspendedGate{}
	var suite testsuite.WorkflowTestSuite
	suite.SetContextPropagators([]workflow.ContextPropagator{workfeature.NewPropagator()})
	env := suite.NewTestWorkflowEnvironment()
	env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{
		interceptors.NewTenantScopeInterceptor(),
		interceptors.NewEntitlementInterceptor(gate, nil, zap.NewNop()),
	}})
	payload := &archivejobscontract.CommandPayload{
		CommandID:      "7d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10",
		InstanceID:     "5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10",
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	var scoped pulid.ID
	env.RegisterActivityWithOptions(
		func(ctx context.Context, _ *archivejobscontract.CommandPayload) (*Outcome, error) {
			tenant, _ := dbscope.TenantFrom(ctx)
			scoped = tenant.OrganizationID
			return &Outcome{SHA256: "done"}, nil
		},
		activity.RegisterOptions{Name: "RunArchiveActivity"},
	)

	env.ExecuteWorkflow(CloudTenantArchiveWorkflow, payload)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, payload.OrganizationID, scoped, "the archive runs in the tenant's scope")
	assert.Zero(
		t,
		gate.asked.Load(),
		"a commanded archive carries no schedule feature, so a suspended tenant is archived",
	)
}
