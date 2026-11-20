package sim

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
)

func (h *harness) call(method, path, body string) (int, []byte) {
	h.t.Helper()
	request, err := http.NewRequestWithContext(
		h.t.Context(),
		method,
		h.url+path,
		strings.NewReader(body),
	)
	if err != nil {
		h.t.Fatalf("request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer response.Body.Close()
	out, _ := io.ReadAll(response.Body)
	return response.StatusCode, out
}

func TestTheControlAPI(t *testing.T) {
	t.Parallel()

	h := newHarness(t, setup{partners: []string{"manufacturer", "carrier"}})
	status, body := h.call(http.MethodGet, "/control/partners", "")
	var listed struct {
		Partners []PartnerInfo `json:"partners"`
	}
	if status != http.StatusOK || sonic.Unmarshal(body, &listed) != nil ||
		len(listed.Partners) != 2 {
		t.Fatalf("partners: %d %s", status, body)
	}
	manufacturer := listed.Partners[0]
	if manufacturer.AS2 == nil || manufacturer.AS2.URL != h.url+"/as2/manufacturer" ||
		!strings.Contains(manufacturer.AS2.CertificatePEM, "BEGIN CERTIFICATE") ||
		manufacturer.Acknowledgment.Delay != "45s" || !manufacturer.Trenova.HasCertificate {
		t.Fatalf("the manufacturer: %+v", manufacturer)
	}
	for _, c := range []struct {
		method, path, body string
		want               int
	}{
		{http.MethodGet, "/control/nobody", "", http.StatusNotFound},
		{http.MethodPost, "/control/manufacturer/trenova", `{"certificatePem":"nope"}`, http.StatusBadRequest},
		{http.MethodPost, "/control/manufacturer/trenova", `{"inboundUrl":"ftp://x"}`, http.StatusBadRequest},
		{http.MethodPost, "/control/manufacturer/send", `{"set":"204"}`, http.StatusBadRequest},
		{http.MethodPost, "/control/manufacturer/send", `{"set":"990","seed":1}`, http.StatusBadRequest},
		{http.MethodPost, "/control/manufacturer/script", `{"answers":[{}]}`, http.StatusConflict},
		{http.MethodGet, "/control/manufacturer/messages?limit=x", "", http.StatusBadRequest},
		{http.MethodGet, "/control/manufacturer/mailbox", "", http.StatusNotFound},
		{http.MethodPost, "/control/reset", `{"partner":"nobody"}`, http.StatusNotFound},
	} {
		if status, body := h.call(c.method, c.path, c.body); status != c.want {
			t.Errorf("%s %s: %d %s, want %d", c.method, c.path, status, body, c.want)
		}
	}

	status, body = h.call(
		http.MethodPost,
		"/control/manufacturer/send",
		`{"set":"204","seed":37,"variation":"change","delta":{"ratePercent":12,"move":{"stop":0,"by":"24h"}}}`,
	)
	var sent struct {
		Messages []journal.Message `json:"messages"`
	}
	if status != http.StatusOK || sonic.Unmarshal(body, &sent) != nil || len(sent.Messages) != 1 ||
		!strings.Contains(sent.Messages[0].Variation, "change rate +12%") {
		t.Fatalf("send: %d %s", status, body)
	}
	h.trenova.next(t)
	status, body = h.call(http.MethodGet, "/control/manufacturer/messages?direction=sent", "")
	if status != http.StatusOK || !bytes.Contains(body, []byte(sent.Messages[0].ID)) {
		t.Fatalf("messages: %d %s", status, body)
	}
	status, body = h.call(http.MethodGet, "/control/manufacturer/conversations", "")
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"key":"KVW-100037"`)) {
		t.Fatalf("conversations: %d %s", status, body)
	}
	if status, _ := h.call(http.MethodPost, "/control/reset", ""); status != http.StatusOK {
		t.Fatalf("reset: %d", status)
	}
	if got := h.sim.Journal().Messages("manufacturer", journal.Filter{}); len(got) != 0 {
		t.Fatalf("reset left %d messages", len(got))
	}
}

func readEvents(t *testing.T, body io.Reader, out chan<- journal.Event) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1<<20), 8<<20)
	for scanner.Scan() {
		line := scanner.Text()
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		var event journal.Event
		if err := sonic.Unmarshal([]byte(data), &event); err != nil {
			t.Errorf("event: %v", err)
			return
		}
		out <- event
	}
	close(out)
}

func (h *harness) stream(ctx context.Context, query, lastID string) <-chan journal.Event {
	h.t.Helper()
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		h.url+"/control/events"+query,
		nil,
	)
	if err != nil {
		h.t.Fatalf("request: %v", err)
	}
	if lastID != "" {
		request.Header.Set("Last-Event-ID", lastID)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		h.t.Fatalf("events: %v", err)
	}
	if response.Header.Get("Content-Type") != "text/event-stream" {
		h.t.Fatalf("events answered %s", response.Header.Get("Content-Type"))
	}
	h.t.Cleanup(func() { _ = response.Body.Close() })
	events := make(chan journal.Event, 64)
	go readEvents(h.t, response.Body, events)
	return events
}

func next(t *testing.T, events <-chan journal.Event, kind string) journal.Event {
	t.Helper()
	deadline := time.After(waitFor)
	for {
		select {
		case event, open := <-events:
			if !open {
				t.Fatalf("the stream ended before %s", kind)
			}
			if event.Type == kind {
				return event
			}
		case <-deadline:
			t.Fatalf("no %s on the stream", kind)
		}
	}
}

func TestTheEventStreamCarriesEveryChangeAndReplays(t *testing.T) {
	t.Parallel()

	h := newHarness(t, setup{partners: []string{"manufacturer", "legacy"}})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	events := h.stream(ctx, "?partner=manufacturer", "")
	if status, _ := h.call(http.MethodGet, "/control/events?partner=nobody", ""); status != http.StatusNotFound {
		t.Fatalf("an unknown partner's stream answered %d", status)
	}
	h.send("legacy", &SendRequest{Set: "204", Seed: 1})
	h.send("manufacturer", &SendRequest{Set: "204", Seed: 1})
	scheduled := next(t, events, "message.scheduled")
	if scheduled.Partner != "manufacturer" || scheduled.Message == nil {
		t.Fatalf("the stream carried %+v", scheduled)
	}
	sent := next(t, events, "message.sent")
	if sent.Message.State != journal.StateDelivered || sent.ID <= scheduled.ID {
		t.Fatalf("message.sent: %+v", sent)
	}
	replayed := h.stream(ctx, "", strconv.FormatUint(scheduled.ID, 10))
	if first := <-replayed; first.ID != scheduled.ID+1 {
		t.Fatalf("the replay began at %d, want %d", first.ID, scheduled.ID+1)
	}
}

func TestARestartKeepsTheJournalAndControlNumbers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	state := filepath.Join(dir, "state.json")
	persist := func(o *Options) {
		o.StatePath = state
		o.IdentityDir = filepath.Join(dir, "identity")
	}
	h := newHarness(t, setup{partners: []string{"manufacturer"}, options: persist})
	sent := h.send("manufacturer", &SendRequest{Set: "204", Seed: 9})[0]
	h.trenova.next(t)
	if err := h.sim.persist(); err != nil {
		t.Fatalf("persist: %v", err)
	}
	before := h.sim.info(h.peerOf("manufacturer"))

	options := Options{
		Partners: openDefinitions(t, []string{"manufacturer"}),
		Clock:    h.clock,
	}
	persist(&options)
	restarted, err := New(options)
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	if _, ok := restarted.Journal().Get(sent.ID); !ok {
		t.Fatal("the restart lost the journal")
	}
	after := restarted.info(restarted.peers["manufacturer"])
	if after.Trenova != before.Trenova || after.AS2.CertificatePEM != before.AS2.CertificatePEM {
		t.Fatalf("the restart lost the connection: %+v vs %+v", after.Trenova, before.Trenova)
	}
	used, _ := strconv.ParseInt(sent.Interchange, 10, 64)
	if next := restarted.peers["manufacturer"].controls.Interchange.Peek(); next <= used {
		t.Fatalf("the restart reuses control number %d (last sent %d)", next, used)
	}
	restarted.closeMailbox()
}
