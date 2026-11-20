package sim

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
)

const carrierTenderKey = "S26036847599"

func carrierTender(t *testing.T, control string) string {
	t.Helper()
	raw, err := os.ReadFile(partnersRoot + "/carrier/lading/samples/tender-out.x12")
	if err != nil {
		t.Fatalf("read the carrier's tender: %v", err)
	}
	group := strings.TrimLeft(control, "0")
	return strings.NewReplacer(
		"000000701", control,
		"|0930|701|", "|0930|"+group+"|",
		"GE|1|701~", "GE|1|"+group+"~",
	).Replace(string(raw))
}

func (h *harness) tenderCarrier(control string) journal.Message {
	h.t.Helper()
	if _, err := h.post(&as2Send{
		partner:    "carrier",
		signed:     true,
		encrypted:  true,
		compressed: true,
		payload:    carrierTender(h.t, control),
	}); err != nil {
		h.t.Fatalf("post the tender: %v", err)
	}
	return h.await("answer.scheduled", nil)
}

func (h *harness) collect(count int, pace time.Duration) []delivered {
	h.t.Helper()
	out := make([]delivered, 0, count)
	for range count {
		h.clock.Advance(pace)
		out = append(out, h.trenova.next(h.t))
	}
	return out
}

func TestACarrierAnswersATenderByItsScript(t *testing.T) {
	t.Parallel()

	h := newHarness(t, setup{partners: []string{"carrier"}})
	queued, err := h.sim.Script("carrier", &ScriptRequest{Answers: []ScriptAnswer{{
		Key:     carrierTenderKey,
		Decline: "No driver near the pickup",
	}}})
	if err != nil || len(queued) != 1 {
		t.Fatalf("script: %v %v", queued, err)
	}
	received := h.tenderCarrier("000000701")
	if !*received.Accepted || len(received.Answers) != 1 || received.Keys[0] != carrierTenderKey {
		t.Fatalf("the tender: accepted %v answers %v keys %v",
			*received.Accepted, received.Answers, received.Keys)
	}
	answer := h.collect(1, 2*time.Second)[0]
	if !strings.Contains(answer.payload, "B1|QYSF|"+carrierTenderKey) ||
		!strings.Contains(answer.payload, "|D~") {
		t.Fatalf("the 990 is not a decline:\n%s", answer.payload)
	}
	h.clock.Advance(2 * time.Minute)
	if ack := h.trenova.next(t); !strings.Contains(ack.payload, "AK5|A") {
		t.Fatalf("the 997:\n%s", ack.payload)
	}
	if left, _ := h.sim.Script("carrier", &ScriptRequest{}); len(left) != 0 {
		t.Fatalf("the script entry was not used up: %v", left)
	}
}

func TestACarrierAnswersByItsPolicyWithoutAScript(t *testing.T) {
	t.Parallel()

	h := newHarness(t, setup{partners: []string{"carrier"}})
	received := h.tenderCarrier("000000702")
	if len(received.Answers) < 3 {
		t.Fatalf("an accepted tender got %d answers", len(received.Answers))
	}
	answers := h.collect(len(received.Answers), 2*time.Second)
	if !strings.Contains(answers[0].payload, "ST|990") ||
		!strings.Contains(answers[0].payload, "|A~") {
		t.Fatalf("the first answer is not an acceptance:\n%s", answers[0].payload)
	}
	if !strings.Contains(answers[1].payload, "ST|214") {
		t.Fatalf("the second answer is not a status:\n%s", answers[1].payload)
	}
	if !strings.Contains(answers[len(answers)-1].payload, "ST|210") {
		t.Fatalf("the last answer is not the invoice:\n%s", answers[len(answers)-1].payload)
	}
	conversation := h.sim.Journal().Conversations("carrier")
	if len(conversation) != 1 || conversation[0].Key != carrierTenderKey ||
		len(conversation[0].Messages) != len(received.Answers)+2 {
		t.Fatalf("the conversation: %+v", conversation)
	}
}
