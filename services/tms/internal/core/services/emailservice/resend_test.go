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

func TestResendSenderSendsMessageWithAttachments(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/emails", r.URL.Path)
		require.Equal(t, "Bearer resend-api-key", r.Header.Get("Authorization"))
		require.Equal(t, "idem-key", r.Header.Get("Idempotency-Key"))

		var payload map[string]any
		require.NoError(t, sonic.ConfigDefault.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "Dispatch <dispatch@example.com>", payload["from"])
		require.Equal(t, []any{"ops@example.com"}, payload["to"])
		require.Equal(t, []any{"billing@example.com"}, payload["cc"])
		require.Equal(t, "Subject", payload["subject"])
		require.Equal(t, "<p>Hello</p>", payload["html"])
		headers, ok := payload["headers"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "dispatch@example.com", headers["Disposition-Notification-To"])

		attachments, ok := payload["attachments"].([]any)
		require.True(t, ok)
		require.Len(t, attachments, 1)
		attachment, ok := attachments[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "invoice.pdf", attachment["filename"])
		require.Equal(t, "application/pdf", attachment["content_type"])
		require.Equal(t, "cGRm", attachment["content"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resend-message-id"}`))
	}))
	defer server.Close()

	result, err := NewResendSender().Send(t.Context(), SendProviderRequest{
		Config: map[string]string{
			"apiKey":  "resend-api-key",
			"baseUrl": server.URL,
		},
		Message: SendProviderMessage{
			IdempotencyKey: "idem-key",
			From:           "Dispatch <dispatch@example.com>",
			To:             []string{"ops@example.com"},
			CC:             []string{"billing@example.com"},
			Subject:        "Subject",
			HTML:           "<p>Hello</p>",
			Headers: map[string]string{
				"Disposition-Notification-To": "dispatch@example.com",
			},
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
	require.Equal(t, "resend-message-id", result.ProviderMessageID)
}

func TestResendSenderIncludesProviderFailureBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"domain is not verified"}`))
	}))
	defer server.Close()

	_, err := NewResendSender().Send(t.Context(), SendProviderRequest{
		Config: map[string]string{
			"apiKey":  "resend-api-key",
			"baseUrl": server.URL,
		},
		Message: SendProviderMessage{
			From:    "Dispatch <dispatch@example.com>",
			To:      []string{"ops@example.com"},
			Subject: "Subject",
			Text:    "Hello",
		},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, serviceports.ErrNonRetryableEmailSend))
	require.Contains(t, err.Error(), "domain is not verified")
}
