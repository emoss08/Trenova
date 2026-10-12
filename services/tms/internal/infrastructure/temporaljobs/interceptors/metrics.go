package interceptors

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"
)

type MetricsInterceptor struct {
	interceptor.InterceptorBase
	metrics *metrics.Temporal
}

func NewMetricsInterceptor(m *metrics.Temporal) *MetricsInterceptor {
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
	metrics *metrics.Temporal
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

	a.metrics.RecordActivityExecution("activity", "default", duration, err)

	return result, err
}

type metricsWorkflowInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	metrics *metrics.Temporal
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

	w.metrics.RecordWorkflowExecution("workflow", duration, err)

	return result, err
}
