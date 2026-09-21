package completionrouter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dyingStreamServer sends some of an SSE reply and then drops the connection,
// which is what a rate-limited or overloaded endpoint actually does.
func dyingStreamServer(t *testing.T, chunks []string) *httptest.Server {
	t.Helper()

	return dyingNamedModelServer(t, "", chunks)
}

// dyingNamedModelServer is a dying stream whose chunks name the model that
// served them, as a real endpoint's do.
func dyingNamedModelServer(t *testing.T, model string, chunks []string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		for _, chunk := range chunks {
			payload := `{"model":"` + model + `","choices":[{"delta":{"content":"` + chunk + `"}}]}`
			_, _ = w.Write([]byte("data: " + payload + "\n\n"))
			flusher.Flush()
		}

		// No [DONE], no close frame: the stream simply stops.
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, err := hijacker.Hijack()
		if err == nil {
			_ = conn.Close()
		}
	}))
}

/*
A reply that stops partway is still a reply.

Asked which drivers hold a current hazmat endorsement, a model listed both
correctly and was most of the way through explaining why they were
NonCompliant — "Sarah's medical card expires in" — when the stream died. The
reader watched a good answer appear and then be replaced by "the assistant
could not finish this reply". Everything that had arrived was discarded.

The words that arrived are the ones that answered the question. They are kept,
the result says it was cut off, and the turn ends rather than erroring.
*/
func TestStreamChat_KeepsWhatArrivedWhenTheStreamDies(t *testing.T) {
	t.Parallel()

	server := dyingStreamServer(t, []string{
		"Sarah Williams - endorsement X. ",
		"Jane Doe - endorsement H. ",
		"Both are NonCompliant because",
	})
	provider := chatProvider("flaky", server.URL, 10)
	service := newTestService(t, provider)

	// Each retry tells the reader to discard what they saw, so what they are
	// looking at when the turn ends is only the last attempt's text.
	var streamed strings.Builder
	request := chatRequest(pulid.Nil)
	request.RetrySink = func(serviceports.ChatRetryNotice) { streamed.Reset() }
	result, err := service.StreamChat(t.Context(), request, func(delta string) {
		streamed.WriteString(delta)
	})

	require.NoError(t, err, "a reply that stopped partway is not an error the reader should see")
	require.NotNil(t, result)

	assert.True(t, result.Truncated)
	assert.Contains(t, result.Text, "Sarah Williams")
	assert.Contains(t, result.Text, "Jane Doe")
	assert.Equal(t, streamed.String(), result.Text,
		"what was saved and what the reader watched have to be the same text")
}

// A stream that dies before saying anything has nothing worth keeping, so the
// usual fallthrough to the next provider still applies.
func TestStreamChat_FallsThroughWhenNothingWasSaid(t *testing.T) {
	t.Parallel()

	dead := dyingStreamServer(t, nil)
	// A streaming server, not the plain-JSON one: the router streams whenever
	// the adapter can, so a fake that only answers the non-streaming shape
	// parses as an empty reply rather than as the fallthrough's success.
	good := completeStreamServer(t, "Answered by the second provider.")

	service := newTestService(t,
		chatProvider("dies-first", dead.URL, 10),
		chatProvider("answers", good.URL, 20),
	)

	result, err := service.StreamChat(t.Context(), chatRequest(pulid.Nil), func(string) {})
	require.NoError(t, err)

	assert.False(t, result.Truncated)
	assert.Contains(t, result.Text, "second provider")
}

// completeStreamServer sends a whole reply and terminates it properly, which is
// what the fallthrough needs to land on.
func completeStreamServer(t *testing.T, content string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		payload := `{"choices":[{"delta":{"content":"` + content + `"}}]}`
		_, _ = w.Write([]byte("data: " + payload + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
}

/*
A reply that dies partway is started over on the next provider.

The reader watched a free-tier model answer half a question and stop, and
was left with "cut off, ask again". With another provider configured there
is no reason to leave them there: the turn starts over on it, the reader is
told the reply is restarting so the half they saw is discarded, and what
they end with is one whole answer from one model.
*/
func TestStreamChat_StartsOverOnTheNextProviderWhenAReplyDies(t *testing.T) {
	t.Parallel()

	dying := dyingStreamServer(t, []string{"Sarah Williams - ", "endorsement X."})
	good := completeStreamServer(t, "Sarah Williams holds endorsement X; Jane Doe holds H.")
	service := newTestService(t,
		chatProvider("dies-midway", dying.URL, 10),
		chatProvider("finishes", good.URL, 20),
	)

	var notices []serviceports.ChatRetryNotice
	request := chatRequest(pulid.Nil)
	request.RetrySink = func(notice serviceports.ChatRetryNotice) { notices = append(notices, notice) }

	var streamed strings.Builder
	result, err := service.StreamChat(t.Context(), request, func(delta string) { streamed.WriteString(delta) })
	require.NoError(t, err)

	assert.False(t, result.Truncated)
	assert.Equal(t, "Sarah Williams holds endorsement X; Jane Doe holds H.", result.Text)
	require.Len(t, notices, 1)
	assert.Equal(t, 1, notices[0].Attempt)
	assert.Equal(t, "finishes", notices[0].Provider)
	assert.Contains(t, streamed.String(), "endorsement X.", "the partial text did reach the sink before the retry")
}

// With nowhere else to go, the retries are spent on the same provider and
// then the partial reply is kept as before rather than lost.
func TestStreamChat_KeepsThePartialReplyOnceTheRetriesAreSpent(t *testing.T) {
	t.Parallel()

	dying := dyingStreamServer(t, []string{"Half an answer"})
	service := newTestService(t, chatProvider("only-one", dying.URL, 10))

	retries := 0
	request := chatRequest(pulid.Nil)
	request.RetrySink = func(serviceports.ChatRetryNotice) { retries++ }

	result, err := service.StreamChat(t.Context(), request, func(string) {})
	require.NoError(t, err)

	assert.True(t, result.Truncated)
	assert.Equal(t, "Half an answer", result.Text)
	assert.Equal(t, maxMidReplyRetries, retries)
}

// A model the person picked is the model they get. Pinned, the turn does
// not fall through to another provider when that one fails outright.
func TestStreamChat_APinnedPreferenceNeverFallsThrough(t *testing.T) {
	t.Parallel()

	dead := dyingStreamServer(t, nil)
	good := completeStreamServer(t, "Answered by the other provider.")
	pinned := chatProvider("chosen", dead.URL, 20)
	service := newTestService(t, chatProvider("fallback", good.URL, 10), pinned)

	request := chatRequest(pinned.ID)
	request.PinPreferred = true

	_, err := service.StreamChat(t.Context(), request, func(string) {})
	require.Error(t, err, "the chosen model failed; nothing else may answer in its name")

	request.PinPreferred = false
	result, err := service.StreamChat(t.Context(), request, func(string) {})
	require.NoError(t, err)
	assert.Contains(t, result.Text, "other provider")
}

// A pin on a provider that is no longer offered degrades to the order, so a
// deleted choice never strands a conversation.
func TestStreamChat_APinOnAMissingProviderUsesTheOrder(t *testing.T) {
	t.Parallel()

	good := completeStreamServer(t, "Answered.")
	service := newTestService(t, chatProvider("only", good.URL, 10))

	request := chatRequest(pulid.MustNew("aiprv_"))
	request.PinPreferred = true

	result, err := service.StreamChat(t.Context(), request, func(string) {})
	require.NoError(t, err)
	assert.Equal(t, "Answered.", result.Text)
}

// A cut-off reply is recorded under the model that actually served it. The
// configured identifier is an alias on some endpoints, and a turn logged under
// the alias when every finished turn is logged under the served name made the
// usage log disagree with itself about which model a person was talking to.
func TestStreamChat_ACutOffReplyNamesTheModelThatServedIt(t *testing.T) {
	t.Parallel()

	server := dyingNamedModelServer(t, "served-model-2026-09", []string{"Sarah Williams - "})
	provider := chatProvider("flaky", server.URL, 10)
	service := newTestService(t, provider)

	result, err := service.StreamChat(t.Context(), chatRequest(pulid.Nil), func(string) {})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Truncated)
	assert.Equal(t, "served-model-2026-09", result.ModelIdentifier)
}
