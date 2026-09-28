package capturejobs

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentuploadjobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const filingFailedMessage = "The document could not be stored. Try filing it again."

// processOptions give a large stack room to render, with a heartbeat short
// enough that a lost worker is replaced in minutes rather than at the
// timeout.
var processOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    2 * time.Minute,
		MaximumAttempts:    5,
	},
}

// recordOptions are for single writes that must land: the record of a filed
// document is what stops it being filed twice.
var recordOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    10,
	},
}

var fileOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval: 5 * time.Second,
		MaximumAttempts: 3,
	},
}

var maintenanceOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval: 30 * time.Second,
		MaximumAttempts: 3,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        temporaltype.ProcessCaptureBatchWorkflowName,
			Fn:          ProcessCaptureBatchWorkflow,
			TaskQueue:   temporaltype.CaptureTaskQueue,
			Description: "Reads a captured batch, divides it into documents and files what needs no person.",
		},
		{
			Name:        temporaltype.FileCaptureItemWorkflowName,
			Fn:          FileCaptureItemWorkflow,
			TaskQueue:   temporaltype.CaptureTaskQueue,
			Description: "Finalizes a captured document's upload and records the document it became.",
		},
		{
			Name:        temporaltype.CaptureMaintenanceWorkflowName,
			Fn:          CaptureMaintenanceWorkflow,
			TaskQueue:   temporaltype.CaptureTaskQueue,
			Description: "Expires stale capture requests and pairings, restarts lost runs and enforces retention.",
		},
	}
}

// ProcessCaptureBatchWorkflow reads a batch and files the items that need no
// person. An item whose automatic filing fails stays in intake for a person,
// so a failed filing never fails the batch.
func ProcessCaptureBatchWorkflow(
	ctx workflow.Context,
	payload *ProcessBatchPayload,
) (*ProcessResult, error) {
	var a *Activities

	processCtx := workflow.WithActivityOptions(ctx, processOptions)
	result := new(ProcessResult)
	if err := workflow.ExecuteActivity(processCtx, a.ProcessCaptureBatchActivity, payload).
		Get(processCtx, result); err != nil {
		return nil, err
	}

	fileCtx := workflow.WithActivityOptions(ctx, fileOptions)
	for _, itemID := range result.AutoFile {
		err := workflow.ExecuteActivity(fileCtx, a.AutoFileCaptureItemActivity, &AutoFilePayload{
			ProcessBatchPayload: *payload,
			ItemID:              itemID,
		}).Get(fileCtx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Error("an automatic capture filing failed",
				"itemId", itemID.String(), "error", err)
		}
	}

	return result, nil
}

var errNoDocument = errors.New("upload finalization produced no document")

// FileCaptureItemWorkflow finalizes a staged filing and records the result.
// The finalize run is the upload pipeline's own workflow, run as a child so
// its document comes back here instead of being looked for later.
func FileCaptureItemWorkflow(ctx workflow.Context, payload *FileItemPayload) error {
	var a *Activities

	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: "document-upload-finalize-" + payload.SessionID.String(),
		TaskQueue:  temporaltype.UploadTaskQueue,
	})

	finalized := new(documentuploadjobs.FinalizeUploadResult)
	err := workflow.ExecuteChildWorkflow(
		childCtx,
		documentuploadjobs.FinalizeDocumentUploadWorkflow,
		&documentuploadjobs.FinalizeUploadPayload{
			BasePayload:   payload.BasePayload,
			SessionID:     payload.SessionID,
			PrincipalType: services.PrincipalTypeUser,
			PrincipalID:   payload.UserID,
		},
	).Get(ctx, finalized)
	if err == nil && (finalized.DocumentID == nil || finalized.DocumentID.IsNil()) {
		err = errNoDocument
	}

	recordCtx := workflow.WithActivityOptions(ctx, recordOptions)
	if err != nil {
		workflow.GetLogger(ctx).Error("a capture filing could not be finalized",
			"itemId", payload.ItemID.String(), "error", err)

		return workflow.ExecuteActivity(recordCtx, a.RecordCaptureFilingFailedActivity, &RecordFailedPayload{
			FileItemPayload: *payload,
			Message:         filingFailedMessage,
		}).
			Get(recordCtx, nil)
	}

	return workflow.ExecuteActivity(recordCtx, a.RecordCaptureFiledActivity, &RecordFiledPayload{
		FileItemPayload: *payload,
		DocumentID:      *finalized.DocumentID,
	}).Get(recordCtx, nil)
}

// CaptureMaintenanceWorkflow runs the capture housekeeping sweep.
func CaptureMaintenanceWorkflow(ctx workflow.Context) (*MaintenanceResult, error) {
	var a *Activities

	ctx = workflow.WithActivityOptions(ctx, maintenanceOptions)
	result := new(MaintenanceResult)
	if err := workflow.ExecuteActivity(ctx, a.CaptureMaintenanceActivity).
		Get(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}
