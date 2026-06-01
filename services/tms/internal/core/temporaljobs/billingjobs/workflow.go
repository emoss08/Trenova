package billingjobs

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs/documentuploadjobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var autoPostInvoiceRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    5,
	MaximumInterval:    30 * time.Second,
}

var autoPostInvoiceActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         autoPostInvoiceRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        AutoPostInvoiceWorkflowName,
			Fn:          AutoPostInvoiceWorkflow,
			TaskQueue:   temporaltype.TaskQueueBilling.String(),
			Description: "Automatically post a draft invoice after billing approval",
		},
		{
			Name:        SendInvoiceEmailWorkflowName,
			Fn:          SendInvoiceEmailWorkflow,
			TaskQueue:   temporaltype.TaskQueueBilling.String(),
			Description: "Send a customer invoice email and supporting documents",
		},
		{
			Name:        GenerateInvoicePDFWorkflowName,
			Fn:          GenerateInvoicePDFWorkflow,
			TaskQueue:   temporaltype.TaskQueueBilling.String(),
			Description: "Generate an invoice PDF through the document upload lifecycle",
		},
	}
}

func AutoPostInvoiceWorkflow(
	ctx workflow.Context,
	payload *AutoPostInvoicePayload,
) (*AutoPostInvoiceResult, error) {
	ctx = workflow.WithActivityOptions(ctx, autoPostInvoiceActivityOptions)

	var a *Activities
	result := new(AutoPostInvoiceResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.AutoPostInvoiceActivity,
		payload,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Invoice auto-post workflow failed", "error", err)
		return nil, err
	}

	return result, nil
}

func SendInvoiceEmailWorkflow(
	ctx workflow.Context,
	payload *SendInvoiceEmailPayload,
) (*SendInvoiceEmailResult, error) {
	ctx = workflow.WithActivityOptions(ctx, autoPostInvoiceActivityOptions)

	var a *Activities
	result := new(SendInvoiceEmailResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.SendInvoiceEmailActivity,
		payload,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Invoice email workflow failed", "error", err)
		return nil, err
	}

	return result, nil
}

func GenerateInvoicePDFWorkflow(
	ctx workflow.Context,
	payload *GenerateInvoicePDFPayload,
) (*GenerateInvoicePDFResult, error) {
	ctx = workflow.WithActivityOptions(ctx, autoPostInvoiceActivityOptions)

	var a *Activities
	prepared := new(PrepareInvoicePDFUploadResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.PrepareInvoicePDFUploadActivity,
		payload,
	).Get(ctx, prepared); err != nil {
		workflow.GetLogger(ctx).Error("Invoice PDF preparation failed", "error", err)
		return nil, err
	}

	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: fmt.Sprintf("document-upload-finalize-%s", prepared.SessionID.String()),
		TaskQueue:  temporaltype.UploadTaskQueue,
	})
	finalized := new(documentuploadjobs.FinalizeUploadResult)
	if err := workflow.ExecuteChildWorkflow(
		childCtx,
		documentuploadjobs.FinalizeDocumentUploadWorkflow,
		&documentuploadjobs.FinalizeUploadPayload{
			BasePayload:   payload.BasePayload,
			SessionID:     prepared.SessionID,
			PrincipalType: payload.PrincipalType,
			PrincipalID:   payload.PrincipalID,
			APIKeyID:      payload.APIKeyID,
		},
	).Get(ctx, finalized); err != nil {
		workflow.GetLogger(ctx).Error("Invoice PDF upload finalization failed", "error", err)
		return nil, err
	}
	if finalized.DocumentID == nil || finalized.DocumentID.IsNil() {
		return nil, temporal.NewNonRetryableApplicationError(
			"Invoice PDF finalization did not produce a document",
			"invoice-pdf-document-missing",
			nil,
		)
	}

	result := new(GenerateInvoicePDFResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.CompleteInvoicePDFGenerationActivity,
		payload,
		*finalized.DocumentID,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Invoice PDF completion failed", "error", err)
		return nil, err
	}

	return result, nil
}
