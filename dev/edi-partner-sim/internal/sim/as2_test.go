package sim

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/acknowledge"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/shared/smime"
)

func TestEveryAS2ShipperRoundTrips(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"manufacturer", "grocer", "legacy"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t, setup{partners: []string{name}})
			settings := h.peerOf(name).def.Partner.Transport.AS2
			sent := h.send(name, &SendRequest{Set: "204", Seed: 37})
			if len(sent) != 1 {
				t.Fatalf("sent %d interchanges", len(sent))
			}
			tender := sent[0]
			if got := h.trenova.next(t); got.payload != tender.Raw ||
				got.from != settings.ID {
				t.Fatalf("trenova received %q from %s", firstLine(got.payload), got.from)
			}
			if tender.AS2 == nil || tender.AS2.Signed != settings.Signed ||
				tender.AS2.Encrypted != settings.Encrypted ||
				tender.AS2.Compressed != settings.Compressed {
				t.Fatalf("the AS2 settings were not honoured: %+v", tender.AS2)
			}
			switch settings.MDN {
			case mdnSync:
				if tender.State != journal.StateDelivered || tender.MDN.Status != mdnProcessed {
					t.Fatalf("sync MDN: %s %+v", tender.State, tender.MDN)
				}
			case mdnAsync:
				if tender.MDN.Status != mdnPending {
					t.Fatalf("async MDN before it arrives: %+v", tender.MDN)
				}
				done := h.await(
					"mdn.received",
					func(m *journal.Message) bool { return m.ID == tender.ID },
				)
				if done.MDN.Status != mdnProcessed || !done.MDN.Signed {
					t.Fatalf("async MDN: %+v", done.MDN)
				}
			}
			if len(tender.Keys) != 1 || tender.Ack == nil ||
				tender.Ack.Status != journal.AckAwaiting {
				t.Fatalf("the tender's key or ack expectation: %v %+v", tender.Keys, tender.Ack)
			}

			ack := trenovaX12(h.peerOf(name).def, false, 9001, "FA", "004010",
				"997~AK1*SM*"+tender.Groups[0].Control+"~AK2*204*0001~AK5*A~AK9*A*1*1*1")
			mdn, err := h.post(&as2Send{
				partner:    name,
				signed:     settings.Signed,
				encrypted:  settings.Encrypted,
				compressed: settings.Compressed,
				payload:    ack,
			})
			if err != nil || !mdn.Processed() || !mdn.Signed {
				t.Fatalf("the simulator's MDN: %+v %v", mdn, err)
			}
			acked := h.await(
				"ack.received",
				func(m *journal.Message) bool { return m.ID == tender.ID },
			)
			if acked.Ack.Code != acknowledge.Accepted || len(acked.Ack.Transactions) != 1 {
				t.Fatalf("the tender's ack: %+v", acked.Ack)
			}
			ackMessage := h.message(acked.Ack.MessageID)
			if ackMessage.Purpose != journal.PurposeAck || ackMessage.RespondsTo != tender.ID ||
				ackMessage.Keys[0] != tender.Keys[0] {
				t.Fatalf("the received ack: %+v", ackMessage)
			}
		})
	}
}

func TestTheSimulatorOpensEveryEncryptionAlgorithm(t *testing.T) {
	t.Parallel()

	h := newHarness(t, setup{partners: []string{"manufacturer"}})
	def := h.peerOf("manufacturer").def
	for index, algorithm := range []string{
		smime.EncryptionAlgorithmTripleDES,
		smime.EncryptionAlgorithmAES128CBC,
		smime.EncryptionAlgorithmAES256CBC,
		smime.EncryptionAlgorithmAES128GCM,
		smime.EncryptionAlgorithmAES256GCM,
	} {
		payload := trenovaX12(def, false, 600+index, "GF", "004010",
			"990~B1*TRNV*KVW-100037*20260302*A")
		mdn, err := h.post(&as2Send{
			partner:   "manufacturer",
			signed:    true,
			encrypted: true,
			algorithm: algorithm,
			payload:   payload,
		})
		if err != nil || !mdn.Processed() {
			t.Fatalf("%s: %+v %v", algorithm, mdn, err)
		}
		received := h.await("message.received", nil)
		if received.Raw != payload || !received.AS2.Encrypted || !received.AS2.Signed {
			t.Fatalf("%s: %+v", algorithm, received.AS2)
		}
	}
}

func TestASignedPartnerRefusesAnUnsignedMessage(t *testing.T) {
	t.Parallel()

	h := newHarness(t, setup{partners: []string{"manufacturer"}})
	payload := trenovaX12(h.peerOf("manufacturer").def, false, 700, "GF", "004010",
		"990~B1*TRNV*KVW-100037*20260302*A")
	mdn, err := h.post(&as2Send{partner: "manufacturer", encrypted: true, payload: payload})
	if err != nil || mdn.Processed() {
		t.Fatalf("an unsigned message was processed: %+v %v", mdn, err)
	}
	received := h.await("message.received", nil)
	if received.State != journal.StateUnreadable || received.Error == "" {
		t.Fatalf("the refused message: %s %q", received.State, received.Error)
	}
}

func TestTheAcknowledgmentWaitsForThePartnersDelay(t *testing.T) {
	t.Parallel()

	h := newHarness(t, setup{partners: []string{"manufacturer"}})
	payload := trenovaX12(h.peerOf("manufacturer").def, false, 501, "GF", "004010",
		"990~B1*TRNV*KVW-100037*20260302*A",
		"990~B1*TRNV*KVW-100038*20260302*Q")
	if _, err := h.post(&as2Send{
		partner: "manufacturer", signed: true, encrypted: true, payload: payload,
	}); err != nil {
		t.Fatalf("post: %v", err)
	}
	received := h.await("message.received", nil)
	if received.Judgement == nil || *received.Accepted || len(received.Keys) != 2 {
		t.Fatalf("the oracle's judgement: accepted %v keys %v", *received.Accepted, received.Keys)
	}
	scheduled := h.await(
		"ack.scheduled",
		func(m *journal.Message) bool { return m.ID == received.ID },
	)
	if !scheduled.Ack.DueAt.Equal(startedAt.Add(45*time.Second)) ||
		scheduled.Ack.Code != acknowledge.Rejected {
		t.Fatalf("the scheduled ack: %+v", scheduled.Ack)
	}
	h.trenova.nothing(t)
	h.clock.Advance(44 * time.Second)
	h.trenova.nothing(t)
	h.clock.Advance(time.Second)
	got := h.trenova.next(t)
	for _, want := range []string{"AK1*GF*501", "AK2*990*0001", "AK5*A", "AK2*990*0002", "AK4*4*", "AK5*R", "AK9*P*2*2*1"} {
		if !strings.Contains(got.payload, want) {
			t.Fatalf("the 997 lacks %q:\n%s", want, got.payload)
		}
	}
	sent := h.await("ack.sent", func(m *journal.Message) bool { return m.ID == received.ID })
	if sent.Ack.Status != journal.AckSent || sent.Ack.At == nil {
		t.Fatalf("the ack status: %+v", sent.Ack)
	}
}

func TestAPartnerThatNeverAcknowledgesWithholdsIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t, setup{
		partners: []string{"manufacturer"},
		edit:     func(d *partner.Definition) { d.Partner.Acknowledgment.Never = 1 },
	})
	payload := trenovaX12(h.peerOf("manufacturer").def, false, 502, "GF", "004010",
		"990~B1*TRNV*KVW-100037*20260302*A")
	if _, err := h.post(&as2Send{
		partner: "manufacturer", signed: true, encrypted: true, payload: payload,
	}); err != nil {
		t.Fatalf("post: %v", err)
	}
	withheld := h.await("ack.withheld", nil)
	if withheld.Ack.Status != journal.AckWithheld || withheld.Ack.MessageID != "" {
		t.Fatalf("the withheld ack: %+v", withheld.Ack)
	}
	h.clock.Advance(time.Hour)
	h.trenova.nothing(t)
}

func TestAResendReusesTheOriginalsBytes(t *testing.T) {
	t.Parallel()

	h := newHarness(t, setup{partners: []string{"legacy", "manufacturer"}})
	original := h.send("legacy", &SendRequest{Set: "204", Seed: 5})[0]
	again := h.send("legacy", &SendRequest{Set: "204", Seed: 5, Variation: "duplicate"})[0]
	if again.Raw != original.Raw || again.Interchange != original.Interchange || !again.Resend {
		t.Fatalf(
			"the resend is not the original: %s vs %s",
			again.Interchange,
			original.Interchange,
		)
	}
	if _, err := h.sim.Send(t.Context(), "legacy", &SendRequest{
		Set: "204", Seed: 6, Variation: "duplicate",
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("a resend of a tender never sent: %v", err)
	}
	first := h.send("manufacturer", &SendRequest{Set: "204", Seed: 5})[0]
	duplicate := h.send("manufacturer", &SendRequest{Set: "204", Seed: 5, Variation: "duplicate"})[0]
	if duplicate.Interchange == first.Interchange || duplicate.Resend {
		t.Fatalf("the manufacturer's duplicate reused control number %s", first.Interchange)
	}
	if _, err := h.sim.Send(t.Context(), "manufacturer", &SendRequest{
		Set: "204", Seed: 5, Variation: "sideways",
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("an unknown variation: %v", err)
	}
}
