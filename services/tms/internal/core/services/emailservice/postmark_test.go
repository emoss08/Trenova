package emailservice

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/require"
)

func TestPostmarkSenderSendsMessageWithAttachments(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/email", r.URL.Path)
		require.Equal(t, "server-token", r.Header.Get("X-Postmark-Server-Token"))

		var payload map[string]any
		require.NoError(t, sonic.ConfigDefault.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "Dispatch <dispatch@example.com>", payload["From"])
		require.Equal(t, "ops@example.com", payload["To"])
		require.Equal(t, "billing@example.com", payload["Cc"])
		require.Equal(t, "outbound", payload["MessageStream"])
		require.Equal(t, "Subject", payload["Subject"])
		require.Equal(t, "<p>Hello</p>", payload["HtmlBody"])
		require.Equal(t, true, payload["TrackOpens"])

		headers, ok := payload["Headers"].([]any)
		require.True(t, ok)
		require.Len(t, headers, 1)
		header, ok := headers[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "Disposition-Notification-To", header["Name"])
		require.Equal(t, "dispatch@example.com", header["Value"])

		attachments, ok := payload["Attachments"].([]any)
		require.True(t, ok)
		require.Len(t, attachments, 1)
		attachment, ok := attachments[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "invoice.pdf", attachment["Name"])
		require.Equal(t, "application/pdf", attachment["ContentType"])
		require.Equal(t, "cGRm", attachment["Content"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"MessageID":"pm-message-id"}`))
	}))
	defer server.Close()

	result, err := NewPostmarkSender().Send(t.Context(), SendProviderRequest{
		Config: map[string]string{
			"serverToken":   "server-token",
			"baseUrl":       server.URL,
			"messageStream": "outbound",
		},
		Message: SendProviderMessage{
			From:    "Dispatch <dispatch@example.com>",
			To:      []string{"ops@example.com"},
			CC:      []string{"billing@example.com"},
			Subject: "Subject",
			HTML:    "<p>Hello</p>",
			Headers: map[string]string{
				"Disposition-Notification-To": "dispatch@example.com",
			},
			OpenTracking: true,
			Attachments: []ProviderAttachment{
				{
					FileName:    "invoice.pdf",
					ContentType: "application/pdf",
					Content:     []byte("pdf"),
				},
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "pm-message-id", result.ProviderMessageID)
}

func TestPostmarkSenderClassifiesFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    int
		retryable bool
	}{
		{name: "rate limit", status: http.StatusTooManyRequests, retryable: true},
		{name: "server error", status: http.StatusInternalServerError, retryable: true},
		{name: "validation error", status: http.StatusUnprocessableEntity, retryable: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "failure", tt.status)
			}))
			defer server.Close()

			_, err := NewPostmarkSender().Send(t.Context(), SendProviderRequest{
				Config: map[string]string{
					"serverToken": "server-token",
					"baseUrl":     server.URL,
				},
				Message: SendProviderMessage{
					From:    "dispatch@example.com",
					To:      []string{"ops@example.com"},
					Subject: "Subject",
					Text:    "Hello",
				},
			})

			require.Error(t, err)
			require.Equal(t, tt.retryable, errors.Is(err, serviceports.ErrRetryableEmailSend))
			require.Equal(t, !tt.retryable, errors.Is(err, serviceports.ErrNonRetryableEmailSend))
			require.Contains(t, err.Error(), "failure")
		})
	}
}
