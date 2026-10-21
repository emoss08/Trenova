package authservice

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func testSSOConfig(issuer string) *tenant.SSOConfig {
	return &tenant.SSOConfig{
		ID:            pulid.MustNew("sso_"),
		OIDCIssuerURL: issuer,
		Version:       1,
		UpdatedAt:     1_700_000_000,
	}
}

func countingProviders(t *testing.T, policy httpsafe.Policy) (*oidcProviders, *atomic.Int32) {
	t.Helper()
	providers := newOIDCProviders(policy, time.Hour)
	var calls atomic.Int32
	providers.discover = func(ctx context.Context, _ string) (*oidc.Provider, error) {
		calls.Add(1)
		client, ok := ctx.Value(oauth2.HTTPClient).(*http.Client)
		if !ok || client != providers.client {
			return nil, errors.New("discovery did not receive the guarded client")
		}
		return &oidc.Provider{}, nil
	}
	return providers, &calls
}

func TestOIDCProvidersCacheDiscoveryPerConfiguration(t *testing.T) {
	t.Parallel()

	providers, calls := countingProviders(t, httpsafe.Policy{})
	cfg := testSSOConfig("https://login.example.com/tenant/v2.0")

	first, err := providers.get(t.Context(), cfg)
	require.NoError(t, err)
	second, err := providers.get(t.Context(), cfg)
	require.NoError(t, err)
	require.Same(t, first, second)
	require.Equal(t, int32(1), calls.Load())

	other := testSSOConfig("https://login.example.com/other/v2.0")
	_, err = providers.get(t.Context(), other)
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load())
}

func TestOIDCProvidersRediscoverWhenTheConfigurationChanges(t *testing.T) {
	t.Parallel()

	providers, calls := countingProviders(t, httpsafe.Policy{})
	cfg := testSSOConfig("https://login.example.com/tenant/v2.0")
	_, err := providers.get(t.Context(), cfg)
	require.NoError(t, err)

	cfg.Version = 2
	cfg.UpdatedAt++
	_, err = providers.get(t.Context(), cfg)
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load())

	cfg.OIDCIssuerURL = "https://idp.example.com/realms/trenova"
	_, err = providers.get(t.Context(), cfg)
	require.NoError(t, err)
	require.Equal(t, int32(3), calls.Load())
}

func TestOIDCProvidersRediscoverAfterTheTTL(t *testing.T) {
	t.Parallel()

	providers, calls := countingProviders(t, httpsafe.Policy{})
	now := time.Unix(1_700_000_000, 0)
	providers.now = func() time.Time { return now }
	cfg := testSSOConfig("https://login.example.com/tenant/v2.0")

	_, err := providers.get(t.Context(), cfg)
	require.NoError(t, err)
	now = now.Add(59 * time.Minute)
	_, err = providers.get(t.Context(), cfg)
	require.NoError(t, err)
	require.Equal(t, int32(1), calls.Load())

	now = now.Add(2 * time.Minute)
	_, err = providers.get(t.Context(), cfg)
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load())
}

func TestOIDCProvidersRefuseUnsafeIssuers(t *testing.T) {
	t.Parallel()

	providers, calls := countingProviders(t, httpsafe.Policy{})
	for _, issuer := range []string{
		"http://login.example.com",
		"https://user:pass@login.example.com",
		"https://127.0.0.1/realms/trenova",
		"https://169.254.169.254/latest",
		"https://[::1]/realms/trenova",
		"file:///etc/passwd",
		"",
	} {
		_, err := providers.get(t.Context(), testSSOConfig(issuer))
		require.Error(t, err, issuer)
	}
	require.Zero(t, calls.Load())

	private, privateCalls := countingProviders(t, httpsafe.Policy{AllowPrivateNetworks: true})
	_, err := private.get(t.Context(), testSSOConfig("https://10.0.4.2/realms/trenova"))
	require.NoError(t, err)
	_, err = private.get(t.Context(), testSSOConfig("https://169.254.169.254/latest"))
	require.ErrorIs(t, err, httpsafe.ErrBlockedAddress)
	require.Equal(t, int32(1), privateCalls.Load())
}

func TestOIDCDiscoveryCannotReachPrivateAddressesThroughAHostname(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	providers := newOIDCProviders(httpsafe.Policy{}, time.Hour)
	issuer := strings.Replace(server.URL, "127.0.0.1", "localhost", 1)
	_, err := providers.get(t.Context(), testSSOConfig(issuer))
	require.ErrorIs(t, err, httpsafe.ErrBlockedAddress)
	require.Zero(t, hits.Load())
}
