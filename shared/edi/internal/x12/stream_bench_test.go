package x12

import (
    "bytes"
    "testing"
)

func BenchmarkSegmentScannerLarge(b *testing.B) {
    tmpl := []byte("ST*204*0001~B2**SCAC*ABC123*CC~S5*1*LD~DTM*133*20240102*0800~S5*2*UL~DTM*132*20240103*1600~SE*7*0001~")
    big := bytes.Repeat(tmpl, 1000)
    d := Delimiters{Element: '*', Component: '>', Segment: '~'}
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        sc := NewSegmentScanner(bytes.NewReader(big), d)
        for sc.Next() { _ = sc.Segment() }
        if sc.Err() != nil { b.Fatal(sc.Err()) }
    }
}

