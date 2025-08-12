package validation

import (
	"os"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func TestGeneric4010Schema_AllowsCommonCodes(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/204/sample1.edi")
	if err != nil {
		t.Skipf("sample not found: %v", err)
	}
	delims, err := x12.DetectDelimiters(raw)
	if err != nil {
		t.Fatalf("delims: %v", err)
	}
	segs, err := x12.ParseSegments(raw, delims)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sch, err := LoadSchema("../../testdata/schema/generic-204-4010.json")
	if err != nil {
		t.Fatalf("schema load: %v", err)
	}
	issues := ValidateWithSchema(segs, sch)
	// Ensure no B2A or N1 code-value errors for valid sample
	for _, is := range issues {
		if is.Code == "B2A-01.VALUE" || is.Code == "N1-01.VALUE" {
			t.Fatalf("unexpected code value issue: %#v", is)
		}
	}
}

func TestGeneric4010Schema_FlagsInvalidCodes(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/204/sample1.edi")
	if err != nil {
		t.Skipf("sample not found: %v", err)
	}
	delims, err := x12.DetectDelimiters(raw)
	if err != nil {
		t.Fatalf("delims: %v", err)
	}
	segs, err := x12.ParseSegments(raw, delims)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Tweak B2A-01 to an invalid value and N1-01 for one occurrence
	for i := range segs {
		if segs[i].Tag == "B2A" {
			if len(segs[i].Elements) >= 1 && len(segs[i].Elements[0]) > 0 {
				segs[i].Elements[0][0] = "ZZ"
				break
			}
		}
	}
	for i := range segs {
		if segs[i].Tag == "N1" {
			if len(segs[i].Elements) >= 1 && len(segs[i].Elements[0]) > 0 {
				segs[i].Elements[0][0] = "XX"
				break
			}
		}
	}
	sch, err := LoadSchema("../../testdata/schema/generic-204-4010.json")
	if err != nil {
		t.Fatalf("schema load: %v", err)
	}
	issues := ValidateWithSchema(segs, sch)
	gotB2A := false
	gotN1 := false
	for _, is := range issues {
		if is.Code == "B2A-01.VALUE" {
			gotB2A = true
		}
		if is.Code == "N1-01.VALUE" {
			gotN1 = true
		}
	}
	if !gotB2A || !gotN1 {
		t.Fatalf("expected B2A-01.VALUE and N1-01.VALUE issues, got: %#v", issues)
	}
}

func TestGeneric4010Schema_DTMQualifierPresenceAndAllowed(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/204/sample1.edi")
	if err != nil {
		t.Skipf("sample not found: %v", err)
	}
	delims, err := x12.DetectDelimiters(raw)
	if err != nil {
		t.Fatalf("delims: %v", err)
	}
	segs, err := x12.ParseSegments(raw, delims)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sch, err := LoadSchema("../../testdata/schema/generic-204-4010.json")
	if err != nil {
		t.Fatalf("schema load: %v", err)
	}
	// First verify no DTM-01 issues in baseline
	issues := ValidateWithSchema(segs, sch)
	for _, is := range issues {
		if is.Code == "DTM-01.VALUE" || is.Code == "PRESENCE.DTM-01" {
			t.Fatalf("unexpected DTM issue: %#v", is)
		}
	}
	// Change a DTM qualifier to invalid and expect VALUE issue
	bad := append([]x12.Segment(nil), segs...)
	for i := range bad {
		if bad[i].Tag == "DTM" {
			if len(bad[i].Elements) > 0 && len(bad[i].Elements[0]) > 0 {
				bad[i].Elements[0][0] = "999"
				break
			}
		}
	}
	issues = ValidateWithSchema(bad, sch)
	found := false
	for _, is := range issues {
		if is.Code == "DTM-01.VALUE" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected DTM-01.VALUE issue")
	}
	// Remove all DTM*133 and expect presence rule to trigger
	filtered := make([]x12.Segment, 0, len(segs))
	for _, s := range segs {
		if s.Tag == "DTM" && len(s.Elements) > 0 && len(s.Elements[0]) > 0 &&
			s.Elements[0][0] == "133" {
			continue
		}
		filtered = append(filtered, s)
	}
	issues = ValidateWithSchema(filtered, sch)
	needPresence := false
	for _, is := range issues {
		if is.Code == "PRESENCE.DTM-01" {
			needPresence = true
			break
		}
	}
	if !needPresence {
		t.Fatalf("expected PRESENCE.DTM-01 due to missing DTM 133")
	}
}
