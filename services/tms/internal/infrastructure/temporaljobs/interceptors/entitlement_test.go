package interceptors

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/workfeature"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const serveTenantActivity = "ServeTenantActivity"

type tenantPayload struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

type planGate struct {
	entitled map[pulid.ID]bool
}

func (g planGate) Enabled(ctx context.Context, _ platformcatalog.FeatureKey) (bool, error) {
	tenant, ok := dbscope.TenantFrom(ctx)
	if !ok {
		return false, errors.New("no tenant")
	}
	return g.entitled[tenant.OrganizationID], nil
}

func (g planGate) EnabledAnywhere(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return true, nil
}

type countingRefusals struct{ count atomic.Int32 }

func (c *countingRefusals) RecordEntitlementRefusal(string, platformcatalog.FeatureKey) {
	c.count.Add(1)
}

func serveWorkflow(ctx workflow.Context, payload tenantPayload) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})
	return workflow.ExecuteActivity(ctx, serveTenantActivity, payload).Get(ctx, nil)
}

type backstopRun struct {
	err      error
	served   int32
	refusals int32
}

func runBackstop(
	t *testing.T,
	feature platformcatalog.FeatureKey,
	gate planGate,
	payload tenantPayload,
) backstopRun {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	suite.SetContextPropagators([]workflow.ContextPropagator{workfeature.NewPropagator()})
	if feature != "" {
		header := &commonpb.Header{Fields: map[string]*commonpb.Payload{}}
		ctx := workfeature.Starting(t.Context(), feature)
		require.NoError(t, workfeature.NewPropagator().Inject(ctx, headerFields{header: header}))
		suite.SetHeader(header)
	}
	env := suite.NewTestWorkflowEnvironment()
	refusals := &countingRefusals{}
	env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{
		NewTenantScopeInterceptor(),
		NewEntitlementInterceptor(gate, refusals, zap.NewNop()),
	}})
	var served atomic.Int32
	env.RegisterActivityWithOptions(func(context.Context, tenantPayload) error {
		served.Add(1)
		return nil
	}, activity.RegisterOptions{Name: serveTenantActivity})
	env.RegisterWorkflow(serveWorkflow)

	env.ExecuteWorkflow(serveWorkflow, payload)
	require.True(t, env.IsWorkflowCompleted())

	return backstopRun{
		err:      env.GetWorkflowError(),
		served:   served.Load(),
		refusals: refusals.count.Load(),
	}
}

type headerFields struct{ header *commonpb.Header }

func (h headerFields) Set(key string, value *commonpb.Payload) { h.header.Fields[key] = value }

func newTenantPayload() tenantPayload {
	return tenantPayload{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func TestEntitlementInterceptor_RefusesWorkForATenantOutsideTheFeature(t *testing.T) {
	t.Parallel()

	lacking := newTenantPayload()
	run := runBackstop(t, platformcatalog.FeatureAIDesk, planGate{}, lacking)

	require.Error(t, run.err)
	var refused *temporal.ApplicationError
	require.ErrorAs(t, run.err, &refused)
	assert.Equal(t, NotEntitledErrorType, refused.Type())
	assert.True(t, refused.NonRetryable(), "a refusal is never retried")
	assert.Zero(t, run.served, "the activity's work never ran")
	assert.Equal(t, int32(1), run.refusals)
}

func TestEntitlementInterceptor_ServesAnEntitledTenant(t *testing.T) {
	t.Parallel()

	entitled := newTenantPayload()
	run := runBackstop(t, platformcatalog.FeatureAIDesk,
		planGate{entitled: map[pulid.ID]bool{entitled.OrganizationID: true}}, entitled)

	require.NoError(t, run.err)
	assert.Equal(t, int32(1), run.served)
	assert.Zero(t, run.refusals)
}

func TestEntitlementInterceptor_LeavesUnscheduledAndTenantlessWorkAlone(t *testing.T) {
	t.Parallel()

	run := runBackstop(t, "", planGate{}, newTenantPayload())
	require.NoError(t, run.err)
	assert.Equal(t, int32(1), run.served)

	run = runBackstop(t, platformcatalog.FeatureAIDesk, planGate{}, tenantPayload{})
	require.NoError(t, run.err)
	assert.Equal(t, int32(1), run.served)
	assert.Zero(t, run.refusals)
}
