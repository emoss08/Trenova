package samsara

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/samsara/addresses"
	"github.com/emoss08/trenova/shared/samsara/drivers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidation(t *testing.T) {
	t.Parallel()

	_, err := New("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token is required")

	_, err = New("token", WithBaseURL("::://bad-url"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid samsara base URL")

	_, err = New("token", WithRetry(RetryConfig{Enabled: true, MaxAttempts: 11}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maxAttempts")
}

func TestNewAcceptsAPIKey(t *testing.T) {
	t.Parallel()

	client, err := New("api-key")
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.Addresses)
	require.NotNil(t, client.Assets)
	require.NotNil(t, client.Compliance)
	require.NotNil(t, client.Drivers)
	require.NotNil(t, client.Dvirs)
	require.NotNil(t, client.Forms)
	require.NotNil(t, client.LiveShares)
	require.NotNil(t, client.Messages)
	require.NotNil(t, client.Routes)
	require.NotNil(t, client.Vehicles)
	require.NotNil(t, client.Webhooks)
}

func TestRetry429ThenSuccess(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/addresses" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		current := calls.Add(1)
		if current == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"Exceeded rate limit.","requestId":"req-1"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[],"pagination":{"endCursor":"","hasNextPage":false}}`))
	}))
	defer server.Close()

	client, err := New(
		"test-token",
		WithBaseURL(server.URL),
		WithRetry(RetryConfig{
			Enabled:        true,
			MaxAttempts:    3,
			InitialBackoff: time.Millisecond,
			MaxBackoff:     2 * time.Millisecond,
		}),
	)
	require.NoError(t, err)

	page, err := client.Addresses.List(t.Context(), addresses.ListParams{})
	require.NoError(t, err)
	assert.Empty(t, page.Data)
	assert.Equal(t, int32(2), calls.Load())
}

func TestRequestReturnsTypedAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Unauthorized response.","requestId":"abc123"}`))
	}))
	defer server.Close()

	client, err := New("bad-token", WithBaseURL(server.URL))
	require.NoError(t, err)

	_, err = client.Addresses.List(t.Context(), addresses.ListParams{})
	require.Error(t, err)
	assert.True(t, IsUnauthorized(err))

	apiErr := &APIError{}
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 401, apiErr.StatusCode)
	assert.Equal(t, "abc123", apiErr.RequestID)
}

func TestClientPacesLevelOneWritesPerToken(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"1654973","name":"Driver"}}`))
	}))
	defer server.Close()

	token := "pacing-token-" + t.Name()
	first, err := New(token, WithBaseURL(server.URL))
	require.NoError(t, err)
	second, err := New(token, WithBaseURL(server.URL))
	require.NoError(t, err)

	started := time.Now()
	_, err = first.Drivers.Update(t.Context(), "1654973", drivers.UpdateRequest{})
	require.NoError(t, err)
	_, err = second.Drivers.Update(t.Context(), "1654974", drivers.UpdateRequest{})
	require.NoError(t, err)

	assert.GreaterOrEqual(t, time.Since(started), 590*time.Millisecond)
	assert.Equal(t, int32(2), calls.Load())
}

func TestClientWithoutRateLimitingDoesNotPace(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"1654973","name":"Driver"}}`))
	}))
	defer server.Close()

	client, err := New(
		"unpaced-token-"+t.Name(),
		WithBaseURL(server.URL),
		WithRateLimiting(false),
	)
	require.NoError(t, err)

	started := time.Now()
	for _, id := range []string{"1654973", "1654974", "1654975"} {
		_, err = client.Drivers.Update(t.Context(), id, drivers.UpdateRequest{})
		require.NoError(t, err)
	}
	assert.Less(t, time.Since(started), 500*time.Millisecond)
}
