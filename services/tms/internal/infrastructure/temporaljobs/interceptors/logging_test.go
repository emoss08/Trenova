package interceptors

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type mockActivityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	result any
	err    error
}

func (m *mockActivityInbound) ExecuteActivity(
	_ context.Context,
	_ *interceptor.ExecuteActivityInput,
) (any, error) {
	return m.result, m.err
}

type mockWorkflowInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	result any
	err    error
}

func (m *mockWorkflowInbound) ExecuteWorkflow(
	_ workflow.Context,
	_ *interceptor.ExecuteWorkflowInput,
) (any, error) {
	return m.result, m.err
}

func TestLoggingInterceptor_InterceptActivity(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	li := NewLoggingInterceptor(logger, "info")

	next := &mockActivityInbound{result: "test", err: nil}
	result := li.InterceptActivity(t.Context(), next)

	require.NotNil(t, result)
	_, ok := result.(*loggingActivityInbound)
	assert.True(t, ok)
}

func TestLoggingInterceptor_InterceptActivity_DebugLevel(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	li := NewLoggingInterceptor(logger, "debug")

	next := &mockActivityInbound{result: "test", err: nil}
	result := li.InterceptActivity(t.Context(), next)

	require.NotNil(t, result)
	inbound, ok := result.(*loggingActivityInbound)
	assert.True(t, ok)
	assert.Equal(t, "debug", inbound.logLevel)
}

func TestLoggingInterceptor_InterceptWorkflow(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	li := NewLoggingInterceptor(logger, "info")

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(func(ctx workflow.Context) (any, error) {
		next := &mockWorkflowInbound{result: "test", err: nil}
		result := li.InterceptWorkflow(ctx, next)
		require.NotNil(t, result)
		_, ok := result.(*loggingWorkflowInbound)
		assert.True(t, ok)
		return nil, nil
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestLoggingInterceptor_InterceptWorkflow_DebugLevel(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	li := NewLoggingInterceptor(logger, "debug")

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(func(ctx workflow.Context) (any, error) {
		next := &mockWorkflowInbound{result: "test", err: nil}
		result := li.InterceptWorkflow(ctx, next)
		require.NotNil(t, result)
		inbound, ok := result.(*loggingWorkflowInbound)
		assert.True(t, ok)
		assert.Equal(t, "debug", inbound.logLevel)
		return nil, nil
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestLoggingActivityInbound_ExecuteActivity_Success_InfoLevel(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	next := &mockActivityInbound{result: "success_result", err: nil}

	inbound := &loggingActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		logger:                         logger,
		logLevel:                       "info",
	}

	result, err := inbound.ExecuteActivity(
		t.Context(),
		&interceptor.ExecuteActivityInput{},
	)
	require.NoError(t, err)
	assert.Equal(t, "success_result", result)
}

func TestLoggingActivityInbound_ExecuteActivity_Success_DebugLevel(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	next := &mockActivityInbound{result: "debug_result", err: nil}

	inbound := &loggingActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		logger:                         logger,
		logLevel:                       "debug",
	}

	result, err := inbound.ExecuteActivity(
		t.Context(),
		&interceptor.ExecuteActivityInput{},
	)
	require.NoError(t, err)
	assert.Equal(t, "debug_result", result)
}

func TestLoggingActivityInbound_ExecuteActivity_Error(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	expectedErr := errors.New("activity failed")
	next := &mockActivityInbound{result: nil, err: expectedErr}

	inbound := &loggingActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		logger:                         logger,
		logLevel:                       "info",
	}

	result, err := inbound.ExecuteActivity(
		t.Context(),
		&interceptor.ExecuteActivityInput{},
	)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, result)
}

func TestLoggingActivityInbound_ExecuteActivity_Error_DebugLevel(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	expectedErr := errors.New("debug activity error")
	next := &mockActivityInbound{result: nil, err: expectedErr}

	inbound := &loggingActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		logger:                         logger,
		logLevel:                       "debug",
	}

	result, err := inbound.ExecuteActivity(
		t.Context(),
		&interceptor.ExecuteActivityInput{},
	)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, result)
}

func TestLoggingWorkflowInbound_ExecuteWorkflow_Success_InfoLevel(t *testing.T) {
	t.Parallel()

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(func(ctx workflow.Context) (any, error) {
		logger := zap.NewNop()
		next := &mockWorkflowInbound{result: "workflow_result", err: nil}

		inbound := &loggingWorkflowInbound{
			WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
			logger:                         logger,
			logLevel:                       "info",
		}

		result, err := inbound.ExecuteWorkflow(ctx, &interceptor.ExecuteWorkflowInput{})
		assert.NoError(t, err)
		assert.Equal(t, "workflow_result", result)
		return result, err
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestLoggingWorkflowInbound_ExecuteWorkflow_Success_DebugLevel(t *testing.T) {
	t.Parallel()

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(func(ctx workflow.Context) (any, error) {
		logger := zap.NewNop()
		next := &mockWorkflowInbound{result: "debug_workflow", err: nil}

		inbound := &loggingWorkflowInbound{
			WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
			logger:                         logger,
			logLevel:                       "debug",
		}

		result, err := inbound.ExecuteWorkflow(ctx, &interceptor.ExecuteWorkflowInput{})
		assert.NoError(t, err)
		assert.Equal(t, "debug_workflow", result)
		return result, err
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestLoggingWorkflowInbound_ExecuteWorkflow_Error(t *testing.T) {
	t.Parallel()

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(func(ctx workflow.Context) (any, error) {
		logger := zap.NewNop()
		expectedErr := errors.New("workflow failed")
		next := &mockWorkflowInbound{result: nil, err: expectedErr}

		inbound := &loggingWorkflowInbound{
			WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
			logger:                         logger,
			logLevel:                       "info",
		}

		result, err := inbound.ExecuteWorkflow(ctx, &interceptor.ExecuteWorkflowInput{})
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
		return nil, expectedErr
	})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func TestLoggingWorkflowInbound_ExecuteWorkflow_Error_DebugLevel(t *testing.T) {
	t.Parallel()

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(func(ctx workflow.Context) (any, error) {
		logger := zap.NewNop()
		expectedErr := errors.New("debug workflow error")
		next := &mockWorkflowInbound{result: nil, err: expectedErr}

		inbound := &loggingWorkflowInbound{
			WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
			logger:                         logger,
			logLevel:                       "debug",
		}

		result, err := inbound.ExecuteWorkflow(ctx, &interceptor.ExecuteWorkflowInput{})
		assert.Error(t, err)
		assert.Nil(t, result)
		return nil, expectedErr
	})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}
