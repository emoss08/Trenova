package interceptors

import (
	"context"
	"reflect"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"
)

const SystemFairnessKey = "system"

func FairnessKey(ctx context.Context, args ...any) string {
	if key, ok := tenantKey(args...); ok {
		return key
	}
	if tenant, ok := dbscope.TenantFrom(ctx); ok && !tenant.OrganizationID.IsNil() {
		return tenant.OrganizationID.String()
	}
	return SystemFairnessKey
}

func tenantKey(args ...any) (string, bool) {
	if tenant, ok := dbscope.TenantOf(args...); ok && !tenant.OrganizationID.IsNil() {
		return tenant.OrganizationID.String(), true
	}
	for _, arg := range args {
		if organization, ok := organizationOf(reflect.ValueOf(arg), 0); ok {
			return organization.String(), true
		}
	}
	return "", false
}

const organizationSearchDepth = 3

var (
	pulidType          = reflect.TypeFor[pulid.ID]()
	organizationFields = [...]string{"OrganizationID", "OrgID"}
)

func organizationOf(v reflect.Value, depth int) (pulid.ID, bool) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return pulid.Nil, false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct || v.Type() == pulidType {
		return pulid.Nil, false
	}
	for _, name := range organizationFields {
		field := v.FieldByName(name)
		if field.IsValid() && field.Type() == pulidType && field.CanInterface() {
			if id, _ := field.Interface().(pulid.ID); !id.IsNil() {
				return id, true
			}
		}
	}
	if depth >= organizationSearchDepth {
		return pulid.Nil, false
	}
	for i := range v.NumField() {
		if !v.Type().Field(i).IsExported() {
			continue
		}
		if id, ok := organizationOf(v.Field(i), depth+1); ok {
			return id, true
		}
	}
	return pulid.Nil, false
}

type Fairness struct {
	interceptor.InterceptorBase
}

func NewFairness() *Fairness {
	return &Fairness{}
}

func (f *Fairness) InterceptClient(
	next interceptor.ClientOutboundInterceptor,
) interceptor.ClientOutboundInterceptor {
	return &fairnessClientOutbound{
		ClientOutboundInterceptorBase: interceptor.ClientOutboundInterceptorBase{Next: next},
	}
}

type fairnessClientOutbound struct {
	interceptor.ClientOutboundInterceptorBase
}

func (c *fairnessClientOutbound) ExecuteWorkflow(
	ctx context.Context,
	in *interceptor.ClientExecuteWorkflowInput,
) (client.WorkflowRun, error) {
	setFairness(ctx, in.Options, in.Args)
	return c.Next.ExecuteWorkflow(ctx, in)
}

func (c *fairnessClientOutbound) SignalWithStartWorkflow(
	ctx context.Context,
	in *interceptor.ClientSignalWithStartWorkflowInput,
) (client.WorkflowRun, error) {
	setFairness(ctx, in.Options, in.Args)
	return c.Next.SignalWithStartWorkflow(ctx, in)
}

func (c *fairnessClientOutbound) CreateSchedule(
	ctx context.Context,
	in *interceptor.ScheduleClientCreateInput,
) (client.ScheduleHandle, error) {
	if in.Options != nil {
		if action, ok := in.Options.Action.(*client.ScheduleWorkflowAction); ok &&
			action.Priority.FairnessKey == "" {
			action.Priority.FairnessKey = FairnessKey(ctx, action.Args...)
		}
	}
	return c.Next.CreateSchedule(ctx, in)
}

func setFairness(ctx context.Context, options *client.StartWorkflowOptions, args []any) {
	if options == nil || options.Priority.FairnessKey != "" {
		return
	}
	options.Priority.FairnessKey = FairnessKey(ctx, args...)
}

func (f *Fairness) InterceptWorkflow(
	_ workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	return &fairnessWorkflowInbound{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
	}
}

type fairnessWorkflowInbound struct {
	interceptor.WorkflowInboundInterceptorBase
}

func (w *fairnessWorkflowInbound) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	return w.Next.Init(&fairnessWorkflowOutbound{
		WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{
			Next: outbound,
		},
	})
}

type fairnessWorkflowOutbound struct {
	interceptor.WorkflowOutboundInterceptorBase
}

func (w *fairnessWorkflowOutbound) ExecuteActivity(
	ctx workflow.Context,
	activityType string,
	args ...any,
) workflow.Future {
	if key, ok := tenantKey(args...); ok {
		options := workflow.GetActivityOptions(ctx)
		if options.Priority.FairnessKey == "" {
			options.Priority.FairnessKey = key
			ctx = workflow.WithActivityOptions(ctx, options)
		}
	}
	return w.Next.ExecuteActivity(ctx, activityType, args...)
}

func (w *fairnessWorkflowOutbound) ExecuteChildWorkflow(
	ctx workflow.Context,
	childWorkflowType string,
	args ...any,
) workflow.ChildWorkflowFuture {
	if key, ok := tenantKey(args...); ok {
		options := workflow.GetChildWorkflowOptions(ctx)
		if options.Priority.FairnessKey == "" {
			options.Priority.FairnessKey = key
			ctx = workflow.WithChildOptions(ctx, options)
		}
	}
	return w.Next.ExecuteChildWorkflow(ctx, childWorkflowType, args...)
}
