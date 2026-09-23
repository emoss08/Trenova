package completionrouter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stallingStreamServer sends chunks of an SSE reply and then holds the
// connection open without finishing, the way a model still writing does. It
// says when a request has arrived, and lets go when the caller does.
func stallingStreamServer(
	t *testing.T,
	chunks []string,
) (*httptest.Server, <-chan struct{}, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32
	arrived := make(chan struct{}, 16)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		for _, chunk := range chunks {
			payload := `{"model":"stalled-model","choices":[{"delta":{"content":"` + chunk + `"}}]}`
			_, _ = w.Write([]byte("data: " + payload + "\n\n"))
		}
		flusher.Flush()
		arrived <- struct{}{}

		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
	t.Cleanup(server.Close)

	return server, arrived, &calls
}

// charged reports whether the breaker holds any failure against a provider.
func charged(service *Service, id pulid.ID) bool {
	service.health.mu.Lock()
	defer service.health.mu.Unlock()

	_, ok := service.health.entries[id]

	return ok
}

// onlyRows waits for n usage rows and then makes sure no more arrive, since
// each row is written off the request path.
func onlyRows(t *testing.T, usage *fakeUsage, n int) []*aiusage.AIUsageRecord {
	t.Helper()

	rows := usage.recorded(t, n)
	assert.Never(t, func() bool {
		usage.mu.Lock()
		defer usage.mu.Unlock()

		return len(usage.rows) > n
	}, 200*time.Millisecond, 10*time.Millisecond, "one attempt, one row")

	return rows[:n]
}

/*
Stop partway through a reply is a Stop.

A person pressed Stop while the model was halfway through a sentence. The
router took the severed stream for a provider that died partway, kept what had
arrived, and returned it as a finished, truncated reply with no error. The turn
service reads a nil error as Completed, so the reply the person stopped was
saved as one they had read to the end.

The stop is an error wrapping the cancellation, which is what the turn's
status is read from. What had arrived still comes back with it, and the
attempt is billed for the words it wrote, not recorded as free.
*/
func TestStreamChat_AStoppedReplyIsAnErrorNotAFinishedReply(t *testing.T) {
	t.Parallel()

	stalling, _, _ := stallingStreamServer(t, []string{"Sarah Williams - "})
	fallback, _, fallbackCalls := stallingStreamServer(t, []string{"Never asked."})
	first := priced(chatProvider("stopped", stalling.URL, 10), "1", "10")
	service := newTestService(t, first, chatProvider("fallback", fallback.URL, 20))
	service.health = newProviderHealth(nil)
	usage := &fakeUsage{}
	service.usage = usage

	ctx, stop := context.WithCancel(t.Context())
	defer stop()

	result, err := service.StreamChat(ctx, chatRequest(pulid.Nil), func(string) { stop() })

	require.Error(t, err, "a stopped reply is not a finished one")
	assert.ErrorIs(t, err, context.Canceled)
	require.NotNil(t, result, "what the reader was shown travels with the stop")
	assert.True(t, result.Truncated)
	assert.Equal(t, "Sarah Williams - ", result.Text)
	assert.Equal(t, "stalled-model", result.ModelIdentifier)

	assert.Zero(t, fallbackCalls.Load(), "a stopped reply is not handed to the next provider")
	counted := charged(service, first.ID)
	assert.False(t, counted, "a stop is not a provider failure")

	row := onlyRows(t, usage, 1)[0]
	assert.Equal(t, first.ID, row.ProviderID)
	assert.False(t, row.Succeeded)
	assert.Equal(t, "cancelled", row.ErrorClass)
	assert.Equal(t, "stalled-model", row.Model)
	assert.Equal(t, approxTokens(len("Sarah Williams - ")), row.OutputTokens,
		"the words the provider wrote before the stop were billed")
	require.NotNil(t, row.CostUSD, "a stopped attempt on a priced provider is not free")
	assert.True(t, row.CostUSD.IsPositive(), row.CostUSD.String())
}

// Stopped before the first word, the turn ends there. Falling through asked
// every remaining provider on a dead context, and each of those instant
// failures was written down against a provider that had done nothing wrong.
func TestStreamChat_AStopBeforeTheFirstWordAsksNoOtherProvider(t *testing.T) {
	t.Parallel()

	stalling, arrived, _ := stallingStreamServer(t, nil)
	fallback, _, fallbackCalls := stallingStreamServer(t, []string{"Never asked."})
	first := chatProvider("stopped", stalling.URL, 10)
	second := chatProvider("fallback", fallback.URL, 20)
	service := newTestService(t, first, second)
	service.health = newProviderHealth(nil)
	usage := &fakeUsage{}
	service.usage = usage

	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	go func() {
		<-arrived
		stop()
	}()

	result, err := service.StreamChat(ctx, chatRequest(pulid.Nil), func(string) {})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, result, "nothing arrived, so nothing is handed back")
	assert.Zero(t, fallbackCalls.Load())

	row := onlyRows(t, usage, 1)[0]
	assert.Equal(t, first.ID, row.ProviderID, "only the stopped provider was attempted")
	assert.Zero(t, row.OutputTokens, "a provider that wrote nothing billed no output")
	for _, provider := range []*aiprovider.Provider{first, second} {
		counted := charged(service, provider.ID)
		assert.False(t, counted, "%s is not charged for the stop", provider.Name)
	}
}

// A deadline the caller set is the caller's, not the provider's. It surfaced
// as a timeout, which the breaker counts as unavailability, so a few turns
// that outlived their own budget rested a provider that was answering.
func TestStreamChat_ACallersDeadlineDoesNotRestTheProvider(t *testing.T) {
	t.Parallel()

	stalling, _, _ := stallingStreamServer(t, nil)
	fallback, _, fallbackCalls := stallingStreamServer(t, []string{"Never asked."})
	first := chatProvider("slow", stalling.URL, 10)
	service := newTestService(t, first, chatProvider("fallback", fallback.URL, 20))
	service.health = newProviderHealth(nil)
	usage := &fakeUsage{}
	service.usage = usage

	for range breakerThreshold {
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		_, err := service.StreamChat(ctx, chatRequest(pulid.Nil), func(string) {})
		cancel()

		require.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded), "%v", err)
	}

	_, resting := service.health.Resting(first.ID)
	assert.False(t, resting, "the caller's deadline says nothing about the provider")
	counted := charged(service, first.ID)
	assert.False(t, counted)
	assert.Zero(t, fallbackCalls.Load())
	onlyRows(t, usage, breakerThreshold)
}

// A structured call follows the same rule: a caller that has gone is not
// answered by the next provider, and the provider is not blamed for it.
func TestCompleteStructured_AStoppedCallAsksNoOtherProvider(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	arrived := make(chan struct{}, 1)
	stalling := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		arrived <- struct{}{}
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
	t.Cleanup(stalling.Close)
	healthy, healthyCalls := chatServer(t, http.StatusOK, `{"answer":"second"}`)

	first := openAIChatProvider("stopped", stalling.URL, 10)
	service := newTestService(t, first, openAIChatProvider("secondary", healthy.URL, 20))
	service.health = newProviderHealth(nil)
	usage := &fakeUsage{}
	service.usage = usage

	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	go func() {
		<-arrived
		stop()
	}()

	_, err := service.CompleteStructured(ctx, generalRequest())

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.EqualValues(t, 1, calls.Load())
	assert.Zero(t, healthyCalls.Load())
	counted := charged(service, first.ID)
	assert.False(t, counted)

	row := onlyRows(t, usage, 1)[0]
	assert.Equal(t, "cancelled", row.ErrorClass)
}

func TestStopped(t *testing.T) {
	t.Parallel()

	live := t.Context()
	gone, cancel := context.WithCancel(t.Context())
	cancel()

	cause := errors.New("unexpected EOF")
	assert.NoError(t, stopped(gone, nil))
	assert.Same(t, cause, stopped(live, cause), "a live caller's error is left alone")

	wrapped := stopped(gone, cause)
	assert.ErrorIs(t, wrapped, context.Canceled)
	assert.ErrorIs(t, wrapped, cause, "the provider's own account is kept")
	assert.Same(t, context.Canceled, stopped(gone, context.Canceled))
}
