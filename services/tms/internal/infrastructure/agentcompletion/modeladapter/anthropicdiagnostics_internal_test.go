package modeladapter

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	diagnosedReply = `{"id":"msg_02NEXT","model":"claude-opus-5-5","stop_reason":"end_turn",` +
		`"content":[{"type":"text","text":"ok"}],` +
		`"usage":{"input_tokens":12,"output_tokens":1,"cache_read_input_tokens":0,` +
		`"cache_creation_input_tokens":41850},` +
		`"diagnostics":{"cache_miss_reason":{"type":"system_changed","cache_missed_input_tokens":41850}}}`
	pendingReply = `{"id":"msg_02NEXT","model":"claude-opus-5-5","stop_reason":"end_turn",` +
		`"content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1},` +
		`"diagnostics":{"cache_miss_reason":null}}`
	matchedReply = `{"id":"msg_02NEXT","model":"claude-opus-5-5","stop_reason":"end_turn",` +
		`"content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1},` +
		`"diagnostics":null}`
	notFoundStream = "event: message_start\n" +
		`data: {"type":"message_start","message":{"id":"msg_02STREAM","model":"claude-opus-5-5",` +
		`"usage":{"input_tokens":1},"diagnostics":{"cache_miss_reason":{"type":"previous_message_not_found"}}}}` +
		"\n\n" +
		"event: message_delta\n" +
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":1}}` + "\n\n" +
		"event: message_stop\n" +
		`data: {"type":"message_stop"}` + "\n\n"
)

// claudeAPIServer stands in for api.anthropic.com: a call made to the Claude
// API's real address reaches it, and it keeps every request body.
func claudeAPIServer(t *testing.T, reply string) (*http.Client, func() []map[string]any) {
	t.Helper()

	var (
		mu     sync.Mutex
		bodies []map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		if body["stream"] == true {
			w.Header().Set("Content-Type", "text/event-stream")
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)

	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: hostRouter{routes: map[string]*url.URL{"api.anthropic.com": target}},
	}

	return client, func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return append([]map[string]any(nil), bodies...)
	}
}

func diagnosedCall(client *http.Client, baseURL string, messages ...Message) *Call {
	call := callFor(aiprovider.KindAnthropicMessages, baseURL, &Request{Messages: messages})
	call.Provider.ID = pulid.MustNew("aip_")
	call.Provider.Model = "claude-opus-5-5"
	call.Client = client
	call.StreamClient = client
	call.CacheDiagnostics = true

	return call
}

// asked is the diagnostics object a request carried, and whether it carried
// one at all.
func asked(body map[string]any) (map[string]any, bool) {
	diagnostics, ok := body["diagnostics"].(map[string]any)

	return diagnostics, ok
}

/*
The Claude API keeps a fingerprint only of a call that asked, so the first call
of a conversation asks with nothing to compare, and each later one names the
reply before it. That reply is the last one this provider gave: a reply from a
provider that answered in its place is passed over.
*/
func TestAnthropicAdapter_NamesTheReplyBeforeWhenAskingWhyTheCacheMissed(t *testing.T) {
	t.Parallel()

	client, bodies := claudeAPIServer(t, matchedReply)
	first := diagnosedCall(client, "", UserMessage("Where is S1?")...)

	_, err := NewAnthropicAdapter().Complete(t.Context(), first)
	require.NoError(t, err)

	other := pulid.MustNew("aip_")
	history := []Message{
		{Role: RoleUser, Content: "Where is S1?"},
		{Role: RoleAssistant, Content: "Checking.", ReplyID: "msg_01EARLIER", ReplyProvider: first.Provider.ID},
		{Role: RoleUser, Content: "And S2?"},
		{Role: RoleAssistant, Content: "Checking S2.", ReplyID: "chatcmpl-9", ReplyProvider: other},
		{Role: RoleUser, Content: "Thanks."},
	}
	second := diagnosedCall(client, "", history...)
	second.Provider.ID = first.Provider.ID

	_, err = NewAnthropicAdapter().Complete(t.Context(), second)
	require.NoError(t, err)

	sent := bodies()
	require.Len(t, sent, 2)
	opened, ok := asked(sent[0])
	require.True(t, ok, "the first call asks, so the next has something to compare with")
	assert.Contains(t, opened, "previous_message_id")
	assert.Nil(t, opened["previous_message_id"])
	named, ok := asked(sent[1])
	require.True(t, ok)
	assert.Equal(t, "msg_01EARLIER", named["previous_message_id"])
}

// A reply this provider gave without an id, kept before ids were, leaves
// nothing to name; an earlier one is not named in its place, since it is not
// the call before this one.
func TestPreviousReply_NamesOnlyTheLastReplyFromTheSameProvider(t *testing.T) {
	t.Parallel()

	provider := pulid.MustNew("aip_")
	call := diagnosedCall(nil, "",
		Message{Role: RoleAssistant, ReplyID: "msg_01OLD", ReplyProvider: provider},
		Message{Role: RoleUser, Content: "Next."},
		Message{Role: RoleAssistant, Content: "Kept before ids.", ReplyProvider: provider},
		Message{Role: RoleAssistant, Content: "A hand-off this server wrote."},
		Message{Role: RoleUser, Content: "Go on."},
	)
	call.Provider.ID = provider

	assert.Empty(t, previousReply(call))

	call.Request.Messages[2].ReplyID = "msg_01LAST"
	assert.Equal(t, "msg_01LAST", previousReply(call))
}

/*
Diagnostics are asked only when the operator turned them on and only of the
Claude API itself: Claude Platform, the cloud platforms and a gateway speak the
same protocol and do not take the field, and a request one of them refuses
would fail the turn over a diagnostic.
*/
func TestAnthropicAdapter_AsksForDiagnosticsOnlyOfTheClaudeAPIWhenTurnedOn(t *testing.T) {
	t.Parallel()

	gateway, gatewayBodies := filesServer(t, false)
	call := diagnosedCall(&http.Client{Timeout: 5 * time.Second}, gateway.URL, UserMessage("hi")...)
	_, err := NewAnthropicAdapter().Complete(t.Context(), call)
	require.NoError(t, err)
	_, ok := asked(gatewayBodies()[0])
	assert.False(t, ok, "a gateway is not asked")

	client, bodies := claudeAPIServer(t, matchedReply)
	off := diagnosedCall(client, "", UserMessage("hi")...)
	off.CacheDiagnostics = false
	_, err = NewAnthropicAdapter().Complete(t.Context(), off)
	require.NoError(t, err)
	_, ok = asked(bodies()[0])
	assert.False(t, ok, "nothing is asked unless the operator turned it on")
}

func TestAnthropicAdapter_ReadsWhyTheCacheMissed(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		reply     string
		diagnosis *CacheDiagnosis
	}{
		{
			name:      "the system prompt changed",
			reply:     diagnosedReply,
			diagnosis: &CacheDiagnosis{Reason: CacheMissSystemChanged, MissedTokens: 41850},
		},
		{
			name:      "the comparison was still running",
			reply:     pendingReply,
			diagnosis: &CacheDiagnosis{Reason: CacheMissPending},
		},
		{name: "nothing differed", reply: matchedReply},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, _ := claudeAPIServer(t, tc.reply)
			resp, err := NewAnthropicAdapter().Complete(t.Context(), diagnosedCall(client, "", UserMessage("hi")...))

			require.NoError(t, err)
			assert.Equal(t, "msg_02NEXT", resp.MessageID)
			assert.Equal(t, tc.diagnosis, resp.CacheDiagnosis)
			assert.Equal(t, tc.diagnosis != nil && tc.diagnosis.Reason == CacheMissSystemChanged,
				resp.CacheDiagnosis.Changed())
		})
	}
}

// A streamed reply says it on message_start, before any text.
func TestAnthropicAdapter_ReadsWhyTheCacheMissedFromAStream(t *testing.T) {
	t.Parallel()

	client, bodies := claudeAPIServer(t, notFoundStream)
	call := diagnosedCall(client, "", UserMessage("hi")...)

	resp, err := NewAnthropicAdapter().(Streamer).Stream(t.Context(), call, func(string) {})

	require.NoError(t, err)
	assert.Equal(t, "msg_02STREAM", resp.MessageID)
	assert.Equal(t, &CacheDiagnosis{Reason: CacheMissPreviousNotFound}, resp.CacheDiagnosis)
	assert.False(t, resp.CacheDiagnosis.Changed(), "no comparison is not a changed prompt")
	_, ok := asked(bodies()[0])
	assert.True(t, ok)
}
