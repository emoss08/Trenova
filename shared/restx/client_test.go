package restx_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testWebKey = "wk_super_secret_value_123"
	testToken  = "sk_live_super_secret_token"
)

type countingLimiter struct {
	calls   atomic.Int32
	buckets []restx.Bucket
	mu      sync.Mutex
	err     error
}

func (l *countingLimiter) Acquire(_ context.Context, bucket restx.Bucket) error {
	l.calls.Add(1)
	l.mu.Lock()
	l.buckets = append(l.buckets, bucket)
	l.mu.Unlock()
	return l.err
}

func newClient(t *testing.T, handler http.HandlerFunc, mutate func(*restx.Config)) *restx.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cfg := restx.Config{
		BaseURL:     server.URL + "/qc/services",
		Timeout:     5 * time.Second,
		UserAgent:   "restx-test",
		Headers:     map[string]string{"Authorization": "Bearer " + testToken},
		QueryParams: map[string]string{"webKey": testWebKey},
		Retry: restx.RetryConfig{
			Enabled:        true,
			MaxAttempts:    3,
			InitialBackoff: time.Millisecond,
			MaxBackoff:     5 * time.Millisecond,
		},
	}
	if mutate != nil {
		mutate(&cfg)
	}

	client, err := restx.New(cfg)
	require.NoError(t, err)
	return client
}

func TestNewValidatesBaseURL(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "://bad", "ftp://example.com", "/relative", "https://example.com?x=1"} {
		_, err := restx.New(restx.Config{BaseURL: raw})
		require.ErrorIs(t, err, restx.ErrInvalidBaseURL, raw)
	}

	_, err := restx.New(restx.Config{
		BaseURL: "https://example.com",
		Limiter: &countingLimiter{},
	})
	require.ErrorIs(t, err, restx.ErrLimiterNoBucket)
}

func TestDoSuccessDecodesAndSendsConfiguredValues(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/qc/services/carriers/name/A%2FB%20Co", r.URL.EscapedPath())
		assert.Equal(t, testWebKey, r.URL.Query().Get("webKey"))
		assert.Equal(t, "10", r.URL.Query().Get("size"))
		assert.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "restx-test", r.Header.Get("User-Agent"))

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"name":"acme"}`, string(body))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":"ok","count":2}`))
	}, nil)

	var out struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}
	resp, err := client.Do(t.Context(), &restx.Request{
		Endpoint: "name",
		Method:   http.MethodPost,
		Path:     "/carriers/name/" + url.PathEscape("A/B Co"),
		Query:    url.Values{"size": []string{"10"}},
		Body:     map[string]string{"name": "acme"},
		Out:      &out,
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, resp.Attempts)
	assert.Equal(t, "ok", out.Value)
	assert.Equal(t, 2, out.Count)
}

func TestDoQueryParamsOverrideRequestQuery(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{testWebKey}, r.URL.Query()["webKey"])
		w.WriteHeader(http.StatusOK)
	}, nil)

	_, err := client.Do(t.Context(), &restx.Request{
		Path:  "/carriers/1",
		Query: url.Values{"webKey": []string{"attacker"}},
	})
	require.NoError(t, err)
}

func TestDoRetriesOn429HonoringRetryAfter(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"slow down"}`))
			return
		}
		_, _ = w.Write([]byte(`{"value":"done"}`))
	}, nil)

	var out struct {
		Value string `json:"value"`
	}
	started := time.Now()
	resp, err := client.Do(t.Context(), &restx.Request{Path: "/x", Out: &out})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, time.Since(started), 900*time.Millisecond)
	assert.Equal(t, int32(2), calls.Load())
	assert.Equal(t, 2, resp.Attempts)
	assert.Equal(t, "done", out.Value)
}

func TestDoObserverInvokedPerAttempt(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	var mu sync.Mutex
	infos := make([]restx.CallInfo, 0, 3)

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}, func(cfg *restx.Config) {
		cfg.Observer = func(info restx.CallInfo) {
			mu.Lock()
			defer mu.Unlock()
			infos = append(infos, info)
		}
	})

	_, err := client.Do(t.Context(), &restx.Request{Endpoint: "probe", Path: "/probe"})
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, infos, 3)
	for i, info := range infos {
		assert.Equal(t, i+1, info.Attempt)
		assert.Equal(t, "probe", info.Endpoint)
		assert.Equal(t, "/probe", info.Path)
	}
	assert.Equal(t, http.StatusServiceUnavailable, infos[0].StatusCode)
	require.Error(t, infos[0].Err)
	assert.Equal(t, http.StatusOK, infos[2].StatusCode)
	assert.NoError(t, infos[2].Err)
}

func TestDoDoesNotRetryOn400Or501(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusBadRequest, http.StatusNotImplemented} {
		var calls atomic.Int32
		client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"bad input","code":"invalid","hint":"fix it"}`))
		}, nil)

		resp, err := client.Do(t.Context(), &restx.Request{Path: "/x"})
		require.Error(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int32(1), calls.Load())
		assert.True(t, restx.IsStatus(err, status))
		assert.Equal(t, status, restx.StatusCode(err))

		var apiErr *restx.APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, "bad input", apiErr.Message)
		assert.Equal(t, "invalid", apiErr.Code)
		assert.Equal(t, "fix it", apiErr.Hint)
	}
}

func TestDoExhaustsRetriesOn5xx(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}, nil)

	_, err := client.Do(t.Context(), &restx.Request{Path: "/x"})
	require.Error(t, err)
	assert.Equal(t, int32(3), calls.Load())
	assert.True(t, restx.IsStatus(err, http.StatusBadGateway))
}

func TestDoLimiterCalledOncePerAttempt(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	limiter := &countingLimiter{}
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}, func(cfg *restx.Config) {
		cfg.Limiter = limiter
		cfg.BucketFor = func(endpoint string) (restx.Bucket, bool) {
			return restx.Bucket{Key: "test:" + endpoint, Limit: 10, Period: time.Minute}, true
		}
	})

	_, err := client.Do(t.Context(), &restx.Request{Endpoint: "profile", Path: "/v2/profile"})
	require.NoError(t, err)
	assert.Equal(t, int32(2), limiter.calls.Load())
	assert.Equal(t, int32(2), calls.Load())
	for _, bucket := range limiter.buckets {
		assert.Equal(t, "test:profile", bucket.Key)
		assert.Equal(t, 1, bucket.Cost)
	}
}

func TestDoLimiterSkippedWhenNoBucket(t *testing.T) {
	t.Parallel()

	limiter := &countingLimiter{}
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, func(cfg *restx.Config) {
		cfg.Limiter = limiter
		cfg.BucketFor = func(string) (restx.Bucket, bool) { return restx.Bucket{}, false }
	})

	_, err := client.Do(t.Context(), &restx.Request{Endpoint: "profile", Path: "/v2/profile"})
	require.NoError(t, err)
	assert.Equal(t, int32(0), limiter.calls.Load())
}

func TestDoLimiterErrorAborts(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	limiter := &countingLimiter{
		err: &restx.RateLimitedError{RetryAfter: 2 * time.Second, Key: "test:profile"},
	}
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}, func(cfg *restx.Config) {
		cfg.Limiter = limiter
		cfg.BucketFor = func(endpoint string) (restx.Bucket, bool) {
			return restx.Bucket{Key: "test:" + endpoint}, true
		}
	})

	_, err := client.Do(t.Context(), &restx.Request{Endpoint: "profile", Path: "/v2/profile"})
	require.Error(t, err)

	var limited *restx.RateLimitedError
	require.ErrorAs(t, err, &limited)
	assert.True(t, restx.IsRateLimited(err))
	assert.Equal(t, 2*time.Second, restx.RetryAfterOf(err))
	assert.Equal(t, int32(0), calls.Load())
	assert.Equal(t, int32(1), limiter.calls.Load())
}

func TestDoContextCanceledDuringBackoff(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}, func(cfg *restx.Config) {
		cfg.Retry = restx.RetryConfig{
			Enabled:        true,
			MaxAttempts:    5,
			InitialBackoff: 5 * time.Second,
			MaxBackoff:     5 * time.Second,
		}
	})

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := client.Do(ctx, &restx.Request{Path: "/x"})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(started), 3*time.Second)
	assert.Equal(t, int32(1), calls.Load())
}

func TestDoDecodeFailure(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"value":`))
	}, nil)

	var out struct {
		Value string `json:"value"`
	}
	_, err := client.Do(t.Context(), &restx.Request{Path: "/x", Out: &out})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode GET /x response")
}

func TestDoCustomErrorDecoder(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("custom failure")
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}, func(cfg *restx.Config) {
		cfg.ErrorDecoder = func(status int, _ []byte, _ http.Header) error {
			assert.Equal(t, http.StatusForbidden, status)
			return sentinel
		}
	})

	_, err := client.Do(t.Context(), &restx.Request{Path: "/x"})
	require.ErrorIs(t, err, sentinel)
}

func TestErrorsNeverContainSecrets(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		payload, _ := sonic.Marshal(map[string]string{
			"error": "invalid key " + r.URL.Query().
				Get("webKey") +
				" / " + r.Header.Get(
				"Authorization",
			),
			"hint": "check " + testToken,
		})
		_, _ = w.Write(payload)
	}, nil)

	_, err := client.Do(t.Context(), &restx.Request{Path: "/carriers/1"})
	require.Error(t, err)
	assertNoSecrets(t, err)

	var apiErr *restx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.NotContains(t, string(apiErr.Body), testWebKey)
	assert.NotContains(t, string(apiErr.Body), testToken)

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachable := server.URL
	server.Close()

	offline, err := restx.New(restx.Config{
		BaseURL:     unreachable,
		Headers:     map[string]string{"Authorization": "Bearer " + testToken},
		QueryParams: map[string]string{"webKey": testWebKey},
	})
	require.NoError(t, err)

	_, err = offline.Do(t.Context(), &restx.Request{Method: http.MethodGet, Path: "/carriers/1"})
	require.Error(t, err)
	assertNoSecrets(t, err)

	var transportErr *restx.TransportError
	require.ErrorAs(t, err, &transportErr)
	assert.Equal(t, "/carriers/1", transportErr.Path)
	assert.Contains(t, err.Error(), "GET /carriers/1")
	assert.NotContains(t, err.Error(), "?")
}

func assertNoSecrets(t *testing.T, err error) {
	t.Helper()
	for current := err; current != nil; current = errors.Unwrap(current) {
		message := current.Error()
		assert.NotContains(t, message, testWebKey)
		assert.NotContains(t, message, testToken)
		assert.NotContains(t, message, url.QueryEscape(testWebKey))
	}
}

func TestRedactURL(t *testing.T) {
	t.Parallel()

	parsed, err := url.Parse(
		"https://mobile.fmcsa.dot.gov/qc/services/carriers/1?webKey=abc123&size=5",
	)
	require.NoError(t, err)

	redacted := restx.RedactURL(parsed, []string{"webkey"})
	assert.NotContains(t, redacted, "abc123")
	assert.Contains(t, redacted, "webKey=REDACTED")
	assert.Contains(t, redacted, "size=5")
	assert.Contains(t, parsed.String(), "abc123")
	assert.Empty(t, restx.RedactURL(nil, nil))
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		ok    bool
		exact time.Duration
		check func(t *testing.T, d time.Duration)
	}{
		{name: "seconds", value: "120", ok: true, exact: 120 * time.Second},
		{name: "zero", value: "0", ok: true, exact: 0},
		{name: "padded", value: " 3 ", ok: true, exact: 3 * time.Second},
		{
			name:  "future date",
			value: time.Now().Add(2 * time.Second).UTC().Format(http.TimeFormat),
			ok:    true,
			check: func(t *testing.T, d time.Duration) {
				t.Helper()
				assert.GreaterOrEqual(t, d, time.Duration(0))
				assert.LessOrEqual(t, d, 3*time.Second)
			},
		},
		{
			name:  "past date",
			value: time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat),
			ok:    true,
			exact: 0,
		},
		{name: "empty", value: "", ok: false},
		{name: "negative", value: "-5", ok: false},
		{name: "garbage", value: "bad", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d, ok := restx.ParseRetryAfter(tt.value)
			assert.Equal(t, tt.ok, ok)
			if !tt.ok {
				return
			}
			if tt.check != nil {
				tt.check(t, d)
				return
			}
			assert.Equal(t, tt.exact, d)
		})
	}
}

func TestAPIErrorMessage(t *testing.T) {
	t.Parallel()

	apiErr := restx.DecodeAPIError(
		http.StatusPaymentRequired,
		[]byte(`{"error":"Payment failed","code":"payment_failed","hint":"update card"}`),
		http.Header{"Retry-After": []string{"5"}, "X-Request-Id": []string{"req-1"}},
	)
	assert.Equal(t, 5*time.Second, apiErr.RetryAfter)
	assert.Equal(t, "req-1", apiErr.RequestID)
	message := apiErr.Error()
	assert.True(
		t,
		strings.HasPrefix(message, "api error: status 402 code payment_failed: Payment failed"),
	)
	assert.Contains(t, message, "hint: update card")

	fallback := restx.DecodeAPIError(http.StatusBadGateway, []byte(`<html>`), nil)
	assert.Equal(t, http.StatusText(http.StatusBadGateway), fallback.Message)

	assert.False(t, restx.IsStatus(errors.New("plain"), 0))
	assert.Equal(t, 0, restx.StatusCode(nil))
}

func TestDoSendsFormBodies(t *testing.T) {
	t.Parallel()

	var gotContentType string
	var gotForm url.Values
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		gotForm, err = url.ParseQuery(string(body))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}, nil)

	_, err := client.Do(t.Context(), &restx.Request{
		Method: http.MethodPost,
		Path:   "/token",
		Form:   url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"abc def"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "application/x-www-form-urlencoded", gotContentType)
	assert.Equal(t, "refresh_token", gotForm.Get("grant_type"))
	assert.Equal(t, "abc def", gotForm.Get("refresh_token"))
}

func TestDoRefusesBodyAndForm(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}, nil)

	_, err := client.Do(t.Context(), &restx.Request{
		Method: http.MethodPost,
		Path:   "/token",
		Body:   map[string]string{"a": "b"},
		Form:   url.Values{"a": {"b"}},
	})
	require.ErrorIs(t, err, restx.ErrBodyAndForm)
	assert.Zero(t, calls.Load())
}
