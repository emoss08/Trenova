package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	samsaratypes "github.com/emoss08/trenova/shared/samsara/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInvalidBaseURL(t *testing.T) {
	t.Parallel()

	_, err := New(Config{
		Token:   "token",
		BaseURL: "://bad-url",
		Timeout: time.Second,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base URL")
}

func TestDoSuccessDecodesResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/ok", r.URL.Path)
		assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		_, _ = w.Write([]byte(`{"value":"ok"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		Token:   "token",
		BaseURL: server.URL,
		Timeout: 2 * time.Second,
		Retry:   RetryConfig{Enabled: false},
	})
	require.NoError(t, err)

	var out struct {
		Value string `json:"value"`
	}
	err = client.Do(t.Context(), Request{
		Method: http.MethodGet,
		Path:   "/ok",
		Out:    &out,
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", out.Value)
}

func TestDoPostEncodesBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/post", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		payload := map[string]string{}
		err = sonic.Unmarshal(body, &payload)
		require.NoError(t, err)
		assert.Equal(t, "hello", payload["message"])
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := New(Config{
		Token:   "token",
		BaseURL: server.URL,
		Timeout: 2 * time.Second,
		Retry:   RetryConfig{Enabled: false},
	})
	require.NoError(t, err)

	err = client.Do(t.Context(), Request{
		Method: http.MethodPost,
		Path:   "/post",
		Body:   map[string]string{"message": "hello"},
	})
	require.NoError(t, err)
}

func TestDoExpectedStatusNoContent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/resource", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(Config{
		Token:   "token",
		BaseURL: server.URL,
		Timeout: 2 * time.Second,
		Retry:   RetryConfig{Enabled: false},
	})
	require.NoError(t, err)

	err = client.Do(t.Context(), Request{
		Method:         http.MethodDelete,
		Path:           "/resource",
		ExpectedStatus: []int{http.StatusNoContent},
		Out:            &struct{}{},
	})
	require.NoError(t, err)
}

func TestDoReturnsAPIErrorOnUnexpectedStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"slow down","requestId":"req-123"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		Token:   "token",
		BaseURL: server.URL,
		Timeout: 2 * time.Second,
		Retry:   RetryConfig{Enabled: false},
	})
	require.NoError(t, err)

	err = client.Do(t.Context(), Request{
		Method: http.MethodGet,
		Path:   "/err",
	})
	require.Error(t, err)

	var apiErr *samsaratypes.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	assert.Equal(t, "slow down", apiErr.Message)
	assert.Equal(t, "req-123", apiErr.RequestID)
}

func TestDoDecodeError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	client, err := New(Config{
		Token:   "token",
		BaseURL: server.URL,
		Timeout: 2 * time.Second,
		Retry:   RetryConfig{Enabled: false},
	})
	require.NoError(t, err)

	var out struct {
		Value string `json:"value"`
	}
	err = client.Do(t.Context(), Request{
		Method: http.MethodGet,
		Path:   "/bad-json",
		Out:    &out,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	d, ok := parseRetryAfter("120")
	require.True(t, ok)
	assert.Equal(t, 120*time.Second, d)

	future := time.Now().Add(2 * time.Second).UTC().Format(http.TimeFormat)
	d, ok = parseRetryAfter(future)
	require.True(t, ok)
	assert.GreaterOrEqual(t, d, 0*time.Second)

	_, ok = parseRetryAfter("")
	assert.False(t, ok)

	_, ok = parseRetryAfter("bad")
	assert.False(t, ok)
}

func TestDoHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(150 * time.Millisecond)
	}))
	defer server.Close()

	client, err := New(Config{
		Token:   "token",
		BaseURL: server.URL,
		Timeout: time.Second,
		Retry:   RetryConfig{Enabled: false},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = client.Do(ctx, Request{
		Method: http.MethodGet,
		Path:   "/slow",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
