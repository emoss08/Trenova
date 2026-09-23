package documentintelligencejobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/boolutils"
	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

const (
	documentAIExtractionPollInterval = 20 * time.Second
	documentAIExtractionMaxWait      = 15 * time.Minute
)

type AsyncAIExtractionCompletion struct {
	ResponseID      string                                `json:"responseId"`
	Model           string                                `json:"model"`
	ExtractedAt     int64                                 `json:"extractedAt"`
	Status          services.AIBackgroundExtractionStatus `json:"status"`
	RawStatus       string                                `json:"rawStatus"`
	ExtractResult   *services.AIExtractResult             `json:"extractResult,omitempty"`
	FailureCode     string                                `json:"failureCode,omitempty"`
	FailureMessage  string                                `json:"failureMessage,omitempty"`
	SubmittedAt     *int64                                `json:"submittedAt,omitempty"`
	LastPolledAt    *int64                                `json:"lastPolledAt,omitempty"`
	AcceptanceState string                                `json:"acceptanceState,omitempty"`
}

type ApplyDocumentAIExtractionPayload struct {
	temporaltype.BasePayload

	DocumentID  pulid.ID                     `json:"documentId"`
	ExtractedAt int64                        `json:"extractedAt"`
	Completion  *AsyncAIExtractionCompletion `json:"completion"`
}

func (a *Activities) startAIExtractionWorkflow(
	ctx context.Context,
	doc *document.Document,
	userID pulid.ID,
	extractedAt int64,
) error {
	if a.workflowStarter == nil || !a.workflowStarter.Enabled() {
		return services.ErrWorkflowStarterDisabled
	}

	_, err := a.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID: fmt.Sprintf(
				"document-ai-extraction-%s-%d",
				doc.ID.String(),
				extractedAt,
			),
			TaskQueue:                                temporaltype.DocumentIntelligenceTaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			StaticSummary: fmt.Sprintf(
				"Running AI extraction for document %s",
				doc.ID,
			),
		},
		"ProcessDocumentAIExtractionWorkflow",
		&ProcessDocumentAIExtractionPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: doc.OrganizationID,
				BusinessUnitID: doc.BusinessUnitID,
				UserID:         userID,
			},
			DocumentID:  doc.ID,
			ExtractedAt: extractedAt,
		},
	)
	if err != nil {
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStarted) {
			return nil
		}
	}

	return err
}

// submitAIExtraction records the extraction and submits it to the model once.
// It returns the completion when there is nothing to wait for: the extraction
// was superseded or already settled, or the router answered inline. A nil
// completion means the model is still working on it.
//
// taskToken is kept on the record for an execution that waits on the old
// poller to complete its activity. An execution that polls on its own timer
// keeps none, which is also what keeps the old poller away from its record.
func (a *Activities) submitAIExtraction( //nolint:funlen // async submission with idempotency checks
	ctx context.Context,
	payload *ProcessDocumentAIExtractionPayload,
	taskToken []byte,
) (*AsyncAIExtractionCompletion, error) {
	if a.aiExtractionRepo == nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"Document AI extraction repository is not configured",
			temporaltype.ErrorTypeNonRetryable.String(),
			nil,
		)
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	doc, err := a.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         payload.DocumentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	content, err := a.contentRepo.GetByDocumentID(ctx, payload.DocumentID, tenantInfo)
	if err != nil {
		return nil, err
	}
	if content.LastExtractedAt == nil || *content.LastExtractedAt != payload.ExtractedAt {
		return &AsyncAIExtractionCompletion{
			ExtractedAt:     payload.ExtractedAt,
			Status:          services.AIBackgroundExtractionStatusFailed,
			FailureCode:     "stale_extraction",
			FailureMessage:  "document extraction has been superseded",
			AcceptanceState: string(aiAcceptanceStatusRejected),
		}, nil
	}

	pages, err := a.contentRepo.ListPagesByDocumentID(ctx, payload.DocumentID, tenantInfo)
	if err != nil {
		return nil, err
	}

	requestHash := hashAIExtractionRequest(doc.OriginalName, content.ContentText, pages)
	info := activity.GetInfo(ctx)
	row := &documentaiextraction.Extraction{
		DocumentID:     payload.DocumentID,
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		UserID:         payload.UserID,
		ExtractedAt:    payload.ExtractedAt,
		RequestHash:    requestHash,
		WorkflowID:     info.WorkflowExecution.ID,
		WorkflowRunID:  info.WorkflowExecution.RunID,
		ActivityID:     info.ActivityID,
		TaskToken:      taskToken,
		Status:         documentaiextraction.StatusPending,
	}
	row, err = a.aiExtractionRepo.SavePending(ctx, row)
	if err != nil {
		return nil, err
	}
	switch row.Status {
	case documentaiextraction.StatusApplied:
		return &AsyncAIExtractionCompletion{
			ResponseID:      row.ResponseID,
			Model:           row.Model,
			ExtractedAt:     payload.ExtractedAt,
			Status:          services.AIBackgroundExtractionStatusCompleted,
			RawStatus:       string(row.Status),
			SubmittedAt:     row.SubmittedAt,
			LastPolledAt:    row.LastPolledAt,
			AcceptanceState: string(aiAcceptanceStatusAccepted),
			FailureCode:     "already_finalized",
			FailureMessage:  "AI extraction was already applied for this document extraction",
		}, nil
	case documentaiextraction.StatusSkipped:
		return &AsyncAIExtractionCompletion{
			ResponseID:      row.ResponseID,
			Model:           row.Model,
			ExtractedAt:     payload.ExtractedAt,
			Status:          services.AIBackgroundExtractionStatusFailed,
			RawStatus:       string(row.Status),
			SubmittedAt:     row.SubmittedAt,
			LastPolledAt:    row.LastPolledAt,
			AcceptanceState: string(aiAcceptanceStatusRejected),
			FailureCode:     "already_finalized",
			FailureMessage:  "AI extraction was already superseded for this document extraction",
		}, nil
	case documentaiextraction.StatusPending,
		documentaiextraction.StatusCompleted,
		documentaiextraction.StatusFailed:
	}

	if strings.TrimSpace(row.ResponseID) == "" {
		submission, submitErr := a.aiDocumentService.SubmitRateConfirmationBackgroundExtraction(
			ctx,
			&services.AIExtractRequest{
				TenantInfo: tenantInfo,
				DocumentID: doc.ID,
				FileName:   doc.OriginalName,
				Text: stringutils.TruncateAndTrim(
					content.ContentText,
					a.cfg.GetMaxInputChars(),
				),
				Pages: toAIDocumentPages(pages, a.cfg.GetMaxInputChars()),
			},
		)
		if submitErr != nil {
			return nil, modelcall.Classify(submitErr)
		}

		now := timeutils.NowUnix()
		row.ResponseID = submission.ResponseID
		row.ProviderID = submission.ProviderID
		row.Model = submission.Model
		row.SubmittedAt = &now
		row.FailureCode = ""
		row.FailureMessage = ""

		// No configured provider could defer the call, so the router ran it
		// inline and the answer is already here. Returning it completes the
		// activity now rather than parking a row the poller can never resolve:
		// there is no handle to poll, and the result cannot be asked for twice.
		if submission.ExtractResult != nil {
			row.Status = documentaiextraction.StatusCompleted
			row.CompletedAt = &now
			if _, err = a.aiExtractionRepo.Update(ctx, row); err != nil {
				return nil, err
			}

			return &AsyncAIExtractionCompletion{
				Model:           submission.Model,
				ExtractedAt:     payload.ExtractedAt,
				Status:          services.AIBackgroundExtractionStatusCompleted,
				RawStatus:       "inline",
				ExtractResult:   submission.ExtractResult,
				SubmittedAt:     &now,
				AcceptanceState: string(aiAcceptanceStatusAccepted),
			}, nil
		}

		if _, err = a.aiExtractionRepo.Update(ctx, row); err != nil {
			return nil, err
		}

		content.StructuredData = markAIDiagnosticsPending(
			content.StructuredData,
			row.ResponseID,
			&now,
		)
		if _, err = a.contentRepo.Upsert(ctx, content); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// SubmitAndAwaitDocumentAIExtractionActivity submits the extraction and leaves
// the activity open for the poller to complete with its task token. It serves
// only executions started before the workflow polled on its own timer.
func (a *Activities) SubmitAndAwaitDocumentAIExtractionActivity(
	ctx context.Context,
	payload *ProcessDocumentAIExtractionPayload,
) (*AsyncAIExtractionCompletion, error) {
	completion, err := a.submitAIExtraction(
		ctx, payload, append([]byte(nil), activity.GetInfo(ctx).TaskToken...),
	)
	if err != nil {
		return nil, err
	}
	if completion == nil {
		return nil, activity.ErrResultPending
	}

	return completion, nil
}

// SubmitDocumentAIExtractionActivity submits the extraction and returns at
// once. The workflow waits on a durable timer and polls for the answer.
func (a *Activities) SubmitDocumentAIExtractionActivity(
	ctx context.Context,
	payload *ProcessDocumentAIExtractionPayload,
) (*AIExtractionProgress, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	completion, err := a.submitAIExtraction(ctx, payload, []byte{})
	if err != nil {
		return nil, err
	}

	return &AIExtractionProgress{Completion: completion}, nil
}

// PollDocumentAIExtractionActivity asks the model once whether the extraction
// is done. A poll that fails is reported as still pending, since the next
// poll asks again; the last poll the workflow allows gives up instead, and
// the extraction is recorded as timed out.
func (a *Activities) PollDocumentAIExtractionActivity(
	ctx context.Context,
	input *PollDocumentAIExtractionInput,
) (*AIExtractionProgress, error) {
	payload := input.Payload
	row, err := a.aiExtractionRepo.GetByDocumentExtractedAt(
		ctx,
		repositories.GetDocumentAIExtractionRequest{
			DocumentID:  payload.DocumentID,
			ExtractedAt: payload.ExtractedAt,
			TenantInfo: pagination.TenantInfo{
				OrgID:  payload.OrganizationID,
				BuID:   payload.BusinessUnitID,
				UserID: payload.UserID,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	completion := a.pollExtraction(ctx, row, now)
	if completion == nil && input.GiveUp {
		completion = timedOutExtraction(row)
	}
	if completion == nil {
		a.touchExtraction(ctx, row)

		return &AIExtractionProgress{}, nil
	}

	a.settleExtraction(ctx, row, completion, now.Unix())

	return &AIExtractionProgress{Completion: completion}, nil
}

// PollPendingDocumentAIExtractionsActivity completes the activities of
// executions that wait on a task token. It serves only executions started
// before the workflow polled on its own timer, and records without a task
// token are not listed for it.
func (a *Activities) PollPendingDocumentAIExtractionsActivity(
	ctx context.Context,
	payload *PollPendingDocumentAIExtractionsPayload,
) (*PollPendingDocumentAIExtractionsResult, error) {
	result := &PollPendingDocumentAIExtractionsResult{}
	if a.aiExtractionRepo == nil || a.temporalClient == nil {
		return result, nil
	}

	now := time.Now()
	rows, err := a.aiExtractionRepo.ListPollable(
		ctx,
		&repositories.ListPollableDocumentAIExtractionRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: payload.OrganizationID,
				BuID:  payload.BusinessUnitID,
			},
			OlderThan: now.Add(-documentAIExtractionPollInterval).Unix(),
			Limit:     payload.Limit,
		},
	)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		completion := a.pollExtraction(ctx, row, now)
		if completion == nil {
			a.touchExtraction(ctx, row)
			result.Pending++

			continue
		}

		if !a.completeAsyncAIActivity(ctx, row, completion) {
			continue
		}

		a.settleExtraction(ctx, row, completion, now.Unix())
		if completion.Status == services.AIBackgroundExtractionStatusCompleted {
			result.Completed++
		} else {
			result.Failed++
		}
	}

	return result, nil
}

// pollExtraction asks the model whether a submitted extraction is done, and
// returns its completion when it is. An extraction past the longest wait is
// complete as timed out; one the model is still working on, or that could not
// be asked about just now, returns nil.
func (a *Activities) pollExtraction(
	ctx context.Context,
	row *documentaiextraction.Extraction,
	now time.Time,
) *AsyncAIExtractionCompletion {
	polledAt := now.Unix()
	row.LastPolledAt = &polledAt

	if row.SubmittedAt != nil &&
		now.Sub(time.Unix(*row.SubmittedAt, 0)) > documentAIExtractionMaxWait {
		return timedOutExtraction(row)
	}

	poll, err := a.aiDocumentService.PollRateConfirmationBackgroundExtraction(
		ctx,
		&services.AIBackgroundExtractPollRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  row.OrganizationID,
				BuID:   row.BusinessUnitID,
				UserID: row.UserID,
			},
			DocumentID: row.DocumentID,
			ResponseID: row.ResponseID,
			ProviderID: row.ProviderID,
		},
	)
	if err != nil {
		a.logger.Warn(
			"failed to poll background AI extraction",
			zap.String("documentId", row.DocumentID.String()),
			zap.Error(err),
		)

		return nil
	}
	if poll.Status == services.AIBackgroundExtractionStatusPending {
		return nil
	}

	completion := &AsyncAIExtractionCompletion{
		ResponseID:      poll.ResponseID,
		Model:           poll.Model,
		ExtractedAt:     row.ExtractedAt,
		Status:          poll.Status,
		RawStatus:       poll.RawStatus,
		ExtractResult:   poll.ExtractResult,
		FailureCode:     poll.FailureCode,
		FailureMessage:  poll.FailureMessage,
		SubmittedAt:     row.SubmittedAt,
		LastPolledAt:    row.LastPolledAt,
		AcceptanceState: string(aiAcceptanceStatusRejected),
	}
	if poll.Status == services.AIBackgroundExtractionStatusCompleted {
		completion.AcceptanceState = string(aiAcceptanceStatusAccepted)
	}

	return completion
}

// timedOutExtraction is an extraction that waited as long as it may.
func timedOutExtraction(row *documentaiextraction.Extraction) *AsyncAIExtractionCompletion {
	return &AsyncAIExtractionCompletion{
		ResponseID:      row.ResponseID,
		Model:           row.Model,
		ExtractedAt:     row.ExtractedAt,
		Status:          services.AIBackgroundExtractionStatusFailed,
		RawStatus:       "expired",
		FailureCode:     "ai_extract_timeout",
		FailureMessage:  "Background AI extraction exceeded the maximum wait time",
		SubmittedAt:     row.SubmittedAt,
		LastPolledAt:    row.LastPolledAt,
		AcceptanceState: string(aiAcceptanceStatusRejected),
	}
}

// touchExtraction records that a pending extraction was polled, so the next
// poll of many goes to the one asked about longest ago.
func (a *Activities) touchExtraction(ctx context.Context, row *documentaiextraction.Extraction) {
	if _, err := a.aiExtractionRepo.Update(ctx, row); err != nil {
		a.logger.Warn(
			"failed to record a pending AI extraction's poll",
			zap.String("documentId", row.DocumentID.String()),
			zap.Error(err),
		)
	}
}

// settleExtraction records how an extraction ended.
func (a *Activities) settleExtraction(
	ctx context.Context,
	row *documentaiextraction.Extraction,
	completion *AsyncAIExtractionCompletion,
	settledAt int64,
) {
	row.CompletedAt = &settledAt
	if completion.Status == services.AIBackgroundExtractionStatusCompleted {
		row.Status = documentaiextraction.StatusCompleted
		row.FailureCode = ""
		row.FailureMessage = ""
	} else {
		row.Status = documentaiextraction.StatusFailed
		row.FailureCode = completion.FailureCode
		row.FailureMessage = completion.FailureMessage
	}

	if _, err := a.aiExtractionRepo.Update(ctx, row); err != nil {
		a.logger.Warn(
			"failed to record how an AI extraction ended",
			zap.String("documentId", row.DocumentID.String()),
			zap.Error(err),
		)
	}
}

func (a *Activities) ListPollableDocumentAIExtractionTenantsActivity(
	ctx context.Context,
	payload *PollPendingDocumentAIExtractionsPayload,
) (*ListDocumentIntelligenceTenantsResult, error) {
	if a.aiExtractionRepo == nil {
		return &ListDocumentIntelligenceTenantsResult{}, nil
	}

	tenants, err := a.aiExtractionRepo.ListPollableTenants(
		ctx,
		&repositories.ListPollableDocumentAIExtractionRequest{
			OlderThan: time.Now().Add(-documentAIExtractionPollInterval).Unix(),
			Limit:     temporaljobs.DefaultTenantScanLimit,
		},
	)
	if err != nil {
		return nil, err
	}

	tenantLimit := temporaljobs.NormalizeLimit(payload.Limit, temporaljobs.DefaultTenantRecordLimit)
	return &ListDocumentIntelligenceTenantsResult{
		Tenants: temporaljobs.BuildTenantWorkItems(tenants, tenantLimit),
	}, nil
}

func (a *Activities) ApplyDocumentAIExtractionResultActivity(
	ctx context.Context,
	payload *ApplyDocumentAIExtractionPayload,
) (*ProcessDocumentAIExtractionResult, error) {
	if payload == nil || payload.Completion == nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"AI extraction completion payload is required",
			temporaltype.ErrorTypeInvalidInput.String(),
			nil,
		)
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}
	doc, err := a.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         payload.DocumentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	content, err := a.contentRepo.GetByDocumentID(ctx, payload.DocumentID, tenantInfo)
	if err != nil {
		return nil, err
	}

	if content.LastExtractedAt == nil || *content.LastExtractedAt != payload.ExtractedAt {
		a.markExtractionSkipped(ctx, payload, tenantInfo)
		return &ProcessDocumentAIExtractionResult{
			DocumentID:      payload.DocumentID,
			ExtractedAt:     payload.ExtractedAt,
			AcceptanceState: string(aiAcceptanceStatusRejected),
		}, nil
	}
	if payload.Completion.FailureCode == "already_finalized" {
		return &ProcessDocumentAIExtractionResult{
			DocumentID:      payload.DocumentID,
			ExtractedAt:     payload.ExtractedAt,
			AcceptanceState: aiAcceptanceStatusFromStructuredData(content.StructuredData),
		}, nil
	}

	updatedIntelligence, diagnostics := a.mergeCompletionIntoIntelligence(content, payload)

	content.StructuredData = buildStructuredData(updatedIntelligence, diagnostics)
	content.ClassificationConfidence = updatedIntelligence.OverallConfidence
	if _, err = a.contentRepo.Upsert(ctx, content); err != nil {
		return nil, err
	}

	control, err := a.getDocumentControl(ctx, doc.OrganizationID, doc.BusinessUnitID)
	if err != nil {
		return nil, err
	}
	if err = a.updateShipmentDraftFromIntelligence(
		ctx, doc, control, updatedIntelligence,
	); err != nil {
		return nil, err
	}

	indexedText := content.ContentText
	if !control.EnableFullTextIndexing {
		indexedText = ""
	}
	a.syncSearchProjection(ctx, doc, indexedText)
	a.markExtractionApplied(ctx, payload, tenantInfo)

	return &ProcessDocumentAIExtractionResult{
		DocumentID:      payload.DocumentID,
		ExtractedAt:     payload.ExtractedAt,
		AcceptanceState: string(diagnostics.AcceptanceStatus),
	}, nil
}

func (a *Activities) markExtractionSkipped(
	ctx context.Context,
	payload *ApplyDocumentAIExtractionPayload,
	tenantInfo pagination.TenantInfo,
) {
	if a.aiExtractionRepo == nil {
		return
	}
	row, repoErr := a.aiExtractionRepo.GetByDocumentExtractedAt(
		ctx,
		repositories.GetDocumentAIExtractionRequest{
			DocumentID:  payload.DocumentID,
			ExtractedAt: payload.ExtractedAt,
			TenantInfo:  tenantInfo,
		},
	)
	if repoErr != nil {
		return
	}
	row.Status = documentaiextraction.StatusSkipped
	row.FailureCode = "stale_extraction"
	row.FailureMessage = "document extraction has been superseded"
	if _, repoErr = a.aiExtractionRepo.Update(ctx, row); repoErr != nil {
		a.logger.Warn(
			"failed to mark stale AI extraction row skipped",
			zap.String("documentId", payload.DocumentID.String()),
			zap.Error(repoErr),
		)
	}
}

func (a *Activities) markExtractionApplied(
	ctx context.Context,
	payload *ApplyDocumentAIExtractionPayload,
	tenantInfo pagination.TenantInfo,
) {
	if a.aiExtractionRepo == nil {
		return
	}
	row, repoErr := a.aiExtractionRepo.GetByDocumentExtractedAt(
		ctx,
		repositories.GetDocumentAIExtractionRequest{
			DocumentID:  payload.DocumentID,
			ExtractedAt: payload.ExtractedAt,
			TenantInfo:  tenantInfo,
		},
	)
	if repoErr != nil {
		return
	}
	row.Status = documentaiextraction.StatusApplied
	if _, repoErr = a.aiExtractionRepo.Update(ctx, row); repoErr != nil {
		a.logger.Warn(
			"failed to mark AI extraction row applied",
			zap.String("documentId", payload.DocumentID.String()),
			zap.Error(repoErr),
		)
	}
}

func (a *Activities) mergeCompletionIntoIntelligence(
	content *documentcontent.Content,
	payload *ApplyDocumentAIExtractionPayload,
) (*DocumentIntelligenceAnalysis, *AIDiagnostics) {
	fallback := analysisFromStructuredData(content.StructuredData)
	diagnostics := &AIDiagnostics{
		FallbackAnalysis: fallback,
		AcceptanceStatus: aiAcceptanceStatusRejected,
		RejectionReason:  payload.Completion.FailureCode,
		ResponseID:       payload.Completion.ResponseID,
		SubmittedAt:      payload.Completion.SubmittedAt,
		LastPolledAt:     payload.Completion.LastPolledAt,
	}

	updatedIntelligence := fallback
	if payload.Completion.Status == services.AIBackgroundExtractionStatusCompleted &&
		payload.Completion.ExtractResult != nil {
		candidate := analysisFromAIExtract(payload.Completion.ExtractResult)
		diagnostics.CandidateAnalysis = candidate
		merged, ok, rejectionReason := mergeAIAnalysis(
			fallback,
			payload.Completion.ExtractResult,
		)
		if ok {
			diagnostics.AcceptanceStatus = aiAcceptanceStatusAccepted
			diagnostics.RejectionReason = ""
			updatedIntelligence = merged
		} else {
			diagnostics.AcceptanceStatus = aiAcceptanceStatusRejected
			diagnostics.RejectionReason = rejectionReason
		}
	}

	return updatedIntelligence, diagnostics
}

func aiAcceptanceStatusFromStructuredData(structured map[string]any) string {
	if diagnostics, ok := structured["aiDiagnostics"].(map[string]any); ok {
		if status := sliceutils.StringValue(diagnostics["acceptanceStatus"]); status != "" {
			return status
		}
	}

	return string(aiAcceptanceStatusNotAttempted)
}

func (a *Activities) completeAsyncAIActivity(
	ctx context.Context,
	row *documentaiextraction.Extraction,
	completion *AsyncAIExtractionCompletion,
) bool {
	if row == nil || len(row.TaskToken) == 0 || completion == nil {
		return false
	}

	if err := a.temporalClient.CompleteActivity(ctx, row.TaskToken, completion, nil); err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			a.logger.Warn(
				"pending AI extraction activity no longer exists",
				zap.String("documentId", row.DocumentID.String()),
				zap.Error(err),
			)
			return true
		}
		a.logger.Warn(
			"failed to complete pending AI extraction activity",
			zap.String("documentId", row.DocumentID.String()),
			zap.Error(err),
		)
		return false
	}

	return true
}

func hashAIExtractionRequest(fileName, text string, pages []*documentcontent.Page) string {
	sum := sha256.New()
	sum.Write([]byte(strings.TrimSpace(fileName)))
	sum.Write([]byte{'\n'})
	sum.Write([]byte(strings.TrimSpace(text)))
	for _, page := range pages {
		fmt.Fprintf(sum, "\n[%d]\n", page.PageNumber)
		sum.Write([]byte(strings.TrimSpace(page.ExtractedText)))
	}

	return hex.EncodeToString(sum.Sum(nil))
}

func toAIDocumentPages(pages []*documentcontent.Page, maxChars int) []services.AIDocumentPage {
	out := make([]services.AIDocumentPage, 0, len(pages))
	pageLimit := maxChars / max(len(pages), 1)
	for _, page := range pages {
		if strings.TrimSpace(page.ExtractedText) == "" {
			continue
		}
		out = append(out, services.AIDocumentPage{
			PageNumber: page.PageNumber,
			Text:       stringutils.TruncateAndTrim(page.ExtractedText, pageLimit),
		})
	}
	return out
}

func markAIDiagnosticsPending(
	structured map[string]any,
	responseID string,
	submittedAt *int64,
) map[string]any {
	if structured == nil {
		structured = map[string]any{}
	}
	diagnostics, _ := structured["aiDiagnostics"].(map[string]any)
	if diagnostics == nil {
		diagnostics = map[string]any{}
	}
	diagnostics["acceptanceStatus"] = aiAcceptanceStatusPending
	diagnostics["rejectionReason"] = ""
	if responseID != "" {
		diagnostics["responseId"] = responseID
	}
	if submittedAt != nil {
		diagnostics["submittedAt"] = *submittedAt
	}
	structured["aiDiagnostics"] = diagnostics
	return structured
}

func analysisFromStructuredData(structured map[string]any) *DocumentIntelligenceAnalysis {
	if structured == nil {
		return &DocumentIntelligenceAnalysis{
			Fields:    map[string]*ReviewField{},
			Stops:     []*IntelligenceStop{},
			Conflicts: []*ReviewConflict{},
		}
	}
	if aiDiagnostics, ok := structured["aiDiagnostics"].(map[string]any); ok {
		if fallback, hasFallback := aiDiagnostics["fallbackAnalysis"].(map[string]any); hasFallback {
			return analysisFromMap(fallback)
		}
	}
	if intelligence, ok := structured["intelligence"].(map[string]any); ok {
		return analysisFromMap(intelligence)
	}
	return &DocumentIntelligenceAnalysis{
		Fields:    map[string]*ReviewField{},
		Stops:     []*IntelligenceStop{},
		Conflicts: []*ReviewConflict{},
	}
}

func analysisFromMap(data map[string]any) *DocumentIntelligenceAnalysis {
	analysis := &DocumentIntelligenceAnalysis{
		Kind:                 sliceutils.StringValue(data["kind"]),
		OverallConfidence:    floatutils.FloatValue(data["overallConfidence"]),
		ReviewStatus:         sliceutils.StringValue(data["reviewStatus"]),
		MissingFields:        sliceutils.StringSliceValue(data["missingFields"]),
		Signals:              sliceutils.StringSliceValue(data["signals"]),
		ClassifierSource:     sliceutils.StringValue(data["classifierSource"]),
		ProviderFingerprint:  sliceutils.StringValue(data["providerFingerprint"]),
		ClassificationReason: sliceutils.StringValue(data["classificationReason"]),
		RawExcerpt:           sliceutils.StringValue(data["rawExcerpt"]),
		Fields:               map[string]*ReviewField{},
		Stops:                []*IntelligenceStop{},
		Conflicts:            []*ReviewConflict{},
	}

	if metadata, ok := data["parsingRuleMetadata"].(map[string]any); ok {
		analysis.ParsingRuleMetadata = parseParsingRuleMetadata(metadata)
	}
	if fields, ok := data["fields"].(map[string]any); ok {
		for key, raw := range fields {
			fieldMap, isFieldMap := raw.(map[string]any)
			if !isFieldMap {
				continue
			}
			analysis.Fields[key] = &ReviewField{
				Label:      sliceutils.StringValue(fieldMap["label"]),
				Value:      sliceutils.StringValue(fieldMap["value"]),
				Confidence: floatutils.FloatValue(fieldMap["confidence"]),
				Excerpt:    sliceutils.StringValue(fieldMap["excerpt"]),
				EvidenceExcerpt: stringutils.FirstNonEmpty(
					sliceutils.StringValue(fieldMap["evidenceExcerpt"]),
					sliceutils.StringValue(fieldMap["excerpt"]),
				),
				PageNumber:     intutils.IntValue(fieldMap["pageNumber"]),
				ReviewRequired: boolutils.BooleanValue(fieldMap["reviewRequired"]),
				Conflict:       boolutils.BooleanValue(fieldMap["conflict"]),
				Source:         sliceutils.StringValue(fieldMap["source"]),
			}
		}
	}
	if stops, ok := data["stops"].([]any); ok {
		for _, raw := range stops {
			stopMap, isStopMap := raw.(map[string]any)
			if !isStopMap {
				continue
			}
			analysis.Stops = append(analysis.Stops, &IntelligenceStop{
				Sequence:            intutils.IntValue(stopMap["sequence"]),
				Role:                sliceutils.StringValue(stopMap["role"]),
				Name:                sliceutils.StringValue(stopMap["name"]),
				AddressLine1:        sliceutils.StringValue(stopMap["addressLine1"]),
				AddressLine2:        sliceutils.StringValue(stopMap["addressLine2"]),
				City:                sliceutils.StringValue(stopMap["city"]),
				State:               sliceutils.StringValue(stopMap["state"]),
				PostalCode:          sliceutils.StringValue(stopMap["postalCode"]),
				Date:                sliceutils.StringValue(stopMap["date"]),
				TimeWindow:          sliceutils.StringValue(stopMap["timeWindow"]),
				AppointmentRequired: boolutils.BooleanValue(stopMap["appointmentRequired"]),
				PageNumber:          intutils.IntValue(stopMap["pageNumber"]),
				EvidenceExcerpt:     sliceutils.StringValue(stopMap["evidenceExcerpt"]),
				Confidence:          floatutils.FloatValue(stopMap["confidence"]),
				ReviewRequired:      boolutils.BooleanValue(stopMap["reviewRequired"]),
				Source:              sliceutils.StringValue(stopMap["source"]),
			})
		}
	}
	if conflicts, ok := data["conflicts"].([]any); ok {
		for _, raw := range conflicts {
			conflictMap, isConflictMap := raw.(map[string]any)
			if !isConflictMap {
				continue
			}
			analysis.Conflicts = append(analysis.Conflicts, &ReviewConflict{
				Key:             sliceutils.StringValue(conflictMap["key"]),
				Label:           sliceutils.StringValue(conflictMap["label"]),
				Values:          sliceutils.StringSliceValue(conflictMap["values"]),
				PageNumbers:     intutils.IntSliceValue(conflictMap["pageNumbers"]),
				EvidenceExcerpt: sliceutils.StringValue(conflictMap["evidenceExcerpt"]),
				Source:          sliceutils.StringValue(conflictMap["source"]),
			})
		}
	}

	return analysis
}

func parseParsingRuleMetadata(data map[string]any) *services.DocumentParsingRuleMetadata {
	ruleSetID, err := pulid.Parse(sliceutils.StringValue(data["ruleSetId"]))
	if err != nil {
		return nil
	}
	ruleVersionID, err := pulid.Parse(sliceutils.StringValue(data["ruleVersionId"]))
	if err != nil {
		return nil
	}
	return &services.DocumentParsingRuleMetadata{
		RuleSetID:        ruleSetID,
		RuleSetName:      sliceutils.StringValue(data["ruleSetName"]),
		RuleVersionID:    ruleVersionID,
		VersionNumber:    intutils.IntValue(data["versionNumber"]),
		ParserMode:       sliceutils.StringValue(data["parserMode"]),
		ProviderMatched:  sliceutils.StringValue(data["providerMatched"]),
		MatchSpecificity: intutils.IntValue(data["matchSpecificity"]),
	}
}

func (a *Activities) updateShipmentDraftFromIntelligence(
	ctx context.Context,
	doc *document.Document,
	control *tenant.DocumentControl,
	intelligence *DocumentIntelligenceAnalysis,
) error {
	isUsable := canGenerateShipmentDraft(control, doc.ResourceType, doc.DetectedKind) &&
		hasUsableShipmentDraft(intelligence)
	draftStatus, err := a.upsertDraft(ctx, doc, doc.DetectedKind, intelligence, isUsable)
	if err != nil {
		return err
	}

	doc.ShipmentDraftStatus = draftStatus
	return a.documentRepo.UpdateIntelligence(ctx, &repositories.UpdateDocumentIntelligenceRequest{
		ID: doc.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: doc.OrganizationID,
			BuID:  doc.BusinessUnitID,
		},
		ContentStatus:       doc.ContentStatus,
		ContentError:        doc.ContentError,
		DetectedKind:        doc.DetectedKind,
		HasExtractedText:    doc.HasExtractedText,
		ShipmentDraftStatus: doc.ShipmentDraftStatus,
		DocumentTypeID:      doc.DocumentTypeID,
	})
}

func (a *Activities) upsertDraft(
	ctx context.Context,
	doc *document.Document,
	kind string,
	intelligence *DocumentIntelligenceAnalysis,
	isReady bool,
) (document.ShipmentDraftStatus, error) {
	draftState := documentshipmentdraft.StatusUnavailable
	draftStatus := document.ShipmentDraftStatusUnavailable
	if isReady {
		draftState = documentshipmentdraft.StatusReady
		draftStatus = document.ShipmentDraftStatusReady
	}

	if _, err := a.draftRepo.Upsert(ctx, &documentshipmentdraft.DocumentShipmentDraft{
		DocumentID:     doc.ID,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		Status:         draftState,
		DocumentKind:   kind,
		Confidence:     intelligence.OverallConfidence,
		DraftData:      intelligence.ToMap(),
	}); err != nil {
		return "", err
	}

	services.PublishAgentEvent(ctx, a.agentEvents, services.AgentEvent{
		Kind:      agent.EventDocumentExtracted,
		SubjectID: doc.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: doc.OrganizationID,
			BuID:  doc.BusinessUnitID,
		},
	})

	return draftStatus, nil
}
