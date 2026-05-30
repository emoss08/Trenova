package emailservice

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/integration"
)

type ResendSender struct {
	client *http.Client
}

func NewResendSender() *ResendSender {
	return &ResendSender{client: &http.Client{Timeout: 20 * time.Second}}
}

func (s *ResendSender) Provider() email.Provider {
	return email.ProviderResend
}

func (s *ResendSender) IntegrationType() integration.Type {
	return integration.TypeResend
}

func (s *ResendSender) Send(
	ctx context.Context,
	req SendProviderRequest,
) (*SendProviderResponse, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(req.Config["baseUrl"]), "/")
	if baseURL == "" {
		baseURL = "https://api.resend.com"
	}
	msg := req.Message

	payload := map[string]any{
		"from":    msg.From,
		"to":      msg.To,
		"subject": msg.Subject,
	}
	if msg.ReplyTo != "" {
		payload["reply_to"] = msg.ReplyTo
	}
	if len(msg.CC) > 0 {
		payload["cc"] = msg.CC
	}
	if len(msg.BCC) > 0 {
		payload["bcc"] = msg.BCC
	}
	if msg.HTML != "" {
		payload["html"] = msg.HTML
	}
	if msg.Text != "" {
		payload["text"] = msg.Text
	}
	if len(msg.Attachments) > 0 {
		attachments := make([]map[string]string, 0, len(msg.Attachments))
		for _, attachment := range msg.Attachments {
			attachments = append(attachments, map[string]string{
				"filename":     attachment.FileName,
				"content":      base64.StdEncoding.EncodeToString(attachment.Content),
				"content_type": attachment.ContentType,
			})
		}
		payload["attachments"] = attachments
	}

	body, err := sonic.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal resend payload: %w", ErrNonRetryableSend, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/emails", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: create resend request: %w", ErrNonRetryableSend, err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.Config["apiKey"])
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if msg.IdempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", msg.IdempotencyKey)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: resend network failure: %w", ErrRetryableSend, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read resend response: %w", ErrRetryableSend, err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var result struct {
			ID string `json:"id"`
		}
		if err = sonic.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("%w: parse resend response: %w", ErrRetryableSend, err)
		}
		return &SendProviderResponse{ProviderMessageID: result.ID}, nil
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: resend status %d", ErrRetryableSend, resp.StatusCode)
	}
	return nil, fmt.Errorf("%w: resend status %d", ErrNonRetryableSend, resp.StatusCode)
}
