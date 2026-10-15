package sim

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const specPath = "../../specs/simpartner-implementation-guide.txt"

func segmentIDs(x12 string) []string {
	segments := strings.Split(strings.TrimSpace(x12), "~")
	ids := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		ids = append(ids, strings.SplitN(segment, "*", 2)[0])
	}
	return ids
}

func specSample(t *testing.T, spec string) string {
	t.Helper()
	start := strings.Index(spec, "ISA*")
	end := strings.Index(spec, "IEA*")
	if start < 0 || end < start {
		t.Fatalf("the guide has no sample interchange")
	}
	closing := strings.Index(spec[end:], "~")
	if closing < 0 {
		t.Fatalf("the sample's IEA is not terminated")
	}
	lines := strings.Split(spec[start:end+closing+1], "\n")
	for index := range lines {
		lines[index] = strings.TrimSpace(lines[index])
	}
	return strings.Join(lines, "")
}

func TestSpecSampleIsTheTenderTheSimulatorSends(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Clean(specPath))
	if err != nil {
		t.Fatalf("read the simulator's guide: %v", err)
	}
	spec := string(raw)
	sample := specSample(t, spec)

	envelope, err := ParseX12Envelope(sample)
	if err != nil {
		t.Fatalf("the guide's sample does not parse: %v", err)
	}
	want := X12Envelope{
		SenderQualifier:          "ZZ",
		SenderID:                 "SIMPARTNER",
		ReceiverQualifier:        "ZZ",
		ReceiverID:               "TRENOVA",
		InterchangeControlNumber: "000000501",
		FunctionalGroupID:        "SM",
		GroupControlNumber:       "501",
		TransactionSet:           "204",
		TransactionControlNumber: transactionControlNumber,
	}
	if *envelope != want {
		t.Fatalf("the guide's sample envelope is %+v, want %+v", *envelope, want)
	}

	sent := BuildLoadTender204(BuildLoadTenderInput{
		SenderID:      "SIMPARTNER",
		ReceiverID:    "TRENOVA",
		ControlNumber: 501,
		ShipmentID:    "SIM000501",
	})
	if got, wantIDs := segmentIDs(sample), segmentIDs(sent); !slices.Equal(got, wantIDs) {
		t.Fatalf("the guide's sample segments are %v, the simulator sends %v", got, wantIDs)
	}
	if !strings.Contains(sample, "B2**SIML**SIM000501**PP~") {
		t.Fatalf("the guide's sample B2 is not the simulator's")
	}
}

func TestSpecNamesTheSimulatorsDefaultsAndNoSecret(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Clean(specPath))
	if err != nil {
		t.Fatalf("read the simulator's guide: %v", err)
	}
	spec := string(raw)
	for _, value := range []string{
		"SIMPARTNER", "TRENOVA", "http://localhost:9210/as2", "localhost", "9222", "trenova",
		"997", "004010", "SIM-SHIP", "SIM-CONS",
	} {
		if !strings.Contains(spec, value) {
			t.Errorf("the guide does not state %q", value)
		}
	}
	if strings.Contains(spec, "BEGIN") || strings.Contains(strings.ToLower(spec), "password:") {
		t.Errorf("the guide carries a key or a password; it must only say where to get them")
	}
}
