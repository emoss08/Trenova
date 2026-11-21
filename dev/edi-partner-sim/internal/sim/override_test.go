package sim

import (
	"errors"
	"strconv"
	"testing"

	"github.com/emoss08/trenova/edi-partner-sim/internal/acknowledge"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
)

func TestAckOverridesRejectOrWithholdTheNextMatchingTransaction(t *testing.T) {
	t.Parallel()

	h := newHarness(t, setup{partners: []string{"manufacturer"}})
	def := h.peerOf("manufacturer").def
	settings := def.Partner.Transport.AS2
	if _, err := h.sim.OverrideAcks("manufacturer", &AckOverrideRequest{Overrides: []AckOverride{
		{Set: "990", Key: "KVW-100037", Action: OverrideReject},
		{Set: "990", Key: "KVW-100037", Action: OverrideWithhold},
	}}); err != nil {
		t.Fatal(err)
	}
	answer := func(control int, key string) journal.Message {
		payload := trenovaX12(def, false, control, "GF", "004010",
			"990~B1*TRNV*"+key+"*20260302*A")
		if _, err := h.post(&as2Send{
			partner: "manufacturer", signed: settings.Signed, encrypted: settings.Encrypted,
			payload: payload,
		}); err != nil {
			t.Fatalf("post: %v", err)
		}
		got := h.await("message.received", func(m *journal.Message) bool {
			return len(m.Keys) == 1 && m.Keys[0] == key && len(m.Groups) == 1 &&
				m.Groups[0].Control == strconv.Itoa(control)
		})
		return h.message(got.ID)
	}
	other := answer(801, "KVW-100038")
	if other.Ack == nil || other.Ack.Code != acknowledge.Accepted {
		t.Fatalf("another load's 990 was overridden: %+v", other.Ack)
	}
	rejected := answer(802, "KVW-100037")
	if rejected.Ack == nil || rejected.Ack.Code != acknowledge.Rejected ||
		rejected.Ack.Status != journal.AckScheduled {
		t.Fatalf("the first override: %+v", rejected.Ack)
	}
	withheld := answer(803, "KVW-100037")
	if withheld.Ack == nil || withheld.Ack.Status != journal.AckWithheld {
		t.Fatalf("the second override: %+v", withheld.Ack)
	}
	normal := answer(804, "KVW-100037")
	if normal.Ack == nil || normal.Ack.Code != acknowledge.Accepted {
		t.Fatalf("the overrides outlived their use: %+v", normal.Ack)
	}
	if _, err := h.sim.OverrideAcks("manufacturer", &AckOverrideRequest{
		Overrides: []AckOverride{{Action: "ignore"}},
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("an unknown action: %v", err)
	}
}
