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
	"github.com/emoss08/trenova/shared/pulid"
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
			ID:                                       fmt.Sprintf("document-ai-extraction-%s-%d", doc.ID.String(), extractedAt),
			TaskQueue:                                temporaltype.DocumentIntelligenceTaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			StaticSummary:                            fmt.Sprintf("Running AI extraction for document %s", doc.ID),
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

func (a *Activities) SubmitAndAwaitDocumentAIExtractionActivity(
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
	}

	if strings.TrimSpace(row.ResponseID) == "" {
		submission, submitErr := a.aiDocumentService.SubmitRateConfirmationBackgroundExtraction(ctx, &services.AIExtractRequest{
			TenantInfo: tenantInfo,
			DocumentID: doc.ID,
			FileName:   doc.OriginalName,
			Text:       truncateExtractedText(content.ContentText, a.cfg.GetAIMaxInputChars()),
			Pages:      toAIDocumentPages(pages, a.cfg.GetAIMaxInputChars()),
		})
		if submitErr != nil {
			return nil, submitErr
		}

		now := time.Now().Unix()
		row.ResponseID = submission.ResponseID
		row.Model = submission.Model
		row.SubmittedAt = &now
		row.FailureCode = ""
		row.FailureMessage = ""
		if _, err = a.aiExtractionRepo.Update(ctx, row); err != nil {
			return nil, err
		}

		content.StructuredData = markAIDiagnosticsPending(content.StructuredData, row.ResponseID, &now)
		if _, err = a.contentRepo.Upsert(ctx, content); err != nil {
			return nil, err
		}
	}

	return nil, activity.ErrResultPending
}

func (a *Activities) PollPendingDocumentAIExtractionsActivity(
	ctx context.Context,
	payload *PollPendingDocumentAIExtractionsPayload,
) (*PollPendingDocumentAIExtractionsResult, error) {
	result := &PollPendingDocumentAIExtractionsResult{}
	if a.aiExtractionRepo == nil || a.temporalClient == nil {
		return result, nil
	}

	now := time.Now()
	rows, err := a.aiExtractionRepo.ListPollable(ctx, now.Add(-documentAIExtractionPollInterval).Unix(), payload.Limit)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		polledAt := now.Unix()
		row.LastPolledAt = &polledAt

		if row.SubmittedAt != nil && now.Sub(time.Unix(*row.SubmittedAt, 0)) > documentAIExtractionMaxWait {
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

		poll, pollErr := a.aiDocumentService.PollRateConfirmationBackgroundExtraction(ctx, &services.AIBackgroundExtractPollRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  row.OrganizationID,
				BuID:   row.BusinessUnitID,
				UserID: row.UserID,
			},
			DocumentID: row.DocumentID,
			ResponseID: row.ResponseID,
		})
		if pollErr != nil {
			a.logger.Warn("failed to poll background AI extraction", zap.String("documentId", row.DocumentID.String()), zap.Error(pollErr))
			if _, err = a.aiExtractionRepo.Update(ctx, row); err != nil {
				a.logger.Warn("failed to touch pending AI extraction poll time", zap.String("documentId", row.DocumentID.String()), zap.Error(err))
			}
			continue
		}

		if poll.Status == services.AIBackgroundExtractionStatusPending {
			if _, err = a.aiExtractionRepo.Update(ctx, row); err != nil {
				a.logger.Warn("failed to update pending AI extraction poll time", zap.String("documentId", row.DocumentID.String()), zap.Error(err))
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
		default:
			row.Status = documentaiextraction.StatusFailed
			row.FailureCode = poll.FailureCode
			row.FailureMessage = poll.FailureMessage
			result.Failed++
		}
		if _, err = a.aiExtractionRepo.Update(ctx, row); err != nil {
			a.logger.Warn("failed to update terminal AI extraction row", zap.String("documentId", row.DocumentID.String()), zap.Error(err))
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
		if a.aiExtractionRepo != nil {
			if row, repoErr := a.aiExtractionRepo.GetByDocumentExtractedAt(ctx, repositories.GetDocumentAIExtractionRequest{
				DocumentID:  payload.DocumentID,
				ExtractedAt: payload.ExtractedAt,
				TenantInfo:  tenantInfo,
			}); repoErr == nil {
				row.Status = documentaiextraction.StatusSkipped
				row.FailureCode = "stale_extraction"
				row.FailureMessage = "document extraction has been superseded"
				if _, repoErr = a.aiExtractionRepo.Update(ctx, row); repoErr != nil {
					a.logger.Warn("failed to mark stale AI extraction row skipped", zap.String("documentId", payload.DocumentID.String()), zap.Error(repoErr))
				}
			}
		}

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

	fallback := analysisFromStructuredData(content.StructuredData)
	diagnostics := aiExtractionDiagnostics{
		FallbackAnalysis: fallback,
		AcceptanceStatus: aiAcceptanceStatusRejected,
		RejectionReason:  payload.Completion.FailureCode,
		ResponseID:       payload.Completion.ResponseID,
		SubmittedAt:      payload.Completion.SubmittedAt,
		LastPolledAt:     payload.Completion.LastPolledAt,
	}

	updatedIntelligence := fallback
	if payload.Completion.Status == services.AIBackgroundExtractionStatusCompleted && payload.Completion.ExtractResult != nil {
		candidate := analysisFromAIExtract(payload.Completion.ExtractResult)
		diagnostics.CandidateAnalysis = &candidate
		merged, ok, rejectionReason := mergeAIAnalysis(fallback, payload.Completion.ExtractResult)
		if ok {
			diagnostics.AcceptanceStatus = aiAcceptanceStatusAccepted
			diagnostics.RejectionReason = ""
			updatedIntelligence = merged
		} else {
			diagnostics.AcceptanceStatus = aiAcceptanceStatusRejected
			diagnostics.RejectionReason = rejectionReason
		}
	}

	content.StructuredData = buildStructuredData(updatedIntelligence, diagnostics)
	content.ClassificationConfidence = updatedIntelligence.OverallConfidence
	if _, err = a.contentRepo.Upsert(ctx, content); err != nil {
		return nil, err
	}

	control, err := a.getDocumentControl(ctx, doc.OrganizationID, doc.BusinessUnitID)
	if err != nil {
		return nil, err
	}
	if err = a.updateShipmentDraftFromIntelligence(ctx, doc, control, updatedIntelligence); err != nil {
		return nil, err
	}

	indexedText := content.ContentText
	if !control.EnableFullTextIndexing {
		indexedText = ""
	}
	a.syncSearchProjection(ctx, doc, indexedText)

	if a.aiExtractionRepo != nil {
		if row, repoErr := a.aiExtractionRepo.GetByDocumentExtractedAt(ctx, repositories.GetDocumentAIExtractionRequest{
			DocumentID:  payload.DocumentID,
			ExtractedAt: payload.ExtractedAt,
			TenantInfo:  tenantInfo,
		}); repoErr == nil {
			row.Status = documentaiextraction.StatusApplied
			if _, repoErr = a.aiExtractionRepo.Update(ctx, row); repoErr != nil {
				a.logger.Warn("failed to mark AI extraction row applied", zap.String("documentId", payload.DocumentID.String()), zap.Error(repoErr))
			}
		}
	}

	return &ProcessDocumentAIExtractionResult{
		DocumentID:      payload.DocumentID,
		ExtractedAt:     payload.ExtractedAt,
		AcceptanceState: string(diagnostics.AcceptanceStatus),
	}, nil
}

func aiAcceptanceStatusFromStructuredData(structured map[string]any) string {
	if diagnostics, ok := structured["aiDiagnostics"].(map[string]any); ok {
		if status := stringValue(diagnostics["acceptanceStatus"]); status != "" {
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
			a.logger.Warn("pending AI extraction activity no longer exists", zap.String("documentId", row.DocumentID.String()), zap.Error(err))
			return true
		}
		a.logger.Warn("failed to complete pending AI extraction activity", zap.String("documentId", row.DocumentID.String()), zap.Error(err))
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
		sum.Write([]byte(fmt.Sprintf("\n[%d]\n", page.PageNumber)))
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
			Text:       truncateExtractedText(page.ExtractedText, pageLimit),
		})
	}
	return out
}

func markAIDiagnosticsPending(structured map[string]any, responseID string, submittedAt *int64) map[string]any {
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

func analysisFromStructuredData(structured map[string]any) documentIntelligenceAnalysis {
	if structured == nil {
		return documentIntelligenceAnalysis{
			Fields:    map[string]reviewField{},
			Stops:     []intelligenceStop{},
			Conflicts: []reviewConflict{},
		}
	}
	if aiDiagnostics, ok := structured["aiDiagnostics"].(map[string]any); ok {
		if fallback, ok := aiDiagnostics["fallbackAnalysis"].(map[string]any); ok {
			return analysisFromMap(fallback)
		}
	}
	if intelligence, ok := structured["intelligence"].(map[string]any); ok {
		return analysisFromMap(intelligence)
	}
	return documentIntelligenceAnalysis{
		Fields:    map[string]reviewField{},
		Stops:     []intelligenceStop{},
		Conflicts: []reviewConflict{},
	}
}

func analysisFromMap(data map[string]any) documentIntelligenceAnalysis {
	analysis := documentIntelligenceAnalysis{
		Kind:                 stringValue(data["kind"]),
		OverallConfidence:    floatValue(data["overallConfidence"]),
		ReviewStatus:         stringValue(data["reviewStatus"]),
		MissingFields:        stringSliceValue(data["missingFields"]),
		Signals:              stringSliceValue(data["signals"]),
		ClassifierSource:     stringValue(data["classifierSource"]),
		ProviderFingerprint:  stringValue(data["providerFingerprint"]),
		ClassificationReason: stringValue(data["classificationReason"]),
		RawExcerpt:           stringValue(data["rawExcerpt"]),
		Fields:               map[string]reviewField{},
		Stops:                []intelligenceStop{},
		Conflicts:            []reviewConflict{},
	}

	if metadata, ok := data["parsingRuleMetadata"].(map[string]any); ok {
		analysis.ParsingRuleMetadata = parseParsingRuleMetadata(metadata)
	}
	if fields, ok := data["fields"].(map[string]any); ok {
		for key, raw := range fields {
			fieldMap, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			analysis.Fields[key] = reviewField{
				Label:           stringValue(fieldMap["label"]),
				Value:           stringValue(fieldMap["value"]),
				Confidence:      floatValue(fieldMap["confidence"]),
				Excerpt:         stringValue(fieldMap["excerpt"]),
				EvidenceExcerpt: firstNonEmpty(stringValue(fieldMap["evidenceExcerpt"]), stringValue(fieldMap["excerpt"])),
				PageNumber:      intValue(fieldMap["pageNumber"]),
				ReviewRequired:  boolValue(fieldMap["reviewRequired"]),
				Conflict:        boolValue(fieldMap["conflict"]),
				Source:          stringValue(fieldMap["source"]),
			}
		}
	}
	if stops, ok := data["stops"].([]any); ok {
		for _, raw := range stops {
			stopMap, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			analysis.Stops = append(analysis.Stops, intelligenceStop{
				Sequence:            intValue(stopMap["sequence"]),
				Role:                stringValue(stopMap["role"]),
				Name:                stringValue(stopMap["name"]),
				AddressLine1:        stringValue(stopMap["addressLine1"]),
				AddressLine2:        stringValue(stopMap["addressLine2"]),
				City:                stringValue(stopMap["city"]),
				State:               stringValue(stopMap["state"]),
				PostalCode:          stringValue(stopMap["postalCode"]),
				Date:                stringValue(stopMap["date"]),
				TimeWindow:          stringValue(stopMap["timeWindow"]),
				AppointmentRequired: boolValue(stopMap["appointmentRequired"]),
				PageNumber:          intValue(stopMap["pageNumber"]),
				EvidenceExcerpt:     stringValue(stopMap["evidenceExcerpt"]),
				Confidence:          floatValue(stopMap["confidence"]),
				ReviewRequired:      boolValue(stopMap["reviewRequired"]),
				Source:              stringValue(stopMap["source"]),
			})
		}
	}
	if conflicts, ok := data["conflicts"].([]any); ok {
		for _, raw := range conflicts {
			conflictMap, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			analysis.Conflicts = append(analysis.Conflicts, reviewConflict{
				Key:             stringValue(conflictMap["key"]),
				Label:           stringValue(conflictMap["label"]),
				Values:          stringSliceValue(conflictMap["values"]),
				PageNumbers:     intSliceValue(conflictMap["pageNumbers"]),
				EvidenceExcerpt: stringValue(conflictMap["evidenceExcerpt"]),
				Source:          stringValue(conflictMap["source"]),
			})
		}
	}

	return analysis
}

func parseParsingRuleMetadata(data map[string]any) *services.DocumentParsingRuleMetadata {
	ruleSetID, err := pulid.Parse(stringValue(data["ruleSetId"]))
	if err != nil {
		return nil
	}
	ruleVersionID, err := pulid.Parse(stringValue(data["ruleVersionId"]))
	if err != nil {
		return nil
	}
	return &services.DocumentParsingRuleMetadata{
		RuleSetID:        ruleSetID,
		RuleSetName:      stringValue(data["ruleSetName"]),
		RuleVersionID:    ruleVersionID,
		VersionNumber:    intValue(data["versionNumber"]),
		ParserMode:       stringValue(data["parserMode"]),
		ProviderMatched:  stringValue(data["providerMatched"]),
		MatchSpecificity: intValue(data["matchSpecificity"]),
	}
}

func stringValue(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func floatValue(v any) float64 {
	switch value := v.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}

func intValue(v any) int {
	switch value := v.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func boolValue(v any) bool {
	value, _ := v.(bool)
	return value
}

func stringSliceValue(v any) []string {
	items, ok := v.([]any)
	if !ok {
		if direct, ok := v.([]string); ok {
			return append([]string{}, direct...)
		}
		return []string{}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := stringValue(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func intSliceValue(v any) []int {
	items, ok := v.([]any)
	if !ok {
		if direct, ok := v.([]int); ok {
			return append([]int{}, direct...)
		}
		return []int{}
	}
	out := make([]int, 0, len(items))
	for _, item := range items {
		out = append(out, intValue(item))
	}
	return out
}

func (a *Activities) updateShipmentDraftFromIntelligence(
	ctx context.Context,
	doc *document.Document,
	control *tenant.DocumentControl,
	intelligence documentIntelligenceAnalysis,
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
		ID:                  doc.ID,
		TenantInfo:          pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID},
		ContentStatus:       doc.ContentStatus,
		ContentError:        doc.ContentError,
		DetectedKind:        doc.DetectedKind,
		HasExtractedText:    doc.HasExtractedText,
		ShipmentDraftStatus: doc.ShipmentDraftStatus,
		DocumentTypeID:      doc.DocumentTypeID,
	})
}
