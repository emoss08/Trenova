package x12

import (
    "strings"
    "testing"
)

func TestSegmentScanner_Basic(t *testing.T) {
    data := "ST*204*0001~B2**SCAC*ABC123*CC~SE*3*0001~"
    d := Delimiters{Element: '*', Component: '>', Segment: '~'}
    sc := NewSegmentScanner(strings.NewReader(data), d)
    got := []string{}
    for sc.Next() { got = append(got, sc.Segment().Tag) }
    if sc.Err() != nil { t.Fatalf("scanner error: %v", sc.Err()) }
    if len(got) != 3 || got[0] != "ST" || got[2] != "SE" {
        t.Fatalf("unexpected tags: %#v", got)
    }
}

