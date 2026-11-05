package emailservice

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var loopbackPolicy = httpsafe.Policy{AllowPrivateNetworks: true}

func TestProviderSendersRefuseALoopbackBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		sender ProviderSender
	}{
		{name: "resend", sender: NewResendSender(httpsafe.Policy{})},
		{name: "postmark", sender: NewPostmarkSender(httpsafe.Policy{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false
			server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			}))
			defer server.Close()

			_, err := tt.sender.Send(t.Context(), SendProviderRequest{
				Config: map[string]string{
					"apiKey":      "dummy",
					"serverToken": "dummy",
					"baseUrl":     server.URL,
				},
				Message: SendProviderMessage{From: "a@b.example", To: []string{"c@d.example"}},
			})

			require.Error(t, err)
			assert.False(t, called)
		})
	}
}
