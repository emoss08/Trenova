package xero_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testConnection = "e1eede29-f875-4a5d-8470-17f6a29a88b1"
	testAuthEvent  = "d99ecdfe-391d-43d2-b834-17636ba90e8d"
)

func newConnectionsClient(t *testing.T, handler http.HandlerFunc) *xero.ConnectionsClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := xero.NewConnectionsClient(testAccessToken, xero.WithBaseURL(server.URL), fastRetry())
	require.NoError(t, err)
	return client
}

func TestNewConnectionsClientRequiresToken(t *testing.T) {
	t.Parallel()

	_, err := xero.NewConnectionsClient(" ")
	require.ErrorIs(t, err, xero.ErrAccessTokenRequired)
}

func TestConnectionsListFiltersByAuthEvent(t *testing.T) {
	t.Parallel()

	client := newConnectionsClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/connections", r.URL.Path)
		assert.Equal(t, testAuthEvent, r.URL.Query().Get("authEventId"))
		assert.Equal(t, "Bearer "+testAccessToken, r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("xero-tenant-id"), "connections are not tenant scoped")
		_, _ = w.Write(fixture(t, "connections.json"))
	})

	connections, err := client.List(t.Context(), testAuthEvent)
	require.NoError(t, err)
	require.Len(t, connections, 1)
	conn := connections[0]
	assert.Equal(t, testConnection, conn.ID)
	assert.Equal(t, testAuthEvent, conn.AuthEventID)
	assert.Equal(t, testTenant, conn.TenantID)
	assert.Equal(t, "ORGANISATION", conn.TenantType)
	assert.Equal(t, "Acme Freight Ltd", conn.TenantName)
	assert.Equal(t, time.Date(2026, 9, 20, 23, 40, 30, 183_313_000, time.UTC), conn.CreatedAt)
	assert.Equal(t, time.Date(2026, 9, 21, 1, 2, 3, 500_000_000, time.UTC), conn.UpdatedAt)
}

func TestConnectionsListWithoutAuthEventSendsNoFilter(t *testing.T) {
	t.Parallel()

	client := newConnectionsClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.RawQuery)
		_, _ = w.Write([]byte(`[]`))
	})

	connections, err := client.List(t.Context(), "")
	require.NoError(t, err)
	assert.Empty(t, connections)
}

func TestConnectionsDelete(t *testing.T) {
	t.Parallel()

	client := newConnectionsClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/connections/"+testConnection, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})

	require.NoError(t, client.Delete(t.Context(), testConnection))
	require.ErrorIs(t, client.Delete(t.Context(), "../tenants"), xero.ErrInvalidID)
	require.ErrorIs(t, client.Delete(t.Context(), ""), xero.ErrIDRequired)
}

func TestConnectionsExpiredTokenIsAuth(t *testing.T) {
	t.Parallel()

	client := newConnectionsClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write(fixture(t, "error_401.json"))
	})

	_, err := client.List(t.Context(), testAuthEvent)
	require.Error(t, err)
	assert.True(t, xero.IsAuth(err))
	assert.NotContains(t, err.Error(), testAccessToken)
}
