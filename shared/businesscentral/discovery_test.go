package businesscentral_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDiscoveryClient(t *testing.T, handler http.HandlerFunc) *businesscentral.DiscoveryClient {
	t.Helper()
	server := newServer(t, handler)
	client, err := businesscentral.NewDiscoveryClient(testAccessToken,
		businesscentral.WithBaseURL(server.URL), fastRetry())
	require.NoError(t, err)
	return client
}

func TestNewDiscoveryClientRequiresToken(t *testing.T) {
	t.Parallel()

	_, err := businesscentral.NewDiscoveryClient("")
	require.ErrorIs(t, err, businesscentral.ErrAccessTokenRequired)
}

func TestEnvironmentsKeepOnlyBusinessCentral(t *testing.T) {
	t.Parallel()

	client := newDiscoveryClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/environments/v1.2")
		_, _ = w.Write(fixture(t, "environments.json"))
	})

	environments, err := client.Environments(t.Context())
	require.NoError(t, err)
	require.Len(t, environments, 2)
	assert.Equal(t, testTenant, environments[0].TenantID)
	assert.Equal(t, "Production", environments[0].Name)
	assert.True(t, environments[0].IsProduction())
	assert.Equal(t, "US", environments[0].CountryCode)
	assert.Equal(t, "uat_sandbox-2", environments[1].Name)
	assert.Equal(t, businesscentral.EnvironmentTypeSandbox, environments[1].Type)
	assert.False(t, environments[1].IsProduction())
}

func TestCompanies(t *testing.T) {
	t.Parallel()

	client := newDiscoveryClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/v2.0/"+testTenant+"/uat_sandbox-2/api/v2.0/companies")
		_, _ = w.Write(fixture(t, "companies.json"))
	})

	companies, err := client.Companies(t.Context(), testTenant, "uat_sandbox-2")
	require.NoError(t, err)
	require.Len(t, companies, 2)
	assert.Equal(t, testCompany, companies[0].ID)
	assert.Equal(t, "CRONUS USA, Inc.", companies[0].Name)
	assert.Equal(t, int64(6127), companies[0].Timestamp)
	assert.Equal(t, "25.0.23364.25190", companies[0].SystemVersion)
	assert.Equal(t, time.Date(2026, 1, 5, 14, 22, 31, 177000000, time.UTC), companies[0].CreatedAt)
	assert.True(t, companies[1].ModifiedAt.IsZero(), "the zero BC date-time is unset")
	assert.Equal(t, "Acme Freight LLC", companies[1].DisplayName)
}

func TestCompaniesValidatesTenantAndEnvironment(t *testing.T) {
	t.Parallel()

	client := newDiscoveryClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("no request is sent for invalid input")
	})
	_, err := client.Companies(t.Context(), "tenant/../x", "Production")
	require.ErrorIs(t, err, businesscentral.ErrInvalidID)
	_, err = client.Companies(t.Context(), testTenant, "Prod/../admin")
	require.ErrorIs(t, err, businesscentral.ErrInvalidEnvironment)
}
