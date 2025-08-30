package temporalutils

import (
	"time"

	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// BuildActivityOptions creates activity options from configuration
func BuildActivityOptions(cfg config.TemporalWorkflowConfig) workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(cfg.ActivityTimeoutSeconds) * time.Second,
		HeartbeatTimeout:    time.Duration(cfg.ActivityHeartbeatSeconds) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Duration(
				cfg.ActivityRetryInitialIntervalSeconds,
			) * time.Second,
			BackoffCoefficient: cfg.ActivityRetryBackoffCoefficient,
			MaximumAttempts:    int32(cfg.ActivityRetryMaxAttempts),
			MaximumInterval:    time.Duration(cfg.ActivityRetryMaxIntervalSeconds) * time.Second,
			NonRetryableErrorTypes: []string{
				string(temporaltype.ErrorTypeInvalidInput),
				string(temporaltype.ErrorTypePermissionDenied),
				string(temporaltype.ErrorTypeResourceNotFound),
				string(temporaltype.ErrorTypeDataIntegrity),
			},
		},
	}
}

// BuildWorkerOptions creates worker options from configuration
func BuildWorkerOptions(cfg config.TemporalWorkerConfig) worker.Options {
	return worker.Options{
		MaxConcurrentActivityExecutionSize:      cfg.MaxConcurrentActivity,
		MaxConcurrentWorkflowTaskExecutionSize:  cfg.MaxConcurrentWorkflow,
		MaxConcurrentLocalActivityExecutionSize: cfg.MaxConcurrentLocalActivity,
		WorkerActivitiesPerSecond:               cfg.WorkerActivitiesPerSecond,
		TaskQueueActivitiesPerSecond:            cfg.TaskQueueActivitiesPerSecond,
		EnableSessionWorker:                     cfg.EnableSessionWorker,
		StickyScheduleToStartTimeout: time.Duration(
			cfg.StickyScheduleToStartTimeoutSeconds,
		) * time.Second,
	}
}

// ApplyActivityOptions applies activity options to a workflow context
func ApplyActivityOptions(
	ctx workflow.Context,
	cfg config.TemporalWorkflowConfig,
) workflow.Context {
	return workflow.WithActivityOptions(ctx, BuildActivityOptions(cfg))
}

// CreateSessionOptions creates session options with standard configuration
func CreateSessionOptions(timeoutSeconds int) *workflow.SessionOptions {
	timeout := time.Duration(timeoutSeconds) * time.Second
	return &workflow.SessionOptions{
		CreationTimeout:  timeout,
		ExecutionTimeout: timeout,
	}
}

// CreateSessionWithConfig creates a session if needed based on configuration
func CreateSessionWithConfig(
	ctx workflow.Context,
	enableSessions bool,
	timeoutSeconds int,
) (workflow.Context, error) {
	if !enableSessions {
		return ctx, nil
	}

	return workflow.CreateSession(ctx, CreateSessionOptions(timeoutSeconds))
}
