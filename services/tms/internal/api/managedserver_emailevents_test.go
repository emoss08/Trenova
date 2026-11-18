package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/modules/workspace/app/emailservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const (
	eventA         = "0192f3a4-5b6c-7d8e-9f01-23456789aaaa"
	eventB         = "0192f3a4-5b6c-7d8e-9f01-23456789bbbb"
	eventRecipient = "dispatcher@acme.example"
)

type fakeEmailEvents struct {
	mu   sync.Mutex
	seen []*emailservice.RelayedSystemEvent
	fail map[string]bool
}

func (f *fakeEmailEvents) HandleRelayedSystemEvent(
	_ context.Context,
	relayed *emailservice.RelayedSystemEvent,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen = append(f.seen, relayed)
	if f.fail[relayed.ID] {
		return errors.New("the database is down")
	}
	return nil
}

type emailEventsRig struct {
	handler http.Handler
	events  *fakeEmailEvents
	logs    *observer.ObservedLogs
}

func newEmailEventsRig(fail ...string) *emailEventsRig {
	core, logs := observer.New(zap.DebugLevel)
	events := &fakeEmailEvents{fail: map[string]bool{}}
	for _, id := range fail {
		events.fail[id] = true
	}
	return &emailEventsRig{
		handler: api.NewManagedHandler(&api.ManagedHandlerParams{
			Usage:             &fakeManagedUsage{},
			Features:          newFeatures(&fakeFeatureRepo{}),
			Invites:           &fakeInvites{},
			Token:             managedToken,
			Logger:            zap.New(core),
			SystemEmailEvents: events,
		}),
		events: events,
		logs:   logs,
	}
}

func (r *emailEventsRig) post(t *testing.T, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/managed/v1/system-email/events",
		strings.NewReader(body),
	)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.handler.ServeHTTP(rec, req)
	return rec
}

func bouncePayload() string {
	return `{"RecordType":"Bounce","ID":883953,"Type":"HardBounce","MessageID":"pm-1","Email":"` +
		eventRecipient + `"}`
}

func eventsBody(t *testing.T, ids ...string) string {
	t.Helper()
	events := make([]map[string]string, 0, len(ids))
	for _, id := range ids {
		events = append(events, map[string]string{
			"id":       id,
			"provider": "Postmark",
			"payload":  bouncePayload(),
		})
	}
	body, err := sonic.Marshal(map[string]any{"events": events})
	require.NoError(t, err)
	return string(body)
}

func handledIDs(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	var out struct {
		Handled []string `json:"handled"`
	}
	require.NoError(t, sonic.Unmarshal(rec.Body.Bytes(), &out))
	return out.Handled
}

func (r *emailEventsRig) assertNoPayloadLogged(t *testing.T) {
	t.Helper()
	for _, entry := range r.logs.All() {
		line := entry.Message + " " + fmt.Sprint(entry.ContextMap())
		assert.NotContains(t, line, eventRecipient)
		assert.NotContains(t, line, "HardBounce")
	}
}

func TestRelayedEmailEventsAreHandedToTheEmailServiceVerbatim(t *testing.T) {
	t.Parallel()

	r := newEmailEventsRig()
	rec := r.post(t, managedToken, eventsBody(t, eventA, eventB))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []string{eventA, eventB}, handledIDs(t, rec))
	require.Len(t, r.events.seen, 2)
	assert.Equal(t, eventA, r.events.seen[0].ID)
	assert.Equal(t, "Postmark", string(r.events.seen[0].Provider))
	assert.Equal(t, bouncePayload(), string(r.events.seen[0].Payload))
	r.assertNoPayloadLogged(t)
}

func TestARelayedEmailEventThatFailsIsLeftOutForTheOperatorToRetry(t *testing.T) {
	t.Parallel()

	r := newEmailEventsRig(eventA)
	rec := r.post(t, managedToken, eventsBody(t, eventA, eventB))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []string{eventB}, handledIDs(t, rec))
	r.assertNoPayloadLogged(t)
}

func TestARelayedEmailEventWithAnOversizedPayloadIsDroppedAsHandled(t *testing.T) {
	t.Parallel()

	r := newEmailEventsRig()
	body, err := sonic.Marshal(map[string]any{"events": []map[string]string{{
		"id":       eventA,
		"provider": "Postmark",
		"payload":  strings.Repeat("a", 65537),
	}}})
	require.NoError(t, err)
	rec := r.post(t, managedToken, string(body))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []string{eventA}, handledIDs(t, rec))
	assert.Empty(t, r.events.seen)
}

func TestRelayedEmailEventsRefuseAMalformedBatch(t *testing.T) {
	t.Parallel()

	tooMany := make([]string, 51)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("0192f3a4-5b6c-7d8e-9f01-%012d", i)
	}
	bodies := map[string]string{
		"not json":      `{"events":`,
		"no events":     `{"events":[]}`,
		"too many":      eventsBody(t, tooMany...),
		"not a uuid":    eventsBody(t, "evt_1"),
		"uppercase id":  eventsBody(t, strings.ToUpper(eventA)),
		"a repeated id": eventsBody(t, eventA, eventA),
		"unknown field": strings.Replace(eventsBody(t, eventA), `"id"`, `"receivedAt":"x","id"`, 1),
	}
	for name, body := range bodies {
		r := newEmailEventsRig()
		rec := r.post(t, managedToken, body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, name)
		assert.Empty(t, r.events.seen, name)
	}
}

func TestRelayedEmailEventsRefuseAnOversizedBody(t *testing.T) {
	t.Parallel()

	r := newEmailEventsRig()
	rec := r.post(t, managedToken, `{"events":[{"id":"`+strings.Repeat("a", 9<<20)+`"}]}`)
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Empty(t, r.events.seen)
}

func TestRelayedEmailEventsRefuseACallerWithoutTheManagedToken(t *testing.T) {
	t.Parallel()

	r := newEmailEventsRig()
	for _, token := range []string{"", "0123456789abcdef0123456789abcdeX"} {
		rec := r.post(t, token, eventsBody(t, eventA))
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	}
	assert.Empty(t, r.events.seen)
}
