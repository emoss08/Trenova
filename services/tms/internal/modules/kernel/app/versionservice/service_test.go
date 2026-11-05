package versionservice

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/updateconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func releaseResponse(tag string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(
			strings.NewReader(`{"tag_name":"` + tag + `","published_at":"2026-01-02T03:04:05Z"}`),
		),
	}
}

func newTestService(transport roundTripFunc) *Service {
	s := New(Params{
		Logger:       zap.NewNop(),
		AppConfig:    &appconfig.Config{Version: "1.0.0"},
		UpdateConfig: &updateconfig.Config{Enabled: true, CheckInterval: time.Hour},
	})
	s.httpClient = &http.Client{Transport: transport}

	return s
}

func TestUpdateStatusAsksTheReleaseServerOncePerInterval(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	s := newTestService(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return releaseResponse("v1.1.0"), nil
	})

	for range 5 {
		status, err := s.GetUpdateStatus(t.Context())
		require.NoError(t, err)
		assert.True(t, status.UpdateAvailable)
		assert.Equal(t, "1.1.0", status.LatestVersion)
	}

	assert.Equal(t, int32(1), calls.Load())
}

func TestUpdateStatusChecksAgainOnceTheIntervalHasPassed(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	s := newTestService(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return releaseResponse("v1.1.0"), nil
	})

	_, err := s.GetUpdateStatus(t.Context())
	require.NoError(t, err)

	s.mu.Lock()
	s.nextCheck = time.Now().Add(-time.Second)
	s.mu.Unlock()

	_, err = s.GetUpdateStatus(t.Context())
	require.NoError(t, err)

	assert.Equal(t, int32(2), calls.Load())
}

func TestUpdateStatusWaitsOutAFailedCheck(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	s := newTestService(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("unreachable")
	})

	for range 5 {
		status, err := s.GetUpdateStatus(t.Context())
		require.NoError(t, err)
		assert.False(t, status.UpdateAvailable)
		assert.Equal(t, "1.0.0", status.CurrentVersion)
	}

	assert.Equal(t, int32(1), calls.Load())
	assert.LessOrEqual(t, time.Until(s.nextCheck), failedCheckCooldown)
}

func TestUpdateStatusWaitsOutAMissingRelease(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	s := newTestService(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}, nil
	})

	for range 3 {
		status, err := s.GetUpdateStatus(t.Context())
		require.NoError(t, err)
		assert.False(t, status.UpdateAvailable)
	}

	assert.Equal(t, int32(1), calls.Load())
}

func TestUpdateStatusKeepsTheLastAnswerWhenACheckFails(t *testing.T) {
	t.Parallel()

	var fail atomic.Bool
	s := newTestService(func(*http.Request) (*http.Response, error) {
		if fail.Load() {
			return nil, errors.New("unreachable")
		}
		return releaseResponse("v1.1.0"), nil
	})

	_, err := s.GetUpdateStatus(t.Context())
	require.NoError(t, err)

	fail.Store(true)
	s.mu.Lock()
	s.nextCheck = time.Now().Add(-time.Second)
	s.mu.Unlock()

	status, err := s.GetUpdateStatus(t.Context())
	require.NoError(t, err)
	assert.True(t, status.UpdateAvailable)
	assert.Equal(t, "1.1.0", status.LatestVersion)
}

func TestConcurrentUpdateStatusReadsShareOneCheck(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	release := make(chan struct{})
	s := newTestService(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		<-release
		return releaseResponse("v1.1.0"), nil
	})

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			status, err := s.GetUpdateStatus(t.Context())
			assert.NoError(t, err)
			assert.True(t, status.UpdateAvailable)
		})
	}

	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	assert.Equal(t, int32(1), calls.Load())
}

func TestACancelledCallerDoesNotFailTheSharedCheck(t *testing.T) {
	t.Parallel()

	s := newTestService(func(r *http.Request) (*http.Response, error) {
		if err := r.Context().Err(); err != nil {
			return nil, err
		}
		return releaseResponse("v1.1.0"), nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	status, err := s.GetUpdateStatus(ctx)
	require.NoError(t, err)
	assert.True(t, status.UpdateAvailable)
}

func TestUpdateStatusMakesNoCallWhenOfflineOrDisabled(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return releaseResponse("v1.1.0"), nil
	})

	offline := newTestService(transport)
	offline.updateConfig.OfflineMode = true
	status, err := offline.GetUpdateStatus(t.Context())
	require.NoError(t, err)
	assert.False(t, status.UpdateAvailable)

	disabled := newTestService(transport)
	disabled.updateConfig.Enabled = false
	status, err = disabled.GetUpdateStatus(t.Context())
	require.NoError(t, err)
	assert.False(t, status.UpdateAvailable)

	assert.Zero(t, calls.Load())
}
