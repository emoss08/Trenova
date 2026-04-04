package openaidocumentservice

import (
	"bytes"
	"context"
	"encoding/json" //nolint:depguard // external API payloads
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type apiError struct {
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	return fmt.Sprintf(
		"openai responses api returned %d: %s",
		e.StatusCode, strings.TrimSpace(e.Body),
	)
}

func (s *Service) buildResponsesRequest(p *structuredResponseParams) responsesRequest {
	return responsesRequest{
		Model: string(p.model),
		Input: []responsesMessage{
			{
				Role: "system",
				Content: []responsesMessagePart{{
					Type: "input_text",
					Text: p.systemPrompt,
				}},
			},
			{
				Role: "user",
				Content: []responsesMessagePart{{
					Type: "input_text",
					Text: p.userPrompt,
				}},
			},
		},
		Text: responsesTextConfig{
			Format: responsesFormat{
				Type:   "json_schema",
				Name:   string(p.operation),
				Schema: p.schema,
				Strict: true,
			},
		},
		MaxOutputTokens: s.cfg.GetAIExtractionMaxTokens(),
	}
}

func (s *Service) executeStructuredResponse(
	ctx context.Context,
	p *structuredResponseParams,
) (*responsesEnvelope, error) {
	runtimeCfg, err := s.integration.GetRuntimeConfig(ctx, pagination.TenantInfo{
		OrgID: p.orgID,
		BuID:  p.buID,
	}, integration.TypeOpenAI)
	if err != nil {
		s.recordAIUsage(string(p.operation), false, "missing_config")
		return nil, err
	}

	requestBody := s.buildResponsesRequest(p)
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
			p,
		)
		if execErr == nil {
			return envelope, nil
		}
		lastErr = execErr
		if attempt == s.cfg.GetAIMaxRetries() || !isRetryableAIError(execErr) {
			break
		}
		sleepDur := time.Duration(attempt+1) * 300 * time.Millisecond
		if sleepErr := retrySleep(ctx, sleepDur); sleepErr != nil {
			return nil, sleepErr
		}
	}

	s.recordAIUsage(string(p.operation), false, "error")
	return nil, lastErr
}

func (s *Service) executeOnce(
	ctx context.Context,
	apiKey string,
	body []byte,
	p *structuredResponseParams,
) (*responsesEnvelope, error) {
	envelope, err := s.doResponsesRequest(ctx, apiKey, http.MethodPost, responsesURL, body)
	if err != nil {
		return nil, err
	}

	text := extractResponseText(envelope)
	if text == "" {
		return nil, errortypes.NewBusinessError("AI response did not contain structured output")
	}
	if err = json.Unmarshal([]byte(text), p.out); err != nil {
		return nil, err
	}

	s.logAIInteraction(ctx, &ailog.Log{
		OrganizationID:   p.orgID,
		BusinessUnitID:   p.buID,
		UserID:           p.userID,
		Prompt:           redactPrompt(p.systemPrompt, p.userPrompt),
		Response:         redactResponse(text),
		Model:            p.model,
		Operation:        p.operation,
		Object:           p.documentID.String(),
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
		return nil, &apiError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	envelope := new(responsesEnvelope)
	if err = json.Unmarshal(respBody, envelope); err != nil {
		return nil, err
	}

	return envelope, nil
}

func retrySleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (s *Service) logAIInteraction(ctx context.Context, entry *ailog.Log) {
	if _, err := s.aiLogRepo.Create(ctx, entry); err != nil {
		s.logger.Warn("failed to persist ai log", zap.Error(err))
	}
}

func (s *Service) recordAIUsage(operation string, success bool, outcome string) {
	s.metrics.Document.RecordAIOutcome(operation, success, outcome)
}
