package integrationservice

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loopbackClient() *http.Client {
	return httpsafe.NewClientWithPolicy(
		connectionTestTimeout,
		httpsafe.Policy{AllowPrivateNetworks: true},
	)
}

func TestConnectionTestersRefuseALoopbackBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typ    integration.Type
		config map[string]string
	}{
		{typ: integration.TypeResend, config: map[string]string{"apiKey": "dummy"}},
		{typ: integration.TypePostmark, config: map[string]string{"serverToken": "dummy"}},
		{typ: integration.TypeOANDAExchangeRates, config: map[string]string{"apiKey": "dummy"}},
		{typ: integration.TypeEIAFuelPrices, config: map[string]string{"apiKey": "dummy"}},
		{typ: integration.TypePCMiler, config: map[string]string{"apiKey": "dummy"}},
		{typ: integration.TypeSamsara, config: map[string]string{"token": "dummy"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.typ), func(t *testing.T) {
			t.Parallel()

			called := false
			server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			}))
			defer server.Close()

			testers := newConnectionTesters(httpsafe.NewClient(connectionTestTimeout))
			config := map[string]string{"baseUrl": server.URL}
			for key, value := range tt.config {
				config[key] = value
			}

			err := testers[tt.typ].Test(t.Context(), config)

			require.Error(t, err)
			assert.False(t, called, "the request never reaches the loopback host")
		})
	}
}

func TestPCMilerConnectionTesterDoesNotRepeatAResponseBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
	}{
		{name: "a refusal", status: http.StatusForbidden},
		{name: "an unreadable success", status: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte("INTERNAL-ONLY-BODY"))
				},
			))
			defer server.Close()

			err := (&pcmilerConnectionTester{client: loopbackClient()}).Test(
				t.Context(),
				map[string]string{"apiKey": "dummy", "baseUrl": server.URL},
			)

			require.Error(t, err)
			assert.NotContains(t, err.Error(), "INTERNAL-ONLY-BODY")
		})
	}
}

func TestValidateURLFields(t *testing.T) {
	t.Parallel()

	spec := integration.ConfigSpecs[integration.TypeResend]
	private := httpsafe.Policy{AllowPrivateNetworks: true}

	tests := []struct {
		name    string
		value   string
		policy  httpsafe.Policy
		refused bool
	}{
		{name: "the provider's own address", value: "https://api.resend.com"},
		{name: "unset keeps the default", value: ""},
		{name: "plain http", value: "http://api.resend.com", refused: true},
		{name: "loopback", value: "https://127.0.0.1", refused: true},
		{name: "cloud metadata", value: "https://169.254.169.254", refused: true},
		{name: "credentials", value: "https://user:pass@api.resend.com", refused: true},
		{name: "a query that rewrites the path", value: "https://api.resend.com/?x=", refused: true},
		{name: "a fragment", value: "https://api.resend.com/#x", refused: true},
		{name: "a simulator on an opted-in network", value: "http://localhost:8091", policy: private},
		{
			name:    "cloud metadata stays refused on an opted-in network",
			value:   "http://169.254.169.254",
			policy:  private,
			refused: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateURLFields(spec, map[string]any{"baseUrl": tt.value}, tt.policy)

			if !tt.refused {
				require.NoError(t, err)
				return
			}
			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
		})
	}
}
