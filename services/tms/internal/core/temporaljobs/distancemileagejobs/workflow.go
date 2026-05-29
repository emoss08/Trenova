package distancemileagejobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var defaultRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var activityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	HeartbeatTimeout:    10 * time.Second,
	RetryPolicy:         defaultRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "ScheduledStoredMileageFlushWorkflow",
			Fn:          ScheduledStoredMileageFlushWorkflow,
			TaskQueue:   temporaltype.DistanceMileageTaskQueue,
			Description: "Flush stored mileage candidates from Redis into PostgreSQL",
		},
	}
}

func ScheduledStoredMileageFlushWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	var a *Activities
	var flushResult *FlushStoredMileageBufferResult
	if err := workflow.ExecuteActivity(ctx, a.FlushStoredMileageBufferActivity).Get(ctx, &flushResult); err != nil {
		logger.Error("failed to flush stored mileage buffer", "error", err)
		return err
	}
	if flushResult == nil || flushResult.RecordCount == 0 {
		logger.Info("no stored mileage candidates to flush")
		return nil
	}
	for _, batch := range flushResult.Batches {
		if len(batch) == 0 {
			continue
		}
		payload := &UpsertStoredMileageBatchPayload{Records: batch}
		var result *UpsertStoredMileageBatchResult
		if err := workflow.ExecuteActivity(ctx, a.UpsertStoredMileageBatchActivity, payload).Get(ctx, &result); err != nil {
			logger.Error("failed to upsert stored mileage batch", "error", err)
			return err
		}
	}
	return nil
}
