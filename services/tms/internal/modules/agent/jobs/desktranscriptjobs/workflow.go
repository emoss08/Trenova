package desktranscriptjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ExportWorkflowName  = desktranscriptports.ExportWorkflowName
	CleanupWorkflowName = "CleanupDeskTranscriptExportsWorkflow"

	exportActivityTimeout  = 2 * time.Hour
	cleanupActivityTimeout = 30 * time.Minute
	exportHeartbeat        = 2 * time.Minute
	cleanupHeartbeat       = time.Minute
	maxAttempts            = 3
)

func activityOptions(timeout, heartbeat time.Duration) workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: timeout,
		HeartbeatTimeout:    heartbeat,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    maxAttempts,
		},
	}
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        ExportWorkflowName,
			Fn:          DeskTranscriptExportWorkflow,
			TaskQueue:   temporaltype.ReportTaskQueue,
			Description: "Write an export of Desk conversations to object storage",
		},
		{
			Name:        CleanupWorkflowName,
			Fn:          CleanupDeskTranscriptExportsWorkflow,
			TaskQueue:   temporaltype.ReportTaskQueue,
			Description: "Delete Desk conversation exports past their download window",
		},
	}
}

func DeskTranscriptExportWorkflow(
	ctx workflow.Context,
	payload *desktranscriptports.ExportPayload,
) (*ExportResult, error) {
	ctx = workflow.WithActivityOptions(ctx, activityOptions(exportActivityTimeout, exportHeartbeat))

	var a *Activities
	var result *ExportResult
	if err := workflow.ExecuteActivity(ctx, a.RunDeskTranscriptExportActivity, payload).
		Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func CleanupDeskTranscriptExportsWorkflow(ctx workflow.Context) (*CleanupResult, error) {
	ctx = workflow.WithActivityOptions(
		ctx,
		activityOptions(cleanupActivityTimeout, cleanupHeartbeat),
	)

	var a *Activities
	var result *CleanupResult
	if err := workflow.ExecuteActivity(ctx, a.CleanupDeskTranscriptExportsActivity).
		Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}
