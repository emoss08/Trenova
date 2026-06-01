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

type PostmarkSender struct {
	client *http.Client
}

func NewPostmarkSender() *PostmarkSender {
	return &PostmarkSender{client: &http.Client{Timeout: 20 * time.Second}}
}

func (s *PostmarkSender) Provider() email.Provider {
	return email.ProviderPostmark
}

func (s *PostmarkSender) IntegrationType() integration.Type {
	return integration.TypePostmark
}

func (s *PostmarkSender) Send(
	ctx context.Context,
	req SendProviderRequest,
) (*SendProviderResponse, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(req.Config["baseUrl"]), "/")
	if baseURL == "" {
		baseURL = "https://api.postmarkapp.com"
	}
	messageStream := strings.TrimSpace(req.Config["messageStream"])
	if messageStream == "" {
		messageStream = "outbound"
	}
	msg := req.Message

	payload := map[string]any{
		"From":          msg.From,
		"To":            strings.Join(msg.To, ","),
		"Subject":       msg.Subject,
		"MessageStream": messageStream,
	}
	if msg.ReplyTo != "" {
		payload["ReplyTo"] = msg.ReplyTo
	}
	if len(msg.CC) > 0 {
		payload["Cc"] = strings.Join(msg.CC, ",")
	}
	if len(msg.BCC) > 0 {
		payload["Bcc"] = strings.Join(msg.BCC, ",")
	}
	if len(msg.Headers) > 0 {
		headers := make([]map[string]string, 0, len(msg.Headers))
		for name, value := range msg.Headers {
			headers = append(headers, map[string]string{
				"Name":  name,
				"Value": value,
			})
		}
		payload["Headers"] = headers
	}
	if msg.OpenTracking {
		payload["TrackOpens"] = true
	}
	if msg.HTML != "" {
		payload["HtmlBody"] = msg.HTML
	}
	if msg.Text != "" {
		payload["TextBody"] = msg.Text
	}
	if len(msg.Attachments) > 0 {
		attachments := make([]map[string]string, 0, len(msg.Attachments))
		for _, attachment := range msg.Attachments {
			attachments = append(attachments, map[string]string{
				"Name":        attachment.FileName,
				"Content":     base64.StdEncoding.EncodeToString(attachment.Content),
				"ContentType": attachment.ContentType,
			})
		}
		payload["Attachments"] = attachments
	}

	body, err := sonic.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal postmark payload: %w", ErrNonRetryableSend, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/email", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: create postmark request: %w", ErrNonRetryableSend, err)
	}
	httpReq.Header.Set("X-Postmark-Server-Token", req.Config["serverToken"])
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: postmark network failure: %w", ErrRetryableSend, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read postmark response: %w", ErrRetryableSend, err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var result struct {
			MessageID string `json:"MessageID"`
		}
		if err = sonic.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("%w: parse postmark response: %w", ErrRetryableSend, err)
		}
		return &SendProviderResponse{ProviderMessageID: result.MessageID}, nil
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, providerStatusError(ErrRetryableSend, "postmark", resp.StatusCode, respBody)
	}
	return nil, providerStatusError(ErrNonRetryableSend, "postmark", resp.StatusCode, respBody)
}
