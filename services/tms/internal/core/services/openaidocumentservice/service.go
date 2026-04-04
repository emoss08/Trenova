package openaidocumentservice

import (
	"context"
	"encoding/json" //nolint:depguard // external API payloads
	"fmt"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const responsesURL = "https://api.openai.com/v1/responses"

type Params struct {
	fx.In

	Logger      *zap.Logger
	Config      *config.Config
	Metrics     *metrics.Registry
	Integration *integrationservice.Service
	AILogRepo   repositories.AILogRepository
}

type Service struct {
	logger      *zap.Logger
	cfg         *config.DocumentIntelligenceConfig
	metrics     *metrics.Registry
	integration *integrationservice.Service
	aiLogRepo   repositories.AILogRepository
	httpClient  *http.Client
}

func New(p Params) serviceports.AIDocumentService {
	cfg := p.Config.GetDocumentIntelligenceConfig()
	return &Service{
		logger:      p.Logger.Named("service.openai-document"),
		cfg:         cfg,
		metrics:     p.Metrics,
		integration: p.Integration,
		aiLogRepo:   p.AILogRepo,
		httpClient: &http.Client{
			Timeout: cfg.GetAITimeout(),
		},
	}
}

func (s *Service) RouteDocument(
	ctx context.Context,
	req *serviceports.AIRouteRequest,
) (*serviceports.AIRouteResult, error) {
	if !s.cfg.AIEnabled() {
		return nil, errortypes.NewBusinessError("AI document intelligence is disabled")
	}

	var parsed routeResponse
	envelope, err := s.executeStructuredResponse(ctx, &structuredResponseParams{
		orgID:        req.TenantInfo.OrgID,
		buID:         req.TenantInfo.BuID,
		userID:       req.TenantInfo.UserID,
		documentID:   req.DocumentID,
		operation:    ailog.OperationDocumentIntelligenceRoute,
		model:        ailog.Model(s.cfg.GetAIClassificationModel()),
		systemPrompt: "You classify transportation documents. Return strict JSON only.",
		userPrompt:   buildRoutePrompt(req),
		schema:       buildRouteSchema(),
		out:          &parsed,
	})
	if err != nil {
		return nil, err
	}

	result := &serviceports.AIRouteResult{
		ShouldExtract:       parsed.ShouldExtract,
		DocumentKind:        strings.TrimSpace(parsed.DocumentKind),
		Confidence:          clampAIConfidence(parsed.Confidence),
		Signals:             parsed.Signals,
		ReviewStatus:        normalizeReviewStatus(parsed.ReviewStatus),
		ClassifierSource:    strings.TrimSpace(parsed.ClassifierSource),
		ProviderFingerprint: strings.TrimSpace(parsed.ProviderFingerprint),
		Reason:              strings.TrimSpace(parsed.Reason),
	}
	s.recordAIUsage("route", envelope != nil, "success")
	return result, nil
}

func (s *Service) ExtractRateConfirmation(
	ctx context.Context,
	req *serviceports.AIExtractRequest,
) (*serviceports.AIExtractResult, error) {
	if !s.cfg.AIEnabled() {
		return nil, errortypes.NewBusinessError("AI document intelligence is disabled")
	}

	parsed := new(extractResponse)
	envelope, err := s.executeStructuredResponse(ctx, &structuredResponseParams{
		orgID:        req.TenantInfo.OrgID,
		buID:         req.TenantInfo.BuID,
		userID:       req.TenantInfo.UserID,
		documentID:   req.DocumentID,
		operation:    ailog.OperationDocumentIntelligenceExtract,
		model:        ailog.Model(s.cfg.GetAIExtractionModel()),
		systemPrompt: "You extract structured rate confirmation data for a transportation management system. Return strict JSON only.",
		userPrompt:   buildExtractPrompt(req),
		schema:       buildExtractSchema(),
		out:          parsed,
	})
	if err != nil {
		return nil, err
	}

	result := convertExtractResponse(parsed)
	result.DocumentKind = strings.TrimSpace(result.DocumentKind)
	result.ReviewStatus = normalizeReviewStatus(result.ReviewStatus)
	result.OverallConfidence = clampAIConfidence(result.OverallConfidence)
	s.recordAIUsage("extract", envelope != nil, "success")
	return result, nil
}

func (s *Service) SubmitRateConfirmationBackgroundExtraction(
	ctx context.Context,
	req *serviceports.AIExtractRequest,
) (*serviceports.AIBackgroundExtractSubmission, error) {
	if !s.cfg.AIEnabled() {
		return nil, errortypes.NewBusinessError("AI document intelligence is disabled")
	}

	p := &structuredResponseParams{
		orgID:        req.TenantInfo.OrgID,
		buID:         req.TenantInfo.BuID,
		userID:       req.TenantInfo.UserID,
		documentID:   req.DocumentID,
		operation:    ailog.OperationDocumentIntelligenceExtract,
		model:        ailog.Model(s.cfg.GetAIExtractionModel()),
		systemPrompt: "You extract structured rate confirmation data for a transportation management system. Return strict JSON only.",
		userPrompt:   buildExtractPrompt(req),
		schema:       buildExtractSchema(),
	}

	requestBody := s.buildResponsesRequest(p)
	requestBody.Background = true
	requestBody.Store = true

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	runtimeCfg, err := s.integration.GetRuntimeConfig(ctx, pagination.TenantInfo{
		OrgID: req.TenantInfo.OrgID,
		BuID:  req.TenantInfo.BuID,
	}, integration.TypeOpenAI)
	if err != nil {
		s.recordAIUsage("extract_background_submit", false, "missing_config")
		return nil, err
	}

	envelope, err := s.doResponsesRequest(
		ctx,
		runtimeCfg.Config["apiKey"],
		http.MethodPost,
		responsesURL,
		body,
	)
	if err != nil {
		s.recordAIUsage("extract_background_submit", false, "error")
		return nil, err
	}
	if strings.TrimSpace(envelope.ID) == "" {
		s.recordAIUsage("extract_background_submit", false, "empty_response_id")
		return nil, errortypes.NewBusinessError(
			"AI background extraction response did not include an ID",
		)
	}

	s.recordAIUsage("extract_background_submit", true, "success")
	return &serviceports.AIBackgroundExtractSubmission{
		ResponseID: envelope.ID,
		Model:      firstNonEmpty(envelope.Model, s.cfg.GetAIExtractionModel()),
		Status:     strings.TrimSpace(envelope.Status),
	}, nil
}

func (s *Service) PollRateConfirmationBackgroundExtraction(
	ctx context.Context,
	req *serviceports.AIBackgroundExtractPollRequest,
) (*serviceports.AIBackgroundExtractPollResult, error) {
	if !s.cfg.AIEnabled() {
		return nil, errortypes.NewBusinessError("AI document intelligence is disabled")
	}

	runtimeCfg, err := s.integration.GetRuntimeConfig(ctx, pagination.TenantInfo{
		OrgID: req.TenantInfo.OrgID,
		BuID:  req.TenantInfo.BuID,
	}, integration.TypeOpenAI)
	if err != nil {
		s.recordAIUsage("extract_background_poll", false, "missing_config")
		return nil, err
	}

	envelope, err := s.doResponsesRequest(
		ctx,
		runtimeCfg.Config["apiKey"],
		http.MethodGet,
		fmt.Sprintf("%s/%s", responsesURL, strings.TrimSpace(req.ResponseID)),
		nil,
	)
	if err != nil {
		s.recordAIUsage("extract_background_poll", false, "error")
		return nil, err
	}

	rawStatus := strings.TrimSpace(envelope.Status)
	switch rawStatus {
	case "", "queued", "in_progress":
		s.recordAIUsage("extract_background_poll", true, "pending")
		return &serviceports.AIBackgroundExtractPollResult{
			ResponseID: req.ResponseID,
			Model:      firstNonEmpty(envelope.Model, s.cfg.GetAIExtractionModel()),
			Status:     serviceports.AIBackgroundExtractionStatusPending,
			RawStatus:  rawStatus,
		}, nil
	case "completed":
		parsed := new(extractResponse)
		text := extractResponseText(envelope)
		if text == "" {
			s.recordAIUsage("extract_background_poll", false, "empty_output")
			return &serviceports.AIBackgroundExtractPollResult{
				ResponseID:     req.ResponseID,
				Model:          firstNonEmpty(envelope.Model, s.cfg.GetAIExtractionModel()),
				Status:         serviceports.AIBackgroundExtractionStatusFailed,
				RawStatus:      rawStatus,
				FailureCode:    "empty_output",
				FailureMessage: "AI background extraction completed without structured output",
			}, nil
		}
		if err = json.Unmarshal([]byte(text), parsed); err != nil {
			s.recordAIUsage("extract_background_poll", false, "invalid_output")
			return &serviceports.AIBackgroundExtractPollResult{ //nolint:nilerr // err captured in FailureMessage
				ResponseID:     req.ResponseID,
				Model:          firstNonEmpty(envelope.Model, s.cfg.GetAIExtractionModel()),
				Status:         serviceports.AIBackgroundExtractionStatusFailed,
				RawStatus:      rawStatus,
				FailureCode:    "invalid_output",
				FailureMessage: err.Error(),
			}, nil
		}
		s.recordAIUsage("extract_background_poll", true, "completed")
		return &serviceports.AIBackgroundExtractPollResult{
			ResponseID:    req.ResponseID,
			Model:         firstNonEmpty(envelope.Model, s.cfg.GetAIExtractionModel()),
			Status:        serviceports.AIBackgroundExtractionStatusCompleted,
			RawStatus:     rawStatus,
			ExtractResult: convertExtractResponse(parsed),
		}, nil
	default:
		incompleteReason := responseIncompleteReason(envelope)
		failureCode := firstNonEmpty(incompleteReason, errorCode(envelope), rawStatus)
		failureMessage := firstNonEmpty(
			errorMessage(envelope),
			incompleteFailureMessage(rawStatus, incompleteReason),
			fmt.Sprintf("AI background extraction ended with status %s", rawStatus),
		)
		s.recordAIUsage("extract_background_poll", true, "terminal_failure")
		return &serviceports.AIBackgroundExtractPollResult{
			ResponseID:     req.ResponseID,
			Model:          firstNonEmpty(envelope.Model, s.cfg.GetAIExtractionModel()),
			Status:         serviceports.AIBackgroundExtractionStatusFailed,
			RawStatus:      rawStatus,
			FailureCode:    failureCode,
			FailureMessage: failureMessage,
		}, nil
	}
}
