package x12

import (
    "os"
    "testing"
)

func TestDetectDelimiters_Sample204(t *testing.T) {
    raw, err := os.ReadFile("../../testdata/204/sample1.edi")
    if err != nil {
        t.Fatalf("read sample: %v", err)
    }
    d, err := DetectDelimiters(raw)
    if err != nil {
        t.Fatalf("detect: %v", err)
    }
    if d.Element != '*' || d.Component != '>' || d.Segment != '~' {
        t.Fatalf("unexpected delimiters: %+v", d)
    }
    if d.Repetition != 0 {
        t.Fatalf("for 004010 repetition should be 0; got %q", d.Repetition)
    }
}

