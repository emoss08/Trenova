package interceptors

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func TestFairnessKey_ComesFromTheArgumentsThenTheContextThenTheSystem(t *testing.T) {
	t.Parallel()

	fromArgs := pulid.MustNew("org_")
	fromCtx := pulid.MustNew("org_")
	tenantCtx := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: fromCtx,
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	assert.Equal(t, fromArgs.String(), FairnessKey(tenantCtx, &pagination.TenantInfo{
		OrgID: fromArgs,
		BuID:  pulid.MustNew("bu_"),
	}), "the work's own tenant wins over the context that starts it")
	assert.Equal(t, fromCtx.String(), FairnessKey(tenantCtx, "no tenant here"))
	assert.Equal(t, SystemFairnessKey, FairnessKey(t.Context()))

	type actor struct{ OrganizationID pulid.ID }
	type payload struct{ Actor *actor }
	assert.Equal(t, fromArgs.String(), FairnessKey(t.Context(), &payload{
		Actor: &actor{OrganizationID: fromArgs},
	}), "an organization named without its business unit still keys the work")
}

type startCapture struct {
	interceptor.ClientOutboundInterceptorBase
	options *client.StartWorkflowOptions
}

func (c *startCapture) ExecuteWorkflow(
	_ context.Context,
	in *interceptor.ClientExecuteWorkflowInput,
) (client.WorkflowRun, error) {
	c.options = in.Options
	return nil, nil
}

func (c *startCapture) SignalWithStartWorkflow(
	_ context.Context,
	in *interceptor.ClientSignalWithStartWorkflowInput,
) (client.WorkflowRun, error) {
	c.options = in.Options
	return nil, nil
}

func TestFairness_SetsTheKeyOnEveryStartTheProcessMakes(t *testing.T) {
	t.Parallel()

	organization := pulid.MustNew("org_")
	args := []any{&pagination.TenantInfo{OrgID: organization, BuID: pulid.MustNew("bu_")}}

	capture := &startCapture{}
	outbound := NewFairness().InterceptClient(capture)
	_, err := outbound.ExecuteWorkflow(t.Context(), &interceptor.ClientExecuteWorkflowInput{
		Options: &client.StartWorkflowOptions{
			Priority: temporal.Priority{PriorityKey: 2},
		},
		Args: args,
	})
	require.NoError(t, err)
	assert.Equal(t, organization.String(), capture.options.Priority.FairnessKey)
	assert.Equal(t, 2, capture.options.Priority.PriorityKey, "a priority set by hand is kept")

	_, err = outbound.SignalWithStartWorkflow(
		t.Context(),
		&interceptor.ClientSignalWithStartWorkflowInput{
			Options: &client.StartWorkflowOptions{},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, SystemFairnessKey, capture.options.Priority.FairnessKey,
		"a start with no tenant uses the system key")

	_, err = outbound.ExecuteWorkflow(t.Context(), &interceptor.ClientExecuteWorkflowInput{
		Options: &client.StartWorkflowOptions{Priority: temporal.Priority{FairnessKey: "own"}},
		Args:    args,
	})
	require.NoError(t, err)
	assert.Equal(t, "own", capture.options.Priority.FairnessKey, "a key set by the start is kept")
}

type priorityCapture struct {
	interceptor.InterceptorBase
	mu       sync.Mutex
	activity []string
	child    []string
}

func (p *priorityCapture) InterceptWorkflow(
	_ workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	return &priorityCaptureInbound{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
		root:                           p,
	}
}

type priorityCaptureInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	root *priorityCapture
}

func (p *priorityCaptureInbound) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	return p.Next.Init(&priorityCaptureOutbound{
		WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{
			Next: outbound,
		},
		root: p.root,
	})
}

type priorityCaptureOutbound struct {
	interceptor.WorkflowOutboundInterceptorBase
	root *priorityCapture
}

func (p *priorityCaptureOutbound) ExecuteActivity(
	ctx workflow.Context,
	activityType string,
	args ...any,
) workflow.Future {
	p.root.mu.Lock()
	p.root.activity = append(p.root.activity, workflow.GetActivityOptions(ctx).Priority.FairnessKey)
	p.root.mu.Unlock()
	return p.Next.ExecuteActivity(ctx, activityType, args...)
}

func (p *priorityCaptureOutbound) ExecuteChildWorkflow(
	ctx workflow.Context,
	childWorkflowType string,
	args ...any,
) workflow.ChildWorkflowFuture {
	p.root.mu.Lock()
	p.root.child = append(p.root.child, workflow.GetChildWorkflowOptions(ctx).Priority.FairnessKey)
	p.root.mu.Unlock()
	return p.Next.ExecuteChildWorkflow(ctx, childWorkflowType, args...)
}

func fanOutWorkflow(ctx workflow.Context, tenants []tenantPayload) error {
	actx := workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{StartToCloseTimeout: time.Minute},
	)
	for _, tenant := range tenants {
		if err := workflow.ExecuteActivity(actx, serveTenantActivity, tenant).Get(actx, nil); err != nil {
			return err
		}
		cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{})
		if err := workflow.ExecuteChildWorkflow(cctx, childWorkflow, tenant).Get(cctx, nil); err != nil {
			return err
		}
	}
	return workflow.ExecuteActivity(actx, serveTenantActivity, tenantPayload{}).Get(actx, nil)
}

func childWorkflow(workflow.Context, tenantPayload) error { return nil }

func TestFairness_KeysEveryChildWorkflowAndActivityByItsTenant(t *testing.T) {
	t.Parallel()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	capture := &priorityCapture{}
	env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{
		capture,
		NewFairness(),
	}})
	env.RegisterActivityWithOptions(func(context.Context, tenantPayload) error { return nil },
		activity.RegisterOptions{Name: serveTenantActivity})
	env.RegisterWorkflow(fanOutWorkflow)
	env.RegisterWorkflow(childWorkflow)

	first, second := newTenantPayload(), newTenantPayload()
	env.ExecuteWorkflow(fanOutWorkflow, []tenantPayload{first, second})
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	assert.Equal(t, []string{
		first.OrganizationID.String(),
		second.OrganizationID.String(),
		"",
	}, capture.activity, "an activity with no tenant inherits its workflow's key")
	assert.Equal(t, []string{
		first.OrganizationID.String(),
		second.OrganizationID.String(),
	}, capture.child)
}
