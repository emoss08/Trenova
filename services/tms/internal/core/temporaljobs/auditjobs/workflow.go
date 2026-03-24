package auditjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var defaultRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
	NonRetryableErrorTypes: []string{
		temporaltype.ErrorTypeInvalidInput.String(),
		temporaltype.ErrorTypeDataIntegrity.String(),
	},
}

var batchActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	HeartbeatTimeout:    10 * time.Second,
	RetryPolicy:         defaultRetryPolicy,
}

var flushActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 1 * time.Minute,
	HeartbeatTimeout:    10 * time.Second,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    30 * time.Second,
	},
}

var dlqActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    time.Minute,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "ProcessAuditBatchWorkflow",
			Fn:          ProcessAuditBatchWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Process batch of audit entries",
		},
		{
			Name:        "ScheduledAuditFlushWorkflow",
			Fn:          ScheduledAuditFlushWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Scheduled workflow to flush audit buffer from Redis",
		},
		{
			Name:        "DLQRetryWorkflow",
			Fn:          DLQRetryWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Retry failed audit entries from dead-letter queue",
		},
	}
}

func ProcessAuditBatchWorkflow(
	ctx workflow.Context,
	payload *ProcessAuditBatchPayload,
) (*ProcessAuditBatchResult, error) {
	ctx = workflow.WithActivityOptions(ctx, batchActivityOptions)

	var a *Activities
	var result *ProcessAuditBatchResult

	err := workflow.ExecuteActivity(ctx, a.ProcessAuditBatchActivity, payload).Get(ctx, &result)
	if err != nil {
		workflow.GetLogger(ctx).Error(
			"Failed to process audit batch",
			"batchId", payload.BatchID.String(),
			"error", err,
		)
		return nil, err
	}

	workflow.GetLogger(ctx).Info(
		"Successfully processed audit batch",
		"batchId", payload.BatchID.String(),
		"processedCount", result.ProcessedCount,
	)

	return result, nil
}

func ScheduledAuditFlushWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting scheduled audit flush workflow")

	ctx = workflow.WithActivityOptions(ctx, flushActivityOptions)

	var a *Activities
	var flushResult *FlushFromRedisResult

	err := workflow.ExecuteActivity(ctx, a.FlushFromRedisActivity).Get(ctx, &flushResult)
	if err != nil {
		logger.Error("Failed to flush from Redis buffer", "error", err)
		return err
	}

	if flushResult.EntryCount == 0 {
		logger.Info("No audit entries to process")
		return nil
	}

	logger.Info("Flushed audit buffer from Redis",
		"entryCount", flushResult.EntryCount,
		"batchCount", len(flushResult.Batches),
	)

	for i, batch := range flushResult.Batches {
		if len(batch) == 0 {
			continue
		}

		var batchID pulid.ID
		err = workflow.SideEffect(ctx, func(_ workflow.Context) any {
			return pulid.MustNew("aeb_")
		}).Get(&batchID)
		if err != nil {
			return err
		}

		batchPayload := &ProcessAuditBatchPayload{
			BasePayload: temporaltype.BasePayload{
				Timestamp: workflow.Now(ctx).Unix(),
			},
			Entries: batch,
			BatchID: batchID,
		}

		if batch[0] != nil {
			batchPayload.OrganizationID = batch[0].OrganizationID
			batchPayload.BusinessUnitID = batch[0].BusinessUnitID
		}

		cwo := workflow.ChildWorkflowOptions{
			TaskQueue:           temporaltype.AuditTaskQueue,
			WorkflowID:          "process-audit-batch-" + batchID.String(),
			WorkflowRunTimeout:  5 * time.Minute,
			WorkflowTaskTimeout: 30 * time.Second,
		}

		childCtx := workflow.WithChildOptions(ctx, cwo)
		var result *ProcessAuditBatchResult

		err = workflow.ExecuteChildWorkflow(
			childCtx,
			ProcessAuditBatchWorkflow,
			batchPayload,
		).Get(childCtx, &result)

		if err != nil {
			logger.Error("Failed to process audit batch, moving to DLQ",
				"batchId", batchID.String(),
				"batchIndex", i,
				"error", err,
			)

			dlqPayload := &MoveToDLQPayload{
				Entries:      batch,
				ErrorMessage: err.Error(),
			}

			dlqErr := workflow.ExecuteActivity(ctx, a.MoveToDLQActivity, dlqPayload).Get(ctx, nil)
			if dlqErr != nil {
				logger.Error("Failed to move batch to DLQ",
					"batchId", batchID.String(),
					"error", dlqErr,
				)
			}
			continue
		}

		logger.Info("Successfully processed audit batch",
			"batchId", batchID.String(),
			"processedCount", result.ProcessedCount,
			"failedCount", result.FailedCount,
		)
	}

	return nil
}

func DLQRetryWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting DLQ retry workflow")

	ctx = workflow.WithActivityOptions(ctx, dlqActivityOptions)

	var a *Activities
	var result *DLQRetryResult

	const retryLimit = 100

	err := workflow.ExecuteActivity(ctx, a.RetryDLQEntriesActivity, retryLimit).Get(ctx, &result)
	if err != nil {
		logger.Error("DLQ retry activity failed", "error", err)
		return err
	}

	logger.Info("DLQ retry workflow completed",
		"retryCount", result.RetryCount,
		"successCount", result.SuccessCount,
		"failedCount", result.FailedCount,
		"exhaustedCount", result.ExhaustedCount,
	)

	return nil
}
