package interceptors

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

func TestMetricsInterceptor_InterceptActivity(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := metrics.NewTemporal(registry, zap.NewNop(), true)
	mi := NewMetricsInterceptor(m)

	next := &mockActivityInbound{result: "test", err: nil}
	result := mi.InterceptActivity(t.Context(), next)

	require.NotNil(t, result)
	_, ok := result.(*metricsActivityInbound)
	assert.True(t, ok)
}

func TestMetricsInterceptor_InterceptActivity_NilMetrics(t *testing.T) {
	t.Parallel()

	mi := NewMetricsInterceptor(nil)

	next := &mockActivityInbound{result: "test", err: nil}
	result := mi.InterceptActivity(t.Context(), next)

	require.NotNil(t, result)
	inbound, ok := result.(*metricsActivityInbound)
	assert.True(t, ok)
	assert.Nil(t, inbound.metrics)
}

func TestMetricsInterceptor_InterceptWorkflow(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := metrics.NewTemporal(registry, zap.NewNop(), true)
	mi := NewMetricsInterceptor(m)

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(func(ctx workflow.Context) (any, error) {
		next := &mockWorkflowInbound{result: "test", err: nil}
		result := mi.InterceptWorkflow(ctx, next)
		require.NotNil(t, result)
		_, ok := result.(*metricsWorkflowInbound)
		assert.True(t, ok)
		return nil, nil
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestMetricsActivityInbound_ExecuteActivity_Success_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := metrics.NewTemporal(registry, zap.NewNop(), true)
	next := &mockActivityInbound{result: "metrics_result", err: nil}

	inbound := &metricsActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		metrics:                        m,
	}

	result, err := inbound.ExecuteActivity(
		t.Context(),
		&interceptor.ExecuteActivityInput{},
	)
	require.NoError(t, err)
	assert.Equal(t, "metrics_result", result)
}

func TestMetricsActivityInbound_ExecuteActivity_Error_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := metrics.NewTemporal(registry, zap.NewNop(), true)
	expectedErr := errors.New("activity error")
	next := &mockActivityInbound{result: nil, err: expectedErr}

	inbound := &metricsActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		metrics:                        m,
	}

	result, err := inbound.ExecuteActivity(
		t.Context(),
		&interceptor.ExecuteActivityInput{},
	)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, result)
}

func TestMetricsActivityInbound_ExecuteActivity_NilMetrics(t *testing.T) {
	t.Parallel()

	next := &mockActivityInbound{result: "passthrough", err: nil}

	inbound := &metricsActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		metrics:                        nil,
	}

	result, err := inbound.ExecuteActivity(
		t.Context(),
		&interceptor.ExecuteActivityInput{},
	)
	require.NoError(t, err)
	assert.Equal(t, "passthrough", result)
}

func TestMetricsActivityInbound_ExecuteActivity_Disabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := metrics.NewTemporal(registry, zap.NewNop(), false)
	next := &mockActivityInbound{result: "disabled_result", err: nil}

	inbound := &metricsActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		metrics:                        m,
	}

	result, err := inbound.ExecuteActivity(
		t.Context(),
		&interceptor.ExecuteActivityInput{},
	)
	require.NoError(t, err)
	assert.Equal(t, "disabled_result", result)
}

func TestMetricsWorkflowInbound_ExecuteWorkflow_Success_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := metrics.NewTemporal(registry, zap.NewNop(), true)

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(func(ctx workflow.Context) (any, error) {
		next := &mockWorkflowInbound{result: "workflow_metrics", err: nil}

		inbound := &metricsWorkflowInbound{
			WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
			metrics:                        m,
		}

		result, err := inbound.ExecuteWorkflow(ctx, &interceptor.ExecuteWorkflowInput{})
		assert.NoError(t, err)
		assert.Equal(t, "workflow_metrics", result)
		return result, err
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestMetricsWorkflowInbound_ExecuteWorkflow_Error_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := metrics.NewTemporal(registry, zap.NewNop(), true)

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(func(ctx workflow.Context) (any, error) {
		expectedErr := errors.New("workflow error")
		next := &mockWorkflowInbound{result: nil, err: expectedErr}

		inbound := &metricsWorkflowInbound{
			WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
			metrics:                        m,
		}

		result, err := inbound.ExecuteWorkflow(ctx, &interceptor.ExecuteWorkflowInput{})
		assert.Error(t, err)
		assert.Nil(t, result)
		return nil, expectedErr
	})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func TestMetricsWorkflowInbound_ExecuteWorkflow_NilMetrics(t *testing.T) {
	t.Parallel()

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(func(ctx workflow.Context) (any, error) {
		next := &mockWorkflowInbound{result: "nil_metrics", err: nil}

		inbound := &metricsWorkflowInbound{
			WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
			metrics:                        nil,
		}

		result, err := inbound.ExecuteWorkflow(ctx, &interceptor.ExecuteWorkflowInput{})
		assert.NoError(t, err)
		assert.Equal(t, "nil_metrics", result)
		return result, err
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestMetricsWorkflowInbound_ExecuteWorkflow_Disabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := metrics.NewTemporal(registry, zap.NewNop(), false)

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(func(ctx workflow.Context) (any, error) {
		next := &mockWorkflowInbound{result: "disabled_workflow", err: nil}

		inbound := &metricsWorkflowInbound{
			WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
			metrics:                        m,
		}

		result, err := inbound.ExecuteWorkflow(ctx, &interceptor.ExecuteWorkflowInput{})
		assert.NoError(t, err)
		assert.Equal(t, "disabled_workflow", result)
		return result, err
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}
