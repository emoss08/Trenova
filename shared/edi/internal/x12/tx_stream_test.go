package x12

import (
    "io"
    "os"
    "testing"
)

func TestTxScanner_Multiple204(t *testing.T) {
    raw, err := os.ReadFile("../../testdata/204/multi_tx.edi")
    if err != nil { t.Skip("fixture not found") }
    d, err := DetectDelimiters(raw)
    if err != nil { t.Fatalf("delims: %v", err) }
    sc := NewTxScanner(bytesReader(raw), d)
    count := 0
    for sc.Next() {
        tx := sc.Tx()
        if tx.SetID != "204" { continue }
        count++
    }
    if sc.Err() != nil { t.Fatalf("scanner err: %v", sc.Err()) }
    if count != 2 { t.Fatalf("expected 2 204 txs, got %d", count) }
}

// bytesReader wraps a byte slice as an io.Reader without copying.
func bytesReader(b []byte) io.Reader { return &byteReader{b: b} }

type byteReader struct{ b []byte }
func (r *byteReader) Read(p []byte) (int, error) {
    if len(r.b) == 0 { return 0, io.EOF }
    n := copy(p, r.b)
    r.b = r.b[n:]
    return n, nil
}
