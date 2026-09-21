package completionrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// busyThenOK answers the first n requests with the given status and a
// Retry-After, then answers properly.
func busyThenOK(t *testing.T, failures int, status int, retryAfter string) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if int(n) <= failures {
			if retryAfter != "" {
				w.Header().Set("Retry-After", retryAfter)
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"message":"The model is overloaded. Please try again later."}}`))

			return
		}
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"Answered."}}]}`))
	}))
	t.Cleanup(server.Close)

	return server, &calls
}

func recordingPauses(service *Service) *[]time.Duration {
	waits := &[]time.Duration{}
	service.pause = func(_ context.Context, wait time.Duration) error {
		*waits = append(*waits, wait)

		return nil
	}

	return waits
}

// A busy provider is asked again, after the wait it asked for, before the
// turn is given up. Two 503s in a row used to end a pinned turn with "ask
// again", when asking again a few seconds later is exactly what a 503 means.
func TestCompleteChat_KeepsAskingABusyProviderAndHonoursItsRetryAfter(t *testing.T) {
	t.Parallel()

	server, calls := busyThenOK(t, 2, http.StatusServiceUnavailable, "2")
	provider := chatProvider("gemini", server.URL, 10)
	service := newTestService(t, provider)
	waits := recordingPauses(service)

	var notices []serviceports.ChatRetryNotice
	request := chatRequest(provider.ID)
	request.PinPreferred = true
	request.RetrySink = func(notice serviceports.ChatRetryNotice) { notices = append(notices, notice) }

	result, err := service.CompleteChat(t.Context(), request)

	require.NoError(t, err)
	assert.Equal(t, "Answered.", result.Text)
	assert.EqualValues(t, 3, calls.Load())
	assert.Equal(t, []time.Duration{2 * time.Second, 2 * time.Second}, *waits,
		"the provider's Retry-After outranks the shorter backoff")
	require.Len(t, notices, 2)
	assert.Equal(t, serviceports.RetryKindBusy, notices[0].Kind)
	assert.Equal(t, 2, notices[0].WaitSeconds)
	assert.Equal(t, "gemini", notices[0].Provider)
}

// The wait is bounded whatever the provider asks. A provider asking for a
// minute is asked again after the cap, and the turn is given up once the
// budget for one provider is spent, rather than a person watching a
// spinner for as long as the provider likes.
func TestCompleteChat_BoundsTheWaitOnABusyProvider(t *testing.T) {
	t.Parallel()

	server, calls := busyThenOK(t, 10, http.StatusTooManyRequests, "60")
	provider := chatProvider("gemini", server.URL, 10)
	service := newTestService(t, provider)
	waits := recordingPauses(service)

	_, err := service.CompleteChat(t.Context(), chatRequest(pulid.Nil))

	require.Error(t, err)
	assert.Equal(t, []time.Duration{maxRetryWait}, *waits, "one capped wait fits the budget; a second would not")
	assert.EqualValues(t, 2, calls.Load())
}

// A request the provider refused is not asked again: a 400 is the request's
// fault and the same request gets the same answer.
func TestCompleteChat_DoesNotRetryARejectedRequest(t *testing.T) {
	t.Parallel()

	server, calls := busyThenOK(t, 10, http.StatusBadRequest, "")
	provider := chatProvider("gemini", server.URL, 10)
	service := newTestService(t, provider)
	waits := recordingPauses(service)

	_, err := service.CompleteChat(t.Context(), chatRequest(pulid.Nil))

	require.Error(t, err)
	assert.Empty(t, *waits)
	assert.EqualValues(t, 1, calls.Load())
}

func TestRetryWait_UsesTheConfiguredAttemptsForOtherRetryableFailures(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	service.ai.MaxRetries = 2

	wait, again := service.retryWait(context.DeadlineExceeded, 0, 0)
	assert.True(t, again, "a timeout is the provider being busy")
	assert.Equal(t, retryDelay(0), wait)

	_, again = service.retryWait(assert.AnError, 1, 0)
	assert.False(t, again, "an unclassified failure gets only the configured attempts")
	wait, again = service.retryWait(assert.AnError, 0, 0)
	assert.True(t, again)
	assert.Equal(t, retryDelay(0), wait)
}
