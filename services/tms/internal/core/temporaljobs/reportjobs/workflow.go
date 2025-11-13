package reportjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "GenerateReportWorkflow",
			Fn:          GenerateReportWorkflow,
			TaskQueue:   temporaltype.ReportTaskQueue,
			Description: "Generate and deliver a data export report",
		},
	}
}

func GenerateReportWorkflow(
	ctx workflow.Context,
	payload *temporaltype.GenerateReportPayload,
) (*temporaltype.ReportResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
			MaximumInterval:    2 * time.Minute,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	so := &workflow.SessionOptions{
		CreationTimeout:  10 * time.Second,
		ExecutionTimeout: 15 * time.Minute,
	}

	sessionCtx, err := workflow.CreateSession(ctx, so)
	if err != nil {
		return nil, err
	}
	defer workflow.CompleteSession(sessionCtx)

	var a *Activities

	err = workflow.ExecuteActivity(sessionCtx, a.UpdateReportStatusActivity, payload.ReportID, "PROCESSING").
		Get(sessionCtx, nil)
	if err != nil {
		return nil, err
	}

	var queryResult temporaltype.QueryExecutionResult
	err = workflow.ExecuteActivity(sessionCtx, a.ExecuteQueryActivity, payload).
		Get(sessionCtx, &queryResult)
	if err != nil {
		var failureMsg string
		if err != nil {
			failureMsg = err.Error()
		}
		_ = workflow.ExecuteActivity(sessionCtx, a.MarkReportFailedActivity, payload.ReportID, failureMsg).
			Get(sessionCtx, nil)
		return nil, err
	}

	var result temporaltype.ReportResult
	err = workflow.ExecuteActivity(sessionCtx, a.GenerateFileActivity, payload, &queryResult).
		Get(sessionCtx, &result)
	if err != nil {
		var failureMsg string
		if err != nil {
			failureMsg = err.Error()
		}
		_ = workflow.ExecuteActivity(sessionCtx, a.MarkReportFailedActivity, payload.ReportID, failureMsg).
			Get(sessionCtx, nil)
		return nil, err
	}

	err = workflow.ExecuteActivity(sessionCtx, a.UploadToStorageActivity, &result).
		Get(sessionCtx, &result)
	if err != nil {
		var failureMsg string
		if err != nil {
			failureMsg = err.Error()
		}
		_ = workflow.ExecuteActivity(sessionCtx, a.MarkReportFailedActivity, payload.ReportID, failureMsg).
			Get(sessionCtx, nil)
		return nil, err
	}

	err = workflow.ExecuteActivity(sessionCtx, a.UpdateReportCompletedActivity, &result).
		Get(sessionCtx, nil)
	if err != nil {
		return nil, err
	}

	if payload.DeliveryMethod == "EMAIL" {
		err = workflow.ExecuteActivity(sessionCtx, a.SendReportEmailActivity, payload, &result).
			Get(sessionCtx, nil)
		if err != nil {
			return nil, err
		}
	}

	return &result, nil
}
