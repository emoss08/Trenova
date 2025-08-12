package ack

import (
	"os"
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func TestGenerate997_AcceptedAndErrors(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/204/sample1.edi")
	if err != nil {
		t.Skip("sample not present")
	}
	d, err := x12.DetectDelimiters(raw)
	if err != nil {
		t.Fatalf("delims: %v", err)
	}
	segs, err := x12.ParseSegments(raw, d)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// No errors -> AK9*A
	iss := validation.Validate204(segs)
	edi := Generate997(segs, d, iss)
	if !strings.Contains(edi, "AK9*A") {
		t.Fatalf("expected AK9*A in ack, got: %s", edi)
	}
	// Introduce an error -> AK9*E
	bad := append([]x12.Segment(nil), segs...)
	for i := range bad {
		if bad[i].Tag == "B2" {
			if len(bad[i].Elements) > 1 {
				bad[i].Elements[1][0] = ""
			}
			break
		}
	}
	iss = validation.Validate204(bad)
	edi = Generate997(bad, d, iss)
	if !strings.Contains(edi, "AK9*E") {
		t.Fatalf("expected AK9*E in ack, got: %s", edi)
	}
}
