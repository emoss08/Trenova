package sim

import (
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/ladder"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
)

func (h *harness) awaitRun(kind string) journal.Event {
	h.t.Helper()
	deadline := time.After(waitFor)
	for {
		select {
		case event, open := <-h.events.Events():
			if !open {
				h.t.Fatalf("the event stream closed while waiting for %s", kind)
			}
			if event.Type == kind && event.Run != nil {
				return event
			}
		case <-deadline:
			h.t.Fatalf("no %s event arrived", kind)
		}
	}
}

func (h *harness) decode(method, path, body string, want int, out any) {
	h.t.Helper()
	status, raw := h.call(method, path, body)
	if status != want {
		h.t.Fatalf("%s %s: %d %s, want %d", method, path, status, raw, want)
	}
	if out != nil {
		if err := sonic.Unmarshal(raw, out); err != nil {
			h.t.Fatalf("%s %s: %v", method, path, err)
		}
	}
}

func TestARunsProgressReachesTheConversationLadder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	state := filepath.Join(dir, "state.json")
	persist := func(o *Options) {
		o.StatePath = state
		o.IdentityDir = filepath.Join(dir, "identity")
	}
	h := newHarness(t, setup{partners: []string{"manufacturer"}, options: persist})
	sent := h.send("manufacturer", &SendRequest{Set: "204", Seed: 37})[0]
	h.trenova.next(t)
	key := sent.Keys[0]
	group := sent.Groups[0]
	ack := trenovaX12(h.peerOf("manufacturer").def, false, 912, "FA", "004010",
		fmt.Sprintf("997~AK1*SM*%s~AK2*204*%s~AK5*A~AK9*A*1*1*1", group.Control,
			group.Transactions[0].Control))
	if _, err := h.post(&as2Send{
		partner: "manufacturer", signed: true, encrypted: true, payload: ack,
	}); err != nil {
		t.Fatalf("post the 997: %v", err)
	}
	h.await("ack.received", func(m *journal.Message) bool { return m.ID == sent.ID })
	answered := h.message(sent.ID)
	if by := answered.Documents[0].AnsweredBy; by == nil || by.Code != "A" || by.Set != "997" {
		t.Fatalf("the 204 does not name the 997 that answers it: %+v", by)
	}
	ackMessage := h.message(answered.Documents[0].AnsweredBy.MessageID)
	answer := ackMessage.Documents[0]
	if len(answer.Answers) != 1 || answer.Answers[0].Number != answered.Documents[0].Number ||
		answer.Keys[0] != key || answer.Title != "Accepted 204 "+trimmed(group.Control) {
		t.Fatalf("the 997 document: %+v", answer)
	}

	var started runlog.Summary
	h.decode(http.MethodPost, "/control/runs",
		`{"kind":"run","number":3,"api":"http://localhost:8080/api/v1"}`, http.StatusCreated,
		&started)
	h.awaitRun("run.started")
	base := "/control/runs/" + started.ID
	h.decode(http.MethodPost, base+"/scenarios", `{"partner":"manufacturer","slug":"held",
		"name":"a rate change is held","status":"running","replay":"edi-sim run --run 3",
		"steps":[{"index":0,"kind":"send","text":"send 204 original as t1","status":"pending"},
		{"index":1,"kind":"expect","text":"expect flow t1 held","status":"pending"}]}`,
		http.StatusOK, nil)
	h.decode(http.MethodPost, base+"/steps", fmt.Sprintf(`{"partner":"manufacturer",
		"scenario":"held","step":{"index":0,"kind":"send","text":"send 204 original as t1",
		"status":"passed","matches":[{"messageId":%q,"key":%q}]}}`, sent.ID, key),
		http.StatusOK, nil)
	h.decode(http.MethodPost, base+"/steps", fmt.Sprintf(`{"partner":"manufacturer",
		"scenario":"held","step":{"index":1,"kind":"expect","text":"expect flow t1 held",
		"status":"passed","matches":[{"messageId":%q}],"events":[{"key":%q,
		"at":"2026-03-02T15:01:00Z","label":"Flow held","detail":"changed the rate",
		"tone":"hold"}]}}`, ackMessage.ID, key), http.StatusOK, nil)
	if event := h.awaitRun("run.step"); event.Partner != "manufacturer" ||
		event.Run.RunID != started.ID || event.Run.Keys[0] != key {
		t.Fatalf("the step event: %+v", event)
	}

	var listed struct {
		Conversations []struct {
			Key      string       `json:"key"`
			Scenario *scenarioRef `json:"scenario"`
		} `json:"conversations"`
	}
	h.decode(http.MethodGet, "/control/manufacturer/conversations", "", http.StatusOK, &listed)
	if len(listed.Conversations) != 1 || listed.Conversations[0].Scenario == nil ||
		!listed.Conversations[0].Scenario.Waiting ||
		listed.Conversations[0].Scenario.Name != "a rate change is held" {
		t.Fatalf("conversations: %+v", listed.Conversations)
	}

	var detail ConversationDetail
	h.decode(http.MethodGet, "/control/manufacturer/conversations/"+key, "", http.StatusOK,
		&detail)
	if detail.Scenario == nil || len(detail.Scenario.Steps) != 2 || detail.Run == nil ||
		detail.Run.Number != 3 || len(detail.Messages) != 2 {
		t.Fatalf("the detail: run %+v scenario %+v messages %d", detail.Run, detail.Scenario,
			len(detail.Messages))
	}
	if len(detail.Rows) != 3 {
		t.Fatalf("rows: %+v", detail.Rows)
	}
	tender, reply, held := detail.Rows[0], detail.Rows[1], detail.Rows[2]
	if tender.Set != "204" || tender.From != ladder.FromPartner ||
		tender.Verdict != ladder.VerdictRead || tender.Expect != ladder.ToneOK ||
		tender.Title != "Original tender" {
		t.Fatalf("the 204 row: %+v", tender)
	}
	if reply.Set != "997" || !reply.Ack || reply.From != ladder.FromTrenova ||
		reply.Expect != ladder.ToneOK || reply.Control != "000000912" {
		t.Fatalf("the 997 row: %+v", reply)
	}
	if held.Kind != ladder.KindEvent || held.Tone != runlog.ToneHold ||
		held.Expect != ladder.ToneOK || held.Label != "Flow held" {
		t.Fatalf("the event row: %+v", held)
	}

	var partners struct {
		Partners []PartnerInfo `json:"partners"`
	}
	h.decode(http.MethodGet, "/control/partners", "", http.StatusOK, &partners)
	if tally := partners.Partners[0].Run; tally == nil || tally.Waiting != 1 {
		t.Fatalf("the partner's tally: %+v", tally)
	}
	var runs struct {
		Runs    []runlog.Summary `json:"runs"`
		Current *runlog.Summary  `json:"current"`
	}
	h.decode(http.MethodGet, "/control/runs", "", http.StatusOK, &runs)
	if runs.Current == nil || runs.Current.ID != started.ID || runs.Current.Tally.Waiting != 1 {
		t.Fatalf("runs: %+v", runs)
	}
	h.decode(http.MethodPost, base, `{"status":"failed"}`, http.StatusOK, nil)
	var full runlog.Run
	h.decode(http.MethodGet, base, "", http.StatusOK, &full)
	if full.Status != runlog.StatusFailed || full.FinishedAt == nil || len(full.Scenarios) != 1 {
		t.Fatalf("the finished run: %+v", full)
	}

	for _, c := range []struct {
		method, path, body string
		want               int
	}{
		{http.MethodPost, "/control/runs", `{"kind":"walk"}`, http.StatusBadRequest},
		{http.MethodGet, "/control/runs/r999", "", http.StatusNotFound},
		{http.MethodPost, "/control/runs/r999/steps", `{"partner":"manufacturer","scenario":"held","step":{"index":0,"status":"passed"}}`, http.StatusNotFound},
		{http.MethodPost, base + "/steps", `{"partner":"manufacturer","scenario":"nope","step":{"index":0,"status":"passed"}}`, http.StatusNotFound},
		{http.MethodPost, base + "/steps", `{"partner":"manufacturer","scenario":"held","step":{"index":0,"status":"great"}}`, http.StatusBadRequest},
		{http.MethodPost, base + "/scenarios", `{"partner":"nobody","slug":"x","status":"running"}`, http.StatusNotFound},
		{http.MethodPost, base, `{"status":"running"}`, http.StatusBadRequest},
		{http.MethodGet, "/control/manufacturer/conversations/NOPE-1", "", http.StatusNotFound},
		{http.MethodGet, "/control/nobody/conversations/NOPE-1", "", http.StatusNotFound},
	} {
		if status, body := h.call(c.method, c.path, c.body); status != c.want {
			t.Errorf("%s %s: %d %s, want %d", c.method, c.path, status, body, c.want)
		}
	}

	if err := h.sim.persist(); err != nil {
		t.Fatalf("persist: %v", err)
	}
	options := Options{Partners: openDefinitions(t, []string{"manufacturer"}), Clock: h.clock}
	persist(&options)
	restarted, err := New(options)
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	defer restarted.closeMailbox()
	if current, ok := restarted.Runs().Current(); !ok || current.ID != started.ID {
		t.Fatalf("the restart lost the run: %+v", current)
	}
	again, err := restarted.Conversation("manufacturer", key)
	if err != nil || again.Scenario == nil || again.Rows[0].Number != tender.Number {
		t.Fatalf("the restarted conversation: %v %+v", err, again.Rows)
	}
}

func TestResetForgetsThePartnersRuns(t *testing.T) {
	t.Parallel()

	h := newHarness(t, setup{partners: []string{"manufacturer"}})
	var started runlog.Summary
	h.decode(http.MethodPost, "/control/runs", `{"kind":"soak","number":4}`,
		http.StatusCreated, &started)
	h.decode(http.MethodPost, "/control/runs/"+started.ID+"/scenarios",
		`{"partner":"manufacturer","slug":"soak-1","status":"passed","keys":["KVW-1"]}`,
		http.StatusOK, nil)
	if _, ok := h.sim.Runs().For("manufacturer", "KVW-1"); !ok {
		t.Fatal("the scenario is not attached to its key")
	}
	h.decode(http.MethodPost, "/control/reset", `{"partner":"manufacturer"}`, http.StatusOK,
		nil)
	if _, ok := h.sim.Runs().For("manufacturer", "KVW-1"); ok {
		t.Fatal("the reset kept the partner's scenario")
	}
}

func trimmed(control string) string {
	for len(control) > 1 && control[0] == '0' {
		control = control[1:]
	}
	return control
}
