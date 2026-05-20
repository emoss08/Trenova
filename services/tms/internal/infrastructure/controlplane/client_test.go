package controlplane

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/require"
)

func TestHTTPControlPlaneClient_SignsRequests(t *testing.T) {
	t.Parallel()

	const (
		apiKey     = "test-api-key"
		instanceID = "inst_01"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		bodyHash := bodySHA256(body)
		timestamp := r.Header.Get("X-Trenova-Timestamp")

		require.Equal(t, "Bearer "+apiKey, r.Header.Get("Authorization"))
		require.Equal(t, instanceID, r.Header.Get("X-Trenova-Instance-ID"))
		require.NotEmpty(t, timestamp)
		require.Equal(t, bodyHash, r.Header.Get("X-Trenova-Body-SHA256"))
		require.Equal(
			t,
			computeSignature(apiKey, http.MethodPost, "/v1/entitlements/check", bodyHash, timestamp),
			r.Header.Get("X-Trenova-Signature"),
		)

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"featureKey":"custom_fields","allowed":true,"checkedAt":123}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewHTTPControlPlaneClient(HTTPControlPlaneClientParams{
		Config: &config.Config{
			Platform: config.PlatformConfig{
				InstanceID: instanceID,
				ControlPlane: config.PlatformControlPlaneConfig{
					Endpoint: server.URL,
					APIKey:   apiKey,
				},
			},
		},
		HTTPClient: server.Client(),
	})

	result, err := client.CheckFeature(t.Context(), &services.FeatureCheckRequest{
		FeatureKey: platformcatalog.FeatureCoreTMS,
	})

	require.NoError(t, err)
	require.True(t, result.Allowed)
}
