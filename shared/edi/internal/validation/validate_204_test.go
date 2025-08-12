package validation

import (
	"os"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func TestValidate204_SampleOK(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/204/sample1.edi")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	d, _ := x12.DetectDelimiters(raw)
	segs, _ := x12.ParseSegments(raw, d)
	issues := Validate204(segs)
	for _, is := range issues {
		if is.Severity == Error {
			t.Fatalf("unexpected error: %+v", is)
		}
	}
}

func TestValidate204_MissingB2(t *testing.T) {
	raw, _ := os.ReadFile("../../testdata/204/invalid_missing_b2.edi")
	d, _ := x12.DetectDelimiters(raw)
	segs, _ := x12.ParseSegments(raw, d)
	issues := Validate204(segs)
	found := false
	for _, is := range issues {
		if is.Code == "B2.MISSING" && is.Severity == Error {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected B2.MISSING error, got: %+v", issues)
	}
}

func TestValidate204_SECountMismatch(t *testing.T) {
	raw, _ := os.ReadFile("../../testdata/204/invalid_se_count.edi")
	d, _ := x12.DetectDelimiters(raw)
	segs, _ := x12.ParseSegments(raw, d)
	issues := Validate204(segs)
	found := false
	for _, is := range issues {
		if is.Code == "SE.COUNT" && is.Severity == Error {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected SE.COUNT error, got: %+v", issues)
	}
}

func TestValidate204_Lenient_SECountWarning(t *testing.T) {
	raw, _ := os.ReadFile("../../testdata/204/invalid_se_count.edi")
	d, _ := x12.DetectDelimiters(raw)
	segs, _ := x12.ParseSegments(raw, d)
	prof := DefaultProfileForVersion(x12.ExtractVersion(segs))
	prof.Strictness = Lenient
	prof.EnforceSECount = false
	issues := Validate204WithProfile(segs, prof)
	var warn bool
	for _, is := range issues {
		if is.Code == "SE.COUNT" && is.Severity == Warning {
			warn = true
		}
		if is.Code == "SE.COUNT" && is.Severity == Error {
			t.Fatalf("SE.COUNT should be warning in lenient mode")
		}
	}
	if !warn {
		t.Fatalf("expected SE.COUNT warning in lenient mode")
	}
}

// S5 type codes are now validated via JSON schema rules rather than hard-coded base rules.

func TestValidate204_DTMFormatInvalid(t *testing.T) {
	raw, _ := os.ReadFile("../../testdata/204/invalid_dtm_format.edi")
	d, _ := x12.DetectDelimiters(raw)
	segs, _ := x12.ParseSegments(raw, d)
	issues := Validate204(segs)
	var badDate, badTime bool
	for _, is := range issues {
		if is.Code == "DTM.DATE.INVALID" {
			badDate = true
		}
		if is.Code == "DTM.TIME.INVALID" {
			badTime = true
		}
	}
	if !(badDate && badTime) {
		t.Fatalf("expected both DTM.DATE.INVALID and DTM.TIME.INVALID, got: %+v", issues)
	}
}

func TestValidate204_B2MissingFields(t *testing.T) {
	raw, _ := os.ReadFile("../../testdata/204/invalid_b2_missing_fields.edi")
	d, _ := x12.DetectDelimiters(raw)
	segs, _ := x12.ParseSegments(raw, d)
	issues := Validate204(segs)
	var scacMissing bool
	for _, is := range issues {
		if is.Code == "B2.SCAC.MISSING" {
			scacMissing = true
		}
	}
	if !scacMissing {
		t.Fatalf("expected B2.SCAC.MISSING, got: %+v", issues)
	}
}

func TestValidate204_6010_AllowsMissingB2ShipID(t *testing.T) {
	// 6010 samples should not require B2-03
	files := []string{
		"../../testdata/204/multi-stop-sample.edi",
		"../../testdata/204/sample3.edi",
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		d, _ := x12.DetectDelimiters(raw)
		segs, _ := x12.ParseSegments(raw, d)
		issues := Validate204(segs)
		for _, is := range issues {
			if is.Code == "B2.SHIPID.MISSING" {
				t.Fatalf("unexpected B2.SHIPID.MISSING for %s", f)
			}
		}
	}
}
