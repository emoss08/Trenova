package sim

import (
	"strings"
	"testing"
)

func TestParseX12EnvelopeAnd997RoundTrip(t *testing.T) {
	t.Parallel()

	tender := BuildLoadTender204(BuildLoadTenderInput{
		SenderID:      "SIMPARTNER",
		ReceiverID:    "TRENOVA",
		ControlNumber: 42,
		ShipmentID:    "SIM000042",
	})
	envelope, err := ParseX12Envelope(tender)
	if err != nil {
		t.Fatalf("parse envelope: %v", err)
	}
	if envelope.SenderID != "SIMPARTNER" || envelope.ReceiverID != "TRENOVA" {
		t.Fatalf("unexpected identities: %+v", envelope)
	}
	if envelope.TransactionSet != "204" || envelope.TransactionControlNumber != "0001" {
		t.Fatalf("unexpected transaction: %+v", envelope)
	}
	if envelope.GroupControlNumber != "42" || envelope.InterchangeControlNumber != "000000042" {
		t.Fatalf("unexpected control numbers: %+v", envelope)
	}

	ack := Build997(Build997Input{
		SenderID:      "TRENOVA",
		ReceiverID:    "SIMPARTNER",
		ControlNumber: 7,
		Original:      envelope,
	})
	if !strings.Contains(ack, "AK1*SM*42~") {
		t.Fatalf("997 missing AK1 group reference: %s", ack)
	}
	if !strings.Contains(ack, "AK2*204*0001~") {
		t.Fatalf("997 missing AK2 transaction reference: %s", ack)
	}
	ackEnvelope, err := ParseX12Envelope(ack)
	if err != nil {
		t.Fatalf("parse 997 envelope: %v", err)
	}
	if ackEnvelope.TransactionSet != "997" || ackEnvelope.FunctionalGroupID != "FA" {
		t.Fatalf("unexpected 997 envelope: %+v", ackEnvelope)
	}
}
