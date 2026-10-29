package interceptors

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/platformmetrics"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"
)

type MetricsInterceptor struct {
	interceptor.InterceptorBase
	metrics *platformmetrics.Temporal
}

func NewMetricsInterceptor(m *platformmetrics.Temporal) *MetricsInterceptor {
	return &MetricsInterceptor{
		metrics: m,
	}
}

func (m *MetricsInterceptor) InterceptActivity(
	_ context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &metricsActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		metrics:                        m.metrics,
	}
}

func (m *MetricsInterceptor) InterceptWorkflow(
	_ workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	return &metricsWorkflowInbound{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
		metrics:                        m.metrics,
	}
}

type metricsActivityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	metrics *platformmetrics.Temporal
}

func (a *metricsActivityInbound) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (any, error) {
	if a.metrics == nil || !a.metrics.IsEnabled() {
		return a.Next.ExecuteActivity(ctx, in)
	}

	a.metrics.IncrementActiveActivities()
	defer a.metrics.DecrementActiveActivities()

	start := time.Now()
	result, err := a.Next.ExecuteActivity(ctx, in)
	duration := time.Since(start).Seconds()

	activityType, taskQueue := activityLabels(ctx)
	a.metrics.RecordActivityExecution(activityType, taskQueue, duration, err)

	return result, err
}

type metricsWorkflowInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	metrics *platformmetrics.Temporal
}

func (w *metricsWorkflowInbound) ExecuteWorkflow(
	ctx workflow.Context,
	in *interceptor.ExecuteWorkflowInput,
) (any, error) {
	if w.metrics == nil || !w.metrics.IsEnabled() {
		return w.Next.ExecuteWorkflow(ctx, in)
	}

	w.metrics.IncrementActiveWorkflows()
	defer w.metrics.DecrementActiveWorkflows()

	start := time.Now()
	result, err := w.Next.ExecuteWorkflow(ctx, in)
	duration := time.Since(start).Seconds()

	w.metrics.RecordWorkflowExecution(workflow.GetInfo(ctx).WorkflowType.Name, duration, err)

	return result, err
}

// activityLabels names the activity and its task queue for the metrics. Outside
// a real activity context (unit tests) there is no info, so the old constant
// labels stand in.
func activityLabels(ctx context.Context) (activityType, taskQueue string) {
	if !activity.IsActivity(ctx) {
		return "activity", "default"
	}
	info := activity.GetInfo(ctx)
	return info.ActivityType.Name, info.TaskQueue
}
