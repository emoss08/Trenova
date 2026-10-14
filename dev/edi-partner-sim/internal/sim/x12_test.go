package sim

import (
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestBuildLoadTender204ClosesTransactionSet(t *testing.T) {
	t.Parallel()

	segments := x12Segments(BuildLoadTender204(BuildLoadTenderInput{
		SenderID:      "SIMPARTNER",
		ReceiverID:    "TRENOVA",
		ControlNumber: 42,
		ShipmentID:    "SIM000042",
	}))
	assertTransactionSetClosed(t, segments)
}

func TestBuild997ClosesTransactionSet(t *testing.T) {
	t.Parallel()

	segments := x12Segments(Build997(Build997Input{
		SenderID:      "TRENOVA",
		ReceiverID:    "SIMPARTNER",
		ControlNumber: 7,
		Original: &X12Envelope{
			FunctionalGroupID:        "SM",
			GroupControlNumber:       "42",
			TransactionSet:           "204",
			TransactionControlNumber: "0001",
		},
	}))
	assertTransactionSetClosed(t, segments)
}

func TestBuildLoadTender204OmitsTrailingEmptyElements(t *testing.T) {
	t.Parallel()

	tender := BuildLoadTender204(BuildLoadTenderInput{
		SenderID:      "SIMPARTNER",
		ReceiverID:    "TRENOVA",
		ControlNumber: 42,
		ShipmentID:    "SIM000042",
	})
	for _, segment := range strings.Split(strings.TrimSuffix(tender, "~"), "~") {
		if strings.HasSuffix(segment, "*") {
			t.Errorf("segment ends with an element separator: %q", segment)
		}
	}
}

func TestBuildLoadTender204StopDateTimeQualifiers(t *testing.T) {
	t.Parallel()

	segments := x12Segments(BuildLoadTender204(BuildLoadTenderInput{
		SenderID:      "SIMPARTNER",
		ReceiverID:    "TRENOVA",
		ControlNumber: 42,
		ShipmentID:    "SIM000042",
	}))
	stopG62s := make(map[string][]string, 2)
	stop := ""
	for _, segment := range segments {
		switch segment[0] {
		case "S5":
			stop = segment[2]
		case "G62":
			if stop == "" {
				t.Fatalf("G62 outside a stop: %v", segment)
			}
			if _, ok := stopG62s[stop]; ok {
				t.Fatalf("stop %s has more than one G62", stop)
			}
			stopG62s[stop] = segment
		}
	}

	tests := []struct {
		stop          string
		dateQualifier string
		timeQualifier string
	}{
		{stop: "CL", dateQualifier: "10", timeQualifier: "Y"},
		{stop: "CU", dateQualifier: "68", timeQualifier: "Z"},
	}
	for _, tt := range tests {
		g62, ok := stopG62s[tt.stop]
		if !ok {
			t.Fatalf("stop %s has no G62", tt.stop)
		}
		if len(g62) != 5 {
			t.Fatalf("stop %s G62 has %d elements: %v", tt.stop, len(g62)-1, g62)
		}
		if g62[1] != tt.dateQualifier || g62[3] != tt.timeQualifier {
			t.Errorf(
				"stop %s G62 qualifiers = %s/%s, want %s/%s",
				tt.stop, g62[1], g62[3], tt.dateQualifier, tt.timeQualifier,
			)
		}
	}

	pickup, err := time.Parse("200601021504", stopG62s["CL"][2]+stopG62s["CL"][4])
	if err != nil {
		t.Fatalf("parse pickup date/time: %v", err)
	}
	delivery, err := time.Parse("200601021504", stopG62s["CU"][2]+stopG62s["CU"][4])
	if err != nil {
		t.Fatalf("parse delivery date/time: %v", err)
	}
	if want := pickup.AddDate(0, 0, 2); !delivery.Equal(want) {
		t.Errorf("delivery = %s, want two days after pickup (%s)", delivery, want)
	}
}

func x12Segments(raw string) [][]string {
	parts := strings.Split(strings.TrimSuffix(raw, "~"), "~")
	segments := make([][]string, 0, len(parts))
	for _, part := range parts {
		segments = append(segments, strings.Split(part, "*"))
	}
	return segments
}

func assertTransactionSetClosed(t *testing.T, segments [][]string) {
	t.Helper()

	start, end := -1, -1
	for index, segment := range segments {
		switch segment[0] {
		case "ST":
			if start != -1 {
				t.Fatalf("more than one ST segment")
			}
			start = index
		case "SE":
			if end != -1 {
				t.Fatalf("more than one SE segment")
			}
			end = index
		}
	}
	if start == -1 || end == -1 || end < start {
		t.Fatalf("missing or misordered ST/SE: ST at %d, SE at %d", start, end)
	}
	st, se := segments[start], segments[end]
	if len(st) != 3 || len(se) != 3 {
		t.Fatalf("unexpected ST/SE shape: %v, %v", st, se)
	}
	if want := strconv.Itoa(end - start + 1); se[1] != want {
		t.Errorf("SE01 = %s, want %s (segments ST through SE)", se[1], want)
	}
	if se[2] != st[2] {
		t.Errorf("SE02 = %s, want ST02 %s", se[2], st[2])
	}
}
