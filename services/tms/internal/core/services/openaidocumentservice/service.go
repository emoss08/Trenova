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
	result := new(serviceports.AIExtractResult)
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
		result,
	)
	if err != nil {
		return nil, err
	}

	result.DocumentKind = strings.TrimSpace(result.DocumentKind)
	result.ReviewStatus = normalizeReviewStatus(result.ReviewStatus)
	result.OverallConfidence = clampAIConfidence(result.OverallConfidence)
	s.recordAIUsage("extract", envelope != nil, "success")
	return result, nil
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
		MaxOutputTokens: 2500,
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, responsesURL, bytes.NewReader(body))
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
	b.WriteString("Classify this transportation document into one of: RateConfirmation, BillOfLading, ProofOfDelivery, Invoice, Other.\n")
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
	b.WriteString("Extract structured rate confirmation data. Use page-local evidence. Mark conflicts and low-confidence fields instead of guessing.\n")
	b.WriteString("Filename: " + strings.TrimSpace(req.FileName) + "\n")
	for _, page := range req.Pages {
		b.WriteString(fmt.Sprintf("\n[Page %d]\n%s\n", page.PageNumber, truncateForAI(page.Text, 3500)))
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
			"label":           map[string]any{"type": "string"},
			"value":           map[string]any{"type": "string"},
			"confidence":      map[string]any{"type": "number"},
			"evidenceExcerpt": map[string]any{"type": "string"},
			"pageNumber":      map[string]any{"type": "integer"},
			"reviewRequired":  map[string]any{"type": "boolean"},
			"conflict":        map[string]any{"type": "boolean"},
			"source":          map[string]any{"type": "string"},
			"alternativeValues": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
		},
		"required": []string{"label", "value", "confidence", "evidenceExcerpt", "pageNumber", "reviewRequired", "conflict", "source", "alternativeValues"},
	}
	stopSchema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"sequence":            map[string]any{"type": "integer"},
			"role":                map[string]any{"type": "string"},
			"name":                map[string]any{"type": "string"},
			"addressLine1":        map[string]any{"type": "string"},
			"addressLine2":        map[string]any{"type": "string"},
			"city":                map[string]any{"type": "string"},
			"state":               map[string]any{"type": "string"},
			"postalCode":          map[string]any{"type": "string"},
			"date":                map[string]any{"type": "string"},
			"timeWindow":          map[string]any{"type": "string"},
			"appointmentRequired": map[string]any{"type": "boolean"},
			"pageNumber":          map[string]any{"type": "integer"},
			"evidenceExcerpt":     map[string]any{"type": "string"},
			"confidence":          map[string]any{"type": "number"},
			"reviewRequired":      map[string]any{"type": "boolean"},
			"source":              map[string]any{"type": "string"},
		},
		"required": []string{"sequence", "role", "name", "addressLine1", "addressLine2", "city", "state", "postalCode", "date", "timeWindow", "appointmentRequired", "pageNumber", "evidenceExcerpt", "confidence", "reviewRequired", "source"},
	}
	conflictSchema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"key":             map[string]any{"type": "string"},
			"label":           map[string]any{"type": "string"},
			"values":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"pageNumbers":     map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
			"evidenceExcerpt": map[string]any{"type": "string"},
			"source":          map[string]any{"type": "string"},
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
			"missingFields":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"signals":           map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"fields": map[string]any{
				"type":                 "object",
				"additionalProperties": fieldSchema,
			},
			"stops":     map[string]any{"type": "array", "items": stopSchema},
			"conflicts": map[string]any{"type": "array", "items": conflictSchema},
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
