package documentextractionjobs

import (
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs/documentextractioncontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	defaultConversionTimeout   = 10 * time.Minute
	conversionHeartbeatTimeout = time.Minute
)

var markConversionFailedOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    10,
	},
}

var convertRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    10 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    6,
	MaximumInterval:    5 * time.Minute,
	NonRetryableErrorTypes: []string{
		temporaltype.ErrorTypeInvalidInput.String(),
		temporaltype.ErrorTypeResourceNotFound.String(),
		temporaltype.ErrorTypeNonRetryable.String(),
	},
}

func RegisterExtractionWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        documentextractioncontract.ProcessWorkflowName,
			Fn:          ProcessDocumentExtractionWorkflow,
			TaskQueue:   temporaltype.DocumentExtractionTaskQueue,
			Description: "Convert a document with Docling, then classify and extract it",
		},
		{
			Name:        documentextractioncontract.ReconcileWorkflowName,
			Fn:          ReconcileDocumentExtractionWorkflow,
			TaskQueue:   temporaltype.DocumentExtractionTaskQueue,
			Description: "Re-drive stale document extraction jobs",
		},
		{
			Name:        "ProcessDocumentAIExtractionWorkflow",
			Fn:          ProcessDocumentAIExtractionWorkflow,
			TaskQueue:   temporaltype.DocumentExtractionTaskQueue,
			Description: "Run asynchronous AI extraction for an indexed document",
		},
	}
}

func ProcessDocumentExtractionWorkflow(
	ctx workflow.Context,
	payload *documentextractioncontract.ProcessDocumentExtractionPayload,
) (*ProcessDocumentIntelligenceResult, error) {
	stepCtx := workflow.WithActivityOptions(ctx, defaultActivityOptions)

	var a *Activities
	var plan PlanDocumentExtractionResult
	if err := workflow.ExecuteActivity(stepCtx, a.PlanDocumentExtractionActivity, payload).
		Get(ctx, &plan); err != nil {
		return nil, err
	}

	switch plan.Route {
	case ExtractionRouteDocling:
	case ExtractionRouteSkip, ExtractionRouteStopped:
		return plan.Result, nil
	default:
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("unknown extraction route %q", plan.Route),
			temporaltype.ErrorTypeNonRetryable.String(),
			nil,
		)
	}

	timeout := time.Duration(plan.ConversionTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultConversionTimeout
	}
	convertCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: timeout,
		HeartbeatTimeout:    conversionHeartbeatTimeout,
		RetryPolicy:         convertRetryPolicy,
	})

	var converted ConvertDocumentResult
	if err := workflow.ExecuteActivity(convertCtx, a.ConvertDocumentActivity, &ConvertDocumentInput{
		BasePayload: payload.BasePayload,
		DocumentID:  payload.DocumentID,
		OCR:         plan.OCR,
	}).Get(ctx, &converted); err != nil {
		return nil, markConversionFailed(ctx, payload, err)
	}
	if !converted.Converted {
		return &ProcessDocumentIntelligenceResult{
			DocumentID: payload.DocumentID,
			Status:     converted.Status,
		}, nil
	}

	var result *ProcessDocumentIntelligenceResult
	if err := workflow.ExecuteActivity(
		stepCtx,
		a.AnalyzeConvertedDocumentActivity,
		&AnalyzeConvertedDocumentInput{
			BasePayload: payload.BasePayload,
			DocumentID:  payload.DocumentID,
		},
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func markConversionFailed(
	ctx workflow.Context,
	payload *documentextractioncontract.ProcessDocumentExtractionPayload,
	convertErr error,
) error {
	if temporal.IsCanceledError(convertErr) {
		return convertErr
	}

	code, message := conversionFailureOf(convertErr)
	var a *Activities
	markCtx := workflow.WithActivityOptions(ctx, markConversionFailedOptions)
	if err := workflow.ExecuteActivity(markCtx, a.MarkConversionFailedActivity, &MarkConversionFailedInput{
		BasePayload: payload.BasePayload,
		DocumentID:  payload.DocumentID,
		Code:        code.String(),
		Message:     message,
	}).
		Get(ctx, nil); err != nil {
		workflow.GetLogger(ctx).Error("Could not mark the document's conversion failed",
			"documentId", payload.DocumentID.String(),
			"error", err,
		)
	}

	return convertErr
}

func conversionFailureOf(err error) (code documentupload.FailureCode, message string) {
	if appErr, ok := errors.AsType[*temporal.ApplicationError](err); ok &&
		appErr.Type() == errorTypeConverterUnavailable {
		return documentupload.FailureConverterUnavailable, converterUnreachableMessage
	}
	if _, ok := errors.AsType[*temporal.TimeoutError](err); ok {
		return documentupload.FailureConverterUnavailable, converterUnreachableMessage
	}

	return documentupload.FailureExtractionFailed, conversionFailedMessage
}

func ReconcileDocumentExtractionWorkflow(
	ctx workflow.Context,
) (*ReconcileDocumentIntelligenceResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, defaultActivityOptions)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	payload := &ReconcileDocumentIntelligencePayload{
		BasePayload: temporaltype.BasePayload{
			Timestamp: workflow.Now(ctx).Unix(),
		},
		OlderThanSeconds: int64((10 * time.Minute).Seconds()),
		Limit:            temporaljobs.DefaultTenantRecordLimit,
	}

	var tenantsResult *ListDocumentIntelligenceTenantsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListDocumentIntelligenceTenantsActivity,
		payload,
	).Get(ctx, &tenantsResult); err != nil {
		return nil, err
	}

	result := &ReconcileDocumentIntelligenceResult{}
	result.TenantsScanned = len(tenantsResult.Tenants)
	for _, tenant := range tenantsResult.Tenants {
		tenantPayload := *payload
		tenantPayload.OrganizationID = tenant.OrganizationID
		tenantPayload.BusinessUnitID = tenant.BusinessUnitID
		tenantPayload.Limit = temporaljobs.NormalizeLimit(
			tenant.Limit,
			temporaljobs.DefaultTenantRecordLimit,
		)

		var tenantResult *ReconcileDocumentIntelligenceResult
		if err := workflow.ExecuteActivity(
			activityCtx,
			a.ReconcileDocumentExtractionActivity,
			&tenantPayload,
		).Get(ctx, &tenantResult); err != nil {
			logger.Error("Document extraction reconciliation tenant failed",
				"orgId", tenant.OrganizationID.String(),
				"buId", tenant.BusinessUnitID.String(),
				"error", err,
			)
			result.AddFailure(tenant, err)
			continue
		}

		result.AddTenantResult(tenantResult.Queued, 0)
		result.Queued += tenantResult.Queued
	}

	return result, nil
}
