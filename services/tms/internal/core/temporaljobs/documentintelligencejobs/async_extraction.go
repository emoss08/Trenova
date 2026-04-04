package documentintelligencejobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	services "github.com/emoss08/trenova/internal/core/ports/services"
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

func (a *Activities) SubmitAndAwaitDocumentAIExtractionActivity( //nolint:funlen // async submission with idempotency checks
	ctx context.Context,
	payload *ProcessDocumentAIExtractionPayload,
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
		TaskToken:      append([]byte(nil), info.TaskToken...),
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
					a.cfg.GetAIMaxInputChars(),
				),
				Pages: toAIDocumentPages(pages, a.cfg.GetAIMaxInputChars()),
			},
		)
		if submitErr != nil {
			return nil, submitErr
		}

		now := timeutils.NowUnix()
		row.ResponseID = submission.ResponseID
		row.Model = submission.Model
		row.SubmittedAt = &now
		row.FailureCode = ""
		row.FailureMessage = ""
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

	return nil, activity.ErrResultPending
}

func (a *Activities) PollPendingDocumentAIExtractionsActivity( //nolint:gocognit,funlen // polling loop with per-row branching
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
		now.Add(-documentAIExtractionPollInterval).Unix(),
		payload.Limit,
	)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		polledAt := now.Unix()
		row.LastPolledAt = &polledAt

		if row.SubmittedAt != nil &&
			now.Sub(time.Unix(*row.SubmittedAt, 0)) > documentAIExtractionMaxWait {
			completion := &AsyncAIExtractionCompletion{
				ResponseID:      row.ResponseID,
				Model:           row.Model,
				ExtractedAt:     row.ExtractedAt,
				Status:          services.AIBackgroundExtractionStatusFailed,
				RawStatus:       "expired",
				FailureCode:     "ai_extract_timeout",
				FailureMessage:  "OpenAI background extraction exceeded the maximum wait time",
				SubmittedAt:     row.SubmittedAt,
				LastPolledAt:    row.LastPolledAt,
				AcceptanceState: string(aiAcceptanceStatusRejected),
			}
			if a.completeAsyncAIActivity(ctx, row, completion) {
				row.Status = documentaiextraction.StatusFailed
				row.FailureCode = completion.FailureCode
				row.FailureMessage = completion.FailureMessage
				row.CompletedAt = &polledAt
				if _, err = a.aiExtractionRepo.Update(ctx, row); err == nil {
					result.Failed++
				}
			}
			continue
		}

		poll, pollErr := a.aiDocumentService.PollRateConfirmationBackgroundExtraction(
			ctx,
			&services.AIBackgroundExtractPollRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID:  row.OrganizationID,
					BuID:   row.BusinessUnitID,
					UserID: row.UserID,
				},
				DocumentID: row.DocumentID,
				ResponseID: row.ResponseID,
			},
		)
		if pollErr != nil {
			a.logger.Warn(
				"failed to poll background AI extraction",
				zap.String("documentId", row.DocumentID.String()),
				zap.Error(pollErr),
			)
			if _, err = a.aiExtractionRepo.Update(ctx, row); err != nil {
				a.logger.Warn(
					"failed to touch pending AI extraction poll time",
					zap.String("documentId", row.DocumentID.String()),
					zap.Error(err),
				)
			}
			continue
		}

		if poll.Status == services.AIBackgroundExtractionStatusPending {
			if _, err = a.aiExtractionRepo.Update(ctx, row); err != nil {
				a.logger.Warn(
					"failed to update pending AI extraction poll time",
					zap.String("documentId", row.DocumentID.String()),
					zap.Error(err),
				)
			}
			result.Pending++
			continue
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

		if !a.completeAsyncAIActivity(ctx, row, completion) {
			continue
		}

		row.CompletedAt = &polledAt
		switch poll.Status {
		case services.AIBackgroundExtractionStatusCompleted:
			row.Status = documentaiextraction.StatusCompleted
			row.FailureCode = ""
			row.FailureMessage = ""
			result.Completed++
		case services.AIBackgroundExtractionStatusFailed:
			row.Status = documentaiextraction.StatusFailed
			row.FailureCode = poll.FailureCode
			row.FailureMessage = poll.FailureMessage
			result.Failed++
		case services.AIBackgroundExtractionStatusPending:
			result.Pending++
		}
		if _, err = a.aiExtractionRepo.Update(ctx, row); err != nil {
			a.logger.Warn(
				"failed to update terminal AI extraction row",
				zap.String("documentId", row.DocumentID.String()),
				zap.Error(err),
			)
		}
	}

	return result, nil
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
				EvidenceExcerpt: firstNonEmpty(
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
		ProviderMatched:  sliceutils.StringValue(data["providerMatmatched"]),
		MatchSpecificity: intutils.IntValue(data["matchSpecificity"]),
	}
}

func (a *Activities) updateShipmentDraftFromIntelligence(
	ctx context.Context,
	doc *document.Document,
	control *tenant.DocumentControl,
	intelligence *DocumentIntelligenceAnalysis,
) error {
	draftStatus := document.ShipmentDraftStatusUnavailable
	if canGenerateShipmentDraft(control, doc.ResourceType, doc.DetectedKind) &&
		hasUsableShipmentDraft(intelligence) {
		if _, err := a.draftRepo.Upsert(ctx, &documentshipmentdraft.Draft{
			DocumentID:     doc.ID,
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
			Status:         documentshipmentdraft.StatusReady,
			DocumentKind:   doc.DetectedKind,
			Confidence:     intelligence.OverallConfidence,
			DraftData:      intelligence.ToMap(),
		}); err != nil {
			return err
		}
		draftStatus = document.ShipmentDraftStatusReady
	} else {
		draftState := documentshipmentdraft.StatusUnavailable
		if _, err := a.draftRepo.Upsert(ctx, &documentshipmentdraft.Draft{
			DocumentID:     doc.ID,
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
			Status:         draftState,
			DocumentKind:   doc.DetectedKind,
			Confidence:     intelligence.OverallConfidence,
			DraftData:      intelligence.ToMap(),
		}); err != nil {
			return err
		}
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
