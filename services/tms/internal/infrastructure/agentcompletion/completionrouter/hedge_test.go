package completionrouter

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// slowServer answers like chatServer after delay, or not at all when the
// caller gives up first. It reads the request first: the server only notices a
// client that hung up once the body is read, and a test then ends as soon as
// the router cancels the attempt rather than when the delay runs out.
func slowServer(t *testing.T, delay time.Duration, content string) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	fast, _ := chatServer(t, http.StatusOK, content)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}
		proxied, err := http.Post(fast.URL, "application/json", bytes.NewReader(body))
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		defer proxied.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(w, proxied.Body)
	}))
	t.Cleanup(server.Close)

	return server, &calls
}

/*
DB-003: the scope guard's three seconds were spent waiting on the first
provider whenever it was slow, and the guard let the question through
unclassified though a second provider was configured. A hedged call asks the
next as well once the first has not answered, and takes whichever answers.
*/
func TestCompleteStructured_HedgedCallTakesTheNextProviderWhenTheFirstIsSlow(t *testing.T) {
	t.Parallel()

	slow, slowCalls := slowServer(t, time.Minute, `{"answer":"first"}`)
	quick, quickCalls := chatServer(t, http.StatusOK, `{"answer":"second"}`)
	svc := newTestService(t,
		openAIChatProvider("slow", slow.URL, 10),
		openAIChatProvider("quick", quick.URL, 20),
	)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	request := generalRequest()
	request.HedgeAfter = 50 * time.Millisecond

	started := time.Now()
	result, err := svc.CompleteStructured(ctx, request)
	require.NoError(t, err)

	assert.JSONEq(t, `{"answer":"second"}`, result.Text)
	assert.Less(t, time.Since(started), 2*time.Second, "the slow provider held the call")
	assert.Equal(t, int32(1), slowCalls.Load())
	assert.Equal(t, int32(1), quickCalls.Load())
}

// A provider that answers before the hedge is asked alone.
func TestCompleteStructured_HedgedCallAsksOnlyTheFirstWhenItAnswersInTime(t *testing.T) {
	t.Parallel()

	first, firstCalls := chatServer(t, http.StatusOK, `{"answer":"first"}`)
	second, secondCalls := chatServer(t, http.StatusOK, `{"answer":"second"}`)
	svc := newTestService(t,
		openAIChatProvider("first", first.URL, 10),
		openAIChatProvider("second", second.URL, 20),
	)

	request := generalRequest()
	request.HedgeAfter = 2 * time.Second

	result, err := svc.CompleteStructured(t.Context(), request)
	require.NoError(t, err)

	assert.JSONEq(t, `{"answer":"first"}`, result.Text)
	assert.Equal(t, int32(1), firstCalls.Load())
	assert.Equal(t, int32(0), secondCalls.Load(), "a hedge is only for a slow first provider")
}

// A first provider that fails outright hands the call on at once rather than
// after the hedge.
func TestCompleteStructured_HedgedCallMovesOnAtOnceWhenTheFirstFails(t *testing.T) {
	t.Parallel()

	failing, _ := chatServer(t, http.StatusBadRequest, "")
	second, secondCalls := chatServer(t, http.StatusOK, `{"answer":"second"}`)
	svc := newTestService(t,
		openAIChatProvider("failing", failing.URL, 10),
		openAIChatProvider("second", second.URL, 20),
	)

	request := generalRequest()
	request.HedgeAfter = time.Minute

	started := time.Now()
	result, err := svc.CompleteStructured(t.Context(), request)
	require.NoError(t, err)

	assert.JSONEq(t, `{"answer":"second"}`, result.Text)
	assert.Less(t, time.Since(started), 10*time.Second)
	assert.Equal(t, int32(1), secondCalls.Load())
}

// Every provider slow: the call ends at the caller's deadline, as before, and
// the caller decides what that means.
func TestCompleteStructured_HedgedCallEndsAtTheDeadlineWhenEveryProviderIsSlow(t *testing.T) {
	t.Parallel()

	first, _ := slowServer(t, time.Minute, `{"answer":"first"}`)
	second, _ := slowServer(t, time.Minute, `{"answer":"second"}`)
	svc := newTestService(t,
		openAIChatProvider("first", first.URL, 10),
		openAIChatProvider("second", second.URL, 20),
	)

	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	request := generalRequest()
	request.HedgeAfter = 100 * time.Millisecond

	started := time.Now()
	_, err := svc.CompleteStructured(ctx, request)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(started), 3*time.Second)
}
