package x12

import (
    "os"
    "testing"
)

func TestParseSegments_BasicOrder(t *testing.T) {
    raw, err := os.ReadFile("../../testdata/204/sample1.edi")
    if err != nil {
        t.Fatalf("read sample: %v", err)
    }
    d, err := DetectDelimiters(raw)
    if err != nil {
        t.Fatalf("detect: %v", err)
    }
    segs, err := ParseSegments(raw, d)
    if err != nil {
        t.Fatalf("parse: %v", err)
    }
    if len(segs) == 0 {
        t.Fatalf("no segments parsed")
    }
    want := []string{"ISA", "GS", "ST", "B2", "B2A"}
    for i, w := range want {
        if i >= len(segs) || segs[i].Tag != w {
            t.Fatalf("unexpected tag at %d: got %q, want %q", i, segs[i].Tag, w)
        }
    }
}

func TestFindSegments_Filter(t *testing.T) {
    raw, _ := os.ReadFile("../../testdata/204/sample1.edi")
    d, _ := DetectDelimiters(raw)
    segs, _ := ParseSegments(raw, d)
    n1s := FindSegments(segs, "N1")
    if len(n1s) < 2 {
        t.Fatalf("expected at least 2 N1 segments, got %d", len(n1s))
    }
}

