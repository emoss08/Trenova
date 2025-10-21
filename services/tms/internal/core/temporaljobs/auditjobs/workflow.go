package auditjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

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
			Description: "Scheduled workflow to flush audit buffer",
		},
	}
}

func ProcessAuditBatchWorkflow(
	ctx workflow.Context,
	payload *ProcessAuditBatchPayload,
) (*ProcessAuditBatchResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		HeartbeatTimeout:    10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
			MaximumInterval:    30 * time.Second,
			NonRetryableErrorTypes: []string{
				temporaltype.ErrorTypeInvalidInput.String(),
				temporaltype.ErrorTypeDataIntegrity.String(),
			},
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

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

func ScheduledAuditFlushWorkflow(
	ctx workflow.Context,
) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting scheduled audit flush workflow")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Minute,
		HeartbeatTimeout:    10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
			MaximumInterval:    30 * time.Second,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	var a *Activities
	var entries []*audit.Entry

	err := workflow.ExecuteActivity(ctx, a.FlushAuditBufferActivity).Get(ctx, &entries)
	if err != nil {
		logger.Error("Failed to flush audit buffer", "error", err)
		return err
	}

	if len(entries) == 0 {
		logger.Info("No audit entries to process")
		return nil
	}

	logger.Info("Flushed audit buffer", "entryCount", len(entries))

	batchID := pulid.MustNew("aeb_")
	batchPayload := &ProcessAuditBatchPayload{
		BasePayload: temporaltype.BasePayload{
			Timestamp: time.Now().Unix(),
		},
		Entries: entries,
		BatchID: batchID,
	}

	if len(entries) > 0 && entries[0] != nil {
		batchPayload.OrganizationID = entries[0].OrganizationID
		batchPayload.BusinessUnitID = entries[0].BusinessUnitID
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
		logger.Error(
			"Failed to process audit batch",
			"batchId", batchID.String(),
			"error", err,
		)
		return err
	}

	logger.Info(
		"Successfully processed audit batch",
		"batchId", batchID.String(),
		"processedCount", result.ProcessedCount,
		"failedCount", result.FailedCount,
	)

	return nil
}
