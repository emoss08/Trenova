package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	samsaratypes "github.com/emoss08/trenova/shared/samsara/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingLimiter struct {
	mu        sync.Mutex
	waits     []string
	penalties []time.Duration
	waitErr   error
}

func (l *recordingLimiter) Wait(_ context.Context, method, path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.waits = append(l.waits, method+" "+path)
	return l.waitErr
}

func (l *recordingLimiter) Penalize(_, _ string, d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.penalties = append(l.penalties, d)
}

func TestDoRetries429HonoringDecimalRetryAfter(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	var firstAt, secondAt atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			firstAt.Store(time.Now().UnixNano())
			w.Header().Set("Retry-After", "0.25")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"Exceeded rate limit.","requestId":"r1"}`))
			return
		}
		secondAt.Store(time.Now().UnixNano())
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	limiter := &recordingLimiter{}
	client, err := New(Config{
		Token:   "token",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
		Retry: RetryConfig{
			Enabled:        true,
			MaxAttempts:    3,
			InitialBackoff: time.Millisecond,
			MaxBackoff:     2 * time.Second,
		},
		Limiter: limiter,
	})
	require.NoError(t, err)

	err = client.Do(t.Context(), Request{Method: http.MethodGet, Path: "/fleet/hos/logs"})
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
	assert.GreaterOrEqual(
		t,
		time.Duration(secondAt.Load()-firstAt.Load()),
		240*time.Millisecond,
	)
	assert.Equal(t, []string{"GET /fleet/hos/logs", "GET /fleet/hos/logs"}, limiter.waits)
	assert.Equal(t, []time.Duration{250 * time.Millisecond}, limiter.penalties)
}

func TestDoStopsAfterMaxAttemptsOn429(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"Exceeded rate limit.","requestId":"r2"}`))
	}))
	defer server.Close()

	limiter := &recordingLimiter{}
	client, err := New(Config{
		Token:   "token",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
		Retry: RetryConfig{
			Enabled:        true,
			MaxAttempts:    3,
			InitialBackoff: time.Millisecond,
			MaxBackoff:     5 * time.Millisecond,
		},
		Limiter: limiter,
	})
	require.NoError(t, err)

	err = client.Do(t.Context(), Request{Method: http.MethodGet, Path: "/fleet/drivers"})
	require.Error(t, err)
	assert.True(t, samsaratypes.IsRateLimit(err))
	assert.Equal(t, int32(3), calls.Load())
	assert.Len(t, limiter.waits, 3)
	assert.Equal(
		t,
		[]time.Duration{
			defaultRateLimitPenalty,
			defaultRateLimitPenalty,
			defaultRateLimitPenalty,
		},
		limiter.penalties,
	)
}

func TestDoCancelsDuringRetryAfterWait(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "30.5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, err := New(Config{
		Token:   "token",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
		Retry: RetryConfig{
			Enabled:        true,
			MaxAttempts:    4,
			InitialBackoff: time.Millisecond,
			MaxBackoff:     time.Minute,
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	started := time.Now()
	err = client.Do(ctx, Request{Method: http.MethodGet, Path: "/fleet/drivers"})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(started), 5*time.Second)
	assert.Equal(t, int32(1), calls.Load())
}

func TestDoReturnsLimiterError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	limiter := &recordingLimiter{waitErr: context.Canceled}
	client, err := New(Config{
		Token:   "token",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
		Retry: RetryConfig{
			Enabled:        true,
			MaxAttempts:    3,
			InitialBackoff: time.Millisecond,
			MaxBackoff:     5 * time.Millisecond,
		},
		Limiter: limiter,
	})
	require.NoError(t, err)

	err = client.Do(t.Context(), Request{Method: http.MethodGet, Path: "/fleet/drivers"})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(0), calls.Load())
	assert.Len(t, limiter.waits, 1)
}
