package validation

import (
	"os"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func TestValidateWithSchema_Logico6010(t *testing.T) {
	sch, err := LoadSchema("../../testdata/schema/logico-204-6010.json")
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}

	// multi-stop-sample should pass schema checks
	raw, err := os.ReadFile("../../testdata/204/multi-stop-sample.edi")
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	d, _ := x12.DetectDelimiters(raw)
	segs, _ := x12.ParseSegments(raw, d)
	issues := ValidateWithSchema(segs, sch)
	for _, is := range issues {
		if is.Severity == Error {
			t.Fatalf("unexpected schema error: %+v", is)
		}
	}

	// file with invalid S5 type should trip schema allowed_values
	raw2, _ := os.ReadFile("../../testdata/204/invalid_s5_type.edi")
	d2, _ := x12.DetectDelimiters(raw2)
	segs2, _ := x12.ParseSegments(raw2, d2)
	issues2 := ValidateWithSchema(segs2, sch)
	found := false
	for _, is := range issues2 {
		if is.Code == "S5-02.VALUE" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected S5-02.VALUE error from schema, got: %+v", issues2)
	}
}
