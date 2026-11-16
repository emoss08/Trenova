package emailservice

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/require"
)

const metadataInstance = "0192f3a4-5b6c-7d8e-9f01-23456789abcd"

func capturePayload(t *testing.T, answer string) (*httptest.Server, *map[string]any) {
	t.Helper()
	payload := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, sonic.ConfigDefault.NewDecoder(r.Body).Decode(&payload))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(server.Close)
	return server, &payload
}

func metadataMessage(metadata map[string]string) SendProviderMessage {
	return SendProviderMessage{
		From:     "Trenova <no-reply@mail.trenova.app>",
		To:       []string{"ops@example.com"},
		Subject:  "Subject",
		Text:     "Hello",
		Metadata: metadata,
	}
}

func TestResendSenderTagsTheMessageWithItsMetadata(t *testing.T) {
	t.Parallel()

	server, payload := capturePayload(t, `{"id":"resend-message-id"}`)
	_, err := NewResendSender(loopbackPolicy).Send(t.Context(), SendProviderRequest{
		Config:  map[string]string{"apiKey": "resend-api-key", "baseUrl": server.URL},
		Message: metadataMessage(map[string]string{"trenova_instance": metadataInstance}),
	})

	require.NoError(t, err)
	require.Equal(t, []any{map[string]any{
		"name":  "trenova_instance",
		"value": metadataInstance,
	}}, (*payload)["tags"])
}

func TestPostmarkSenderCarriesTheMessagesMetadata(t *testing.T) {
	t.Parallel()

	server, payload := capturePayload(t, `{"MessageID":"postmark-message-id"}`)
	_, err := NewPostmarkSender(loopbackPolicy).Send(t.Context(), SendProviderRequest{
		Config:  map[string]string{"serverToken": "postmark-token", "baseUrl": server.URL},
		Message: metadataMessage(map[string]string{"trenova_instance": metadataInstance}),
	})

	require.NoError(t, err)
	require.Equal(
		t,
		map[string]any{"trenova_instance": metadataInstance},
		(*payload)["Metadata"],
	)
}

func TestAMessageWithoutMetadataCarriesNoTags(t *testing.T) {
	t.Parallel()

	resendServer, resendPayload := capturePayload(t, `{"id":"resend-message-id"}`)
	_, err := NewResendSender(loopbackPolicy).Send(t.Context(), SendProviderRequest{
		Config:  map[string]string{"apiKey": "resend-api-key", "baseUrl": resendServer.URL},
		Message: metadataMessage(nil),
	})
	require.NoError(t, err)
	require.NotContains(t, *resendPayload, "tags")

	postmarkServer, postmarkPayload := capturePayload(t, `{"MessageID":"postmark-message-id"}`)
	_, err = NewPostmarkSender(loopbackPolicy).Send(t.Context(), SendProviderRequest{
		Config:  map[string]string{"serverToken": "postmark-token", "baseUrl": postmarkServer.URL},
		Message: metadataMessage(nil),
	})
	require.NoError(t, err)
	require.NotContains(t, *postmarkPayload, "Metadata")
}
