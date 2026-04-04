package openaidocumentservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json" //nolint:depguard // external API payloads
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

type responsesRequest struct {
	Model           string              `json:"model"`
	Input           []responsesMessage  `json:"input"`
	Text            responsesTextConfig `json:"text"`
	MaxOutputTokens int                 `json:"max_output_tokens,omitempty"`
	Background      bool                `json:"background,omitempty"`
	Store           bool                `json:"store,omitempty"`
}

type responsesMessage struct {
	Role    string                 `json:"role"`
	Content []responsesMessagePart `json:"content"`
}

type responsesMessagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesTextConfig struct {
	Format responsesFormat `json:"format"`
}

type responsesFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

type responsesEnvelope struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Model      string `json:"model"`
	OutputText string `json:"output_text"`
	Output     []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Usage struct {
		InputTokens         int `json:"input_tokens"`
		OutputTokens        int `json:"output_tokens"`
		TotalTokens         int `json:"total_tokens"`
		OutputTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"output_tokens_details"`
	} `json:"usage"`
	ServiceTier string `json:"service_tier"`
	Error       *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
}

type routeResponse struct {
	ShouldExtract       bool     `json:"shouldExtract"`
	DocumentKind        string   `json:"documentKind"`
	Confidence          float64  `json:"confidence"`
	Signals             []string `json:"signals"`
	ReviewStatus        string   `json:"reviewStatus"`
	ClassifierSource    string   `json:"classifierSource"`
	ProviderFingerprint string   `json:"providerFingerprint"`
	Reason              string   `json:"reason"`
}

type extractFieldResponse struct {
	Key               string   `json:"key"`
	Label             string   `json:"label"`
	Value             string   `json:"value"`
	Confidence        float64  `json:"confidence"`
	EvidenceExcerpt   string   `json:"evidenceExcerpt"`
	PageNumber        int      `json:"pageNumber"`
	ReviewRequired    bool     `json:"reviewRequired"`
	Conflict          bool     `json:"conflict"`
	Source            string   `json:"source"`
	AlternativeValues []string `json:"alternativeValues"`
}

type extractResponse struct {
	DocumentKind      string                            `json:"documentKind"`
	OverallConfidence float64                           `json:"overallConfidence"`
	ReviewStatus      string                            `json:"reviewStatus"`
	MissingFields     []string                          `json:"missingFields"`
	Signals           []string                          `json:"signals"`
	Fields            []extractFieldResponse            `json:"fields"`
	Stops             []serviceports.AIDocumentStop     `json:"stops"`
	Conflicts         []serviceports.AIDocumentConflict `json:"conflicts"`
}

func (s *Service) RouteDocument(
	ctx context.Context,
	req *serviceports.AIRouteRequest,
) (*serviceports.AIRouteResult, error) {
	if !s.cfg.AIEnabled() {
		return nil, errortypes.NewBusinessError("AI document intelligence is disabled")
	}

	systemPrompt := "You classify transportation documents. Return strict JSON only."
	userPrompt := buildRoutePrompt(req)
	var parsed routeResponse
	envelope, err := s.executeStructuredResponse(
		ctx,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
		req.TenantInfo.UserID,
		req.DocumentID,
		ailog.OperationDocumentIntelligenceRoute,
		ailog.Model(s.cfg.GetAIClassificationModel()),
		systemPrompt,
		userPrompt,
		buildRouteSchema(),
		&parsed,
	)
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

	systemPrompt := "You extract structured rate confirmation data for a transportation management system. Return strict JSON only."
	userPrompt := buildExtractPrompt(req)
	parsed := new(extractResponse)
	envelope, err := s.executeStructuredResponse(
		ctx,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
		req.TenantInfo.UserID,
		req.DocumentID,
		ailog.OperationDocumentIntelligenceExtract,
		ailog.Model(s.cfg.GetAIExtractionModel()),
		systemPrompt,
		userPrompt,
		buildExtractSchema(),
		parsed,
	)
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

	systemPrompt := "You extract structured rate confirmation data for a transportation management system. Return strict JSON only."
	userPrompt := buildExtractPrompt(req)
	requestBody := responsesRequest{
		Model: string(ailog.Model(s.cfg.GetAIExtractionModel())),
		Input: []responsesMessage{
			{
				Role: "system",
				Content: []responsesMessagePart{{
					Type: "input_text",
					Text: systemPrompt,
				}},
			},
			{
				Role: "user",
				Content: []responsesMessagePart{{
					Type: "input_text",
					Text: userPrompt,
				}},
			},
		},
		Text: responsesTextConfig{
			Format: responsesFormat{
				Type:   "json_schema",
				Name:   string(ailog.OperationDocumentIntelligenceExtract),
				Schema: buildExtractSchema(),
				Strict: true,
			},
		},
		MaxOutputTokens: s.cfg.GetAIExtractionMaxTokens(),
		Background:      true,
		Store:           true,
	}

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

	envelope, err := s.doResponsesRequest(ctx, runtimeCfg.Config["apiKey"], http.MethodPost, responsesURL, body)
	if err != nil {
		s.recordAIUsage("extract_background_submit", false, "error")
		return nil, err
	}
	if strings.TrimSpace(envelope.ID) == "" {
		s.recordAIUsage("extract_background_submit", false, "empty_response_id")
		return nil, errortypes.NewBusinessError("AI background extraction response did not include an ID")
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
			return &serviceports.AIBackgroundExtractPollResult{
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

func (s *Service) executeStructuredResponse(
	ctx context.Context,
	orgID, buID, userID, documentID pulid.ID,
	operation ailog.Operation,
	model ailog.Model,
	systemPrompt, userPrompt string,
	schema map[string]any,
	out any,
) (*responsesEnvelope, error) {
	runtimeCfg, err := s.integration.GetRuntimeConfig(ctx, pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	}, integration.TypeOpenAI)
	if err != nil {
		s.recordAIUsage(string(operation), false, "missing_config")
		return nil, err
	}

	requestBody := responsesRequest{
		Model: string(model),
		Input: []responsesMessage{
			{
				Role: "system",
				Content: []responsesMessagePart{{
					Type: "input_text",
					Text: systemPrompt,
				}},
			},
			{
				Role: "user",
				Content: []responsesMessagePart{{
					Type: "input_text",
					Text: userPrompt,
				}},
			},
		},
		Text: responsesTextConfig{
			Format: responsesFormat{
				Type:   "json_schema",
				Name:   string(operation),
				Schema: schema,
				Strict: true,
			},
		},
		MaxOutputTokens: s.cfg.GetAIExtractionMaxTokens(),
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= s.cfg.GetAIMaxRetries(); attempt++ {
		envelope, execErr := s.executeOnce(
			ctx,
			runtimeCfg.Config["apiKey"],
			body,
			orgID,
			buID,
			userID,
			documentID,
			operation,
			model,
			systemPrompt,
			userPrompt,
			out,
		)
		if execErr == nil {
			return envelope, nil
		}
		lastErr = execErr
		if attempt == s.cfg.GetAIMaxRetries() || !isRetryableAIError(execErr) {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 300 * time.Millisecond)
	}

	s.recordAIUsage(string(operation), false, "error")
	return nil, lastErr
}

func (s *Service) executeOnce(
	ctx context.Context,
	apiKey string,
	body []byte,
	orgID, buID, userID, documentID pulid.ID,
	operation ailog.Operation,
	model ailog.Model,
	systemPrompt, userPrompt string,
	out any,
) (*responsesEnvelope, error) {
	envelope, err := s.doResponsesRequest(ctx, apiKey, http.MethodPost, responsesURL, body)
	if err != nil {
		return nil, err
	}

	text := extractResponseText(envelope)
	if text == "" {
		return nil, errortypes.NewBusinessError("AI response did not contain structured output")
	}
	if err = json.Unmarshal([]byte(text), out); err != nil {
		return nil, err
	}

	s.logAIInteraction(ctx, &ailog.Log{
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		UserID:           userID,
		Prompt:           redactPrompt(systemPrompt, userPrompt),
		Response:         redactResponse(text),
		Model:            model,
		Operation:        operation,
		Object:           documentID.String(),
		ServiceTier:      envelope.ServiceTier,
		PromptTokens:     envelope.Usage.InputTokens,
		CompletionTokens: envelope.Usage.OutputTokens,
		TotalTokens:      envelope.Usage.TotalTokens,
		ReasoningTokens:  envelope.Usage.OutputTokensDetails.ReasoningTokens,
	})

	return envelope, nil
}

func (s *Service) doResponsesRequest(
	ctx context.Context,
	apiKey, method, url string,
	body []byte,
) (*responsesEnvelope, error) {
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai responses api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	envelope := new(responsesEnvelope)
	if err = json.Unmarshal(respBody, envelope); err != nil {
		return nil, err
	}

	return envelope, nil
}

func (s *Service) logAIInteraction(ctx context.Context, entry *ailog.Log) {
	if _, err := s.aiLogRepo.Create(ctx, entry); err != nil {
		s.logger.Warn("failed to persist ai log", zap.Error(err))
	}
}

func (s *Service) recordAIUsage(operation string, success bool, outcome string) {
	s.metrics.Document.RecordAIOutcome(operation, success, outcome)
}

func buildRoutePrompt(req *serviceports.AIRouteRequest) string {
	var b strings.Builder
	b.WriteString("Classify this transportation document into one of: RateConfirmation, BillOfLading, ProofOfDelivery, Other.\n")
	b.WriteString("Use the extracted text, feature summary, and any provider fingerprint hint. Return strict JSON only.\n")
	b.WriteString("Set shouldExtract=true only when the documentKind is RateConfirmation and the evidence is strong enough for structured extraction.\n")
	b.WriteString("Filename: " + strings.TrimSpace(req.FileName) + "\n")
	if req.Fingerprint != nil {
		b.WriteString(fmt.Sprintf("Provider fingerprint hint: provider=%s kindHint=%s confidence=%.2f signals=%s\n",
			req.Fingerprint.Provider,
			req.Fingerprint.KindHint,
			req.Fingerprint.Confidence,
			strings.Join(req.Fingerprint.Signals, ", "),
		))
	}
	if req.Features != nil {
		b.WriteString("Normalized features:\n")
		b.WriteString(fmt.Sprintf("Titles: %s\n", strings.Join(req.Features.TitleCandidates, " | ")))
		b.WriteString(fmt.Sprintf("Section labels: %s\n", strings.Join(req.Features.SectionLabels, " | ")))
		b.WriteString(fmt.Sprintf("Party labels: %s\n", strings.Join(req.Features.PartyLabels, " | ")))
		b.WriteString(fmt.Sprintf("Reference labels: %s\n", strings.Join(req.Features.ReferenceLabels, " | ")))
		b.WriteString(fmt.Sprintf("Money signals: %s\n", strings.Join(req.Features.MoneySignals, " | ")))
		b.WriteString(fmt.Sprintf("Stop signals: %s\n", strings.Join(req.Features.StopSignals, " | ")))
		b.WriteString(fmt.Sprintf("Terms signals: %s\n", strings.Join(req.Features.TermsSignals, " | ")))
		b.WriteString(fmt.Sprintf("Signature signals: %s\n", strings.Join(req.Features.SignatureSignals, " | ")))
	}
	b.WriteString("Document text excerpt:\n")
	b.WriteString(truncateForAI(req.Text, 4000))
	b.WriteString("\nPage summaries:\n")
	for _, page := range req.Pages {
		b.WriteString(fmt.Sprintf("Page %d: %s\n", page.PageNumber, truncateForAI(page.Text, 800)))
	}
	return b.String()
}

func buildExtractPrompt(req *serviceports.AIExtractRequest) string {
	var b strings.Builder
	b.WriteString("Extract structured rate confirmation data for a TMS.\n")
	b.WriteString("Return only compact canonical fields and stop data needed for shipment creation/review.\n")
	b.WriteString("Do not emit extra broker-specific or descriptive fields beyond the canonical key set.\n")
	b.WriteString("Use page-local evidence. Keep evidence excerpts short and specific. Mark conflicts and low-confidence fields instead of guessing.\n")
	b.WriteString("Canonical field keys: loadNumber, referenceNumber, shipper, consignee, rate, equipmentType, commodity, pickupDate, deliveryDate, pickupWindow, deliveryWindow, pickupNumber, deliveryNumber, appointmentNumber, bol, poNumber, scac, proNumber, paymentTerms, billTo, carrierName, carrierContact, containerNumber, trailerNumber, tractorNumber, fuelSurcharge, serviceType.\n")
	b.WriteString("Filename: " + strings.TrimSpace(req.FileName) + "\n")
	for _, page := range req.Pages {
		b.WriteString(fmt.Sprintf("\n[Page %d]\n%s\n", page.PageNumber, truncateForAI(page.Text, 2500)))
	}
	return b.String()
}

func buildRouteSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"shouldExtract": map[string]any{"type": "boolean"},
			"documentKind":  map[string]any{"type": "string"},
			"confidence":    map[string]any{"type": "number"},
			"signals": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
			"reviewStatus":        map[string]any{"type": "string"},
			"classifierSource":    map[string]any{"type": "string"},
			"providerFingerprint": map[string]any{"type": "string"},
			"reason":              map[string]any{"type": "string"},
		},
		"required": []string{"shouldExtract", "documentKind", "confidence", "signals", "reviewStatus", "classifierSource", "providerFingerprint", "reason"},
	}
}

func buildExtractSchema() map[string]any {
	fieldSchema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"key": map[string]any{
				"type": "string",
				"enum": []string{
					"loadNumber", "referenceNumber", "shipper", "consignee", "rate", "equipmentType", "commodity",
					"pickupDate", "deliveryDate", "pickupWindow", "deliveryWindow", "pickupNumber", "deliveryNumber",
					"appointmentNumber", "bol", "poNumber", "scac", "proNumber", "paymentTerms", "billTo",
					"carrierName", "carrierContact", "containerNumber", "trailerNumber", "tractorNumber",
					"fuelSurcharge", "serviceType",
				},
			},
			"label":           map[string]any{"type": "string", "maxLength": 64},
			"value":           map[string]any{"type": "string", "maxLength": 256},
			"confidence":      map[string]any{"type": "number"},
			"evidenceExcerpt": map[string]any{"type": "string", "maxLength": 200},
			"pageNumber":      map[string]any{"type": "integer"},
			"reviewRequired":  map[string]any{"type": "boolean"},
			"conflict":        map[string]any{"type": "boolean"},
			"source":          map[string]any{"type": "string", "maxLength": 32},
			"alternativeValues": map[string]any{
				"type":     "array",
				"maxItems": 4,
				"items":    map[string]any{"type": "string", "maxLength": 128},
			},
		},
		"required": []string{"key", "label", "value", "confidence", "evidenceExcerpt", "pageNumber", "reviewRequired", "conflict", "source", "alternativeValues"},
	}
	stopSchema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"sequence":            map[string]any{"type": "integer"},
			"role":                map[string]any{"type": "string"},
			"name":                map[string]any{"type": "string", "maxLength": 128},
			"addressLine1":        map[string]any{"type": "string", "maxLength": 160},
			"addressLine2":        map[string]any{"type": "string", "maxLength": 160},
			"city":                map[string]any{"type": "string", "maxLength": 80},
			"state":               map[string]any{"type": "string", "maxLength": 16},
			"postalCode":          map[string]any{"type": "string", "maxLength": 20},
			"date":                map[string]any{"type": "string", "maxLength": 40},
			"timeWindow":          map[string]any{"type": "string", "maxLength": 64},
			"appointmentRequired": map[string]any{"type": "boolean"},
			"pageNumber":          map[string]any{"type": "integer"},
			"evidenceExcerpt":     map[string]any{"type": "string", "maxLength": 200},
			"confidence":          map[string]any{"type": "number"},
			"reviewRequired":      map[string]any{"type": "boolean"},
			"source":              map[string]any{"type": "string", "maxLength": 32},
		},
		"required": []string{"sequence", "role", "name", "addressLine1", "addressLine2", "city", "state", "postalCode", "date", "timeWindow", "appointmentRequired", "pageNumber", "evidenceExcerpt", "confidence", "reviewRequired", "source"},
	}
	conflictSchema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"key":             map[string]any{"type": "string", "maxLength": 64},
			"label":           map[string]any{"type": "string", "maxLength": 64},
			"values":          map[string]any{"type": "array", "maxItems": 4, "items": map[string]any{"type": "string", "maxLength": 128}},
			"pageNumbers":     map[string]any{"type": "array", "maxItems": 6, "items": map[string]any{"type": "integer"}},
			"evidenceExcerpt": map[string]any{"type": "string", "maxLength": 200},
			"source":          map[string]any{"type": "string", "maxLength": 32},
		},
		"required": []string{"key", "label", "values", "pageNumbers", "evidenceExcerpt", "source"},
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"documentKind":      map[string]any{"type": "string"},
			"overallConfidence": map[string]any{"type": "number"},
			"reviewStatus":      map[string]any{"type": "string"},
			"missingFields":     map[string]any{"type": "array", "maxItems": 12, "items": map[string]any{"type": "string", "maxLength": 64}},
			"signals":           map[string]any{"type": "array", "maxItems": 8, "items": map[string]any{"type": "string", "maxLength": 120}},
			"fields":            map[string]any{"type": "array", "maxItems": 18, "items": fieldSchema},
			"stops":             map[string]any{"type": "array", "maxItems": 8, "items": stopSchema},
			"conflicts":         map[string]any{"type": "array", "maxItems": 6, "items": conflictSchema},
		},
		"required": []string{"documentKind", "overallConfidence", "reviewStatus", "missingFields", "signals", "fields", "stops", "conflicts"},
	}
}

func truncateForAI(text string, max int) string {
	text = strings.TrimSpace(text)
	if max <= 0 || len(text) <= max {
		return text
	}
	return text[:max]
}

func convertExtractResponse(parsed *extractResponse) *serviceports.AIExtractResult {
	result := &serviceports.AIExtractResult{
		DocumentKind:      "",
		OverallConfidence: 0,
		ReviewStatus:      "",
		MissingFields:     []string{},
		Signals:           []string{},
		Fields:            map[string]serviceports.AIDocumentField{},
		Stops:             []serviceports.AIDocumentStop{},
		Conflicts:         []serviceports.AIDocumentConflict{},
	}
	if parsed == nil {
		return result
	}

	result.DocumentKind = parsed.DocumentKind
	result.OverallConfidence = parsed.OverallConfidence
	result.ReviewStatus = parsed.ReviewStatus
	result.MissingFields = parsed.MissingFields
	result.Signals = parsed.Signals
	result.Stops = parsed.Stops
	result.Conflicts = parsed.Conflicts
	for _, field := range parsed.Fields {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			continue
		}
		result.Fields[key] = serviceports.AIDocumentField{
			Label:             field.Label,
			Value:             field.Value,
			Confidence:        field.Confidence,
			EvidenceExcerpt:   field.EvidenceExcerpt,
			PageNumber:        field.PageNumber,
			ReviewRequired:    field.ReviewRequired,
			Conflict:          field.Conflict,
			Source:            field.Source,
			AlternativeValues: field.AlternativeValues,
		}
	}

	return result
}

func extractResponseText(envelope *responsesEnvelope) string {
	if envelope == nil {
		return ""
	}
	if strings.TrimSpace(envelope.OutputText) != "" {
		return envelope.OutputText
	}
	for _, output := range envelope.Output {
		for _, content := range output.Content {
			if strings.TrimSpace(content.Text) != "" {
				return content.Text
			}
		}
	}
	return ""
}

func errorCode(envelope *responsesEnvelope) string {
	if envelope == nil || envelope.Error == nil {
		return ""
	}
	return strings.TrimSpace(envelope.Error.Code)
}

func errorMessage(envelope *responsesEnvelope) string {
	if envelope == nil || envelope.Error == nil {
		return ""
	}
	return strings.TrimSpace(envelope.Error.Message)
}

func responseIncompleteReason(envelope *responsesEnvelope) string {
	if envelope == nil || envelope.IncompleteDetails == nil {
		return ""
	}
	return strings.TrimSpace(envelope.IncompleteDetails.Reason)
}

func incompleteFailureMessage(status, reason string) string {
	if strings.TrimSpace(reason) == "" {
		return ""
	}

	if strings.EqualFold(status, "incomplete") {
		return fmt.Sprintf("AI background extraction ended incomplete: %s", reason)
	}

	return fmt.Sprintf("AI background extraction ended with status %s: %s", status, reason)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func redactPrompt(systemPrompt, userPrompt string) string {
	sum := sha256.Sum256([]byte(userPrompt))
	return fmt.Sprintf("system=%q user_sha256=%s user_preview=%q", systemPrompt, hex.EncodeToString(sum[:]), truncateForAI(userPrompt, 512))
}

func redactResponse(text string) string {
	sum := sha256.Sum256([]byte(text))
	return fmt.Sprintf("sha256=%s preview=%q", hex.EncodeToString(sum[:]), truncateForAI(text, 1024))
}

func normalizeReviewStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "ready":
		return "Ready"
	case "needsreview", "needs_review":
		return "NeedsReview"
	case "unavailable":
		return "Unavailable"
	default:
		return "NeedsReview"
	}
}

func clampAIConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func isRetryableAIError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "429") ||
		strings.Contains(msg, "500") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "504") ||
		strings.Contains(msg, "timeout")
}
