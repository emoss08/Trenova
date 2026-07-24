package anthropiccompletionservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

type transportError struct {
	statusCode int
	retryable  bool
	message    string
}

func (e *transportError) Error() string {
	return fmt.Sprintf("anthropic request failed (status %d): %s", e.statusCode, e.message)
}

func (s *Service) executeMessages(
	ctx context.Context,
	apiKey string,
	body messagesRequest,
) (*messagesResponse, error) {
	encoded, err := sonic.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode anthropic request: %w", err)
	}

	attempts := s.cfg.GetAIMaxRetries()
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		resp, reqErr := s.doRequest(ctx, apiKey, encoded)
		if reqErr == nil {
			return resp, nil
		}

		lastErr = reqErr
		if !isRetryable(reqErr) {
			return nil, reqErr
		}

		s.logger.Warn("retrying anthropic request",
			zap.Int("attempt", attempt+1),
			zap.Error(reqErr),
		)
	}

	return nil, lastErr
}

func isRetryable(err error) bool {
	var te *transportError
	if errors.As(err, &te) {
		return te.retryable
	}

	return true
}

func (s *Service) doRequest(
	ctx context.Context,
	apiKey string,
	body []byte,
) (*messagesResponse, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		anthropicMessagesURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("build anthropic request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute anthropic request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &transportError{
			statusCode: resp.StatusCode,
			retryable:  resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500,
			message:    parseErrorMessage(payload),
		}
	}

	var envelope messagesResponse
	if err = sonic.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}

	return &envelope, nil
}

func parseErrorMessage(payload []byte) string {
	var envelope anthropicErrorEnvelope
	if err := sonic.Unmarshal(payload, &envelope); err == nil && envelope.Error.Message != "" {
		return envelope.Error.Message
	}

	return string(payload)
}
