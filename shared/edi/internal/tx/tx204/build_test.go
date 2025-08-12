package tx204

import (
	"os"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func TestBuildFromSegments_MVP(t *testing.T) {
	raw, err := os.ReadFile("../../../testdata/204/sample1.edi")
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	delims, err := x12.DetectDelimiters(raw)
	if err != nil {
		t.Fatalf("detect delimiters: %v", err)
	}
	segs, err := x12.ParseSegments(raw, delims)
	if err != nil {
		t.Fatalf("parse segments: %v", err)
	}
	lt := BuildFromSegments(segs)
	if lt.Control.STControl != "0001" {
		t.Fatalf("unexpected ST control: %q", lt.Control.STControl)
	}
	if len(lt.Stops) != 2 {
		t.Fatalf("expected 2 stops, got %d", len(lt.Stops))
	}
	if lt.Header.References["PO"][0] != "PO12345" {
		t.Fatalf("missing PO reference")
	}
}
