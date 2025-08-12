package main

import (
    "bufio"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

func TestScan_PerTransactionNDJSON(t *testing.T) {
    tmp := t.TempDir()
    // copy multi_tx.edi into temp dir
    src := filepath.Join("..", "..", "testdata", "204", "multi_tx.edi")
    dst := filepath.Join(tmp, "multi_tx.edi")
    in, err := os.Open(src)
    if err != nil { t.Skip("multi_tx.edi not available") }
    defer in.Close()
    out, err := os.Create(dst)
    if err != nil { t.Fatalf("create dst: %v", err) }
    if _, err := io.Copy(out, in); err != nil { t.Fatalf("copy: %v", err) }
    out.Close()

    // build edi-scan
    bin := filepath.Join(tmp, "edi-scan-testbin")
    cmd := exec.Command("go", "build", "-o", bin)
    cmd.Env = append(os.Environ(), "GOWORK=off")
    if b, err := cmd.CombinedOutput(); err != nil { t.Fatalf("build: %v\n%s", err, string(b)) }

    outPath := filepath.Join(tmp, "out.ndjson")
    run := exec.Command(bin, "-dir", tmp, "-out", outPath, "-per-tx")
    if b, err := run.CombinedOutput(); err != nil { t.Fatalf("scan run: %v\n%s", err, string(b)) }
    f, err := os.Open(outPath)
    if err != nil { t.Fatalf("open out: %v", err) }
    defer f.Close()
    sc := bufio.NewScanner(f)
    count := 0
    for sc.Scan() { count++ }
    if count != 2 {
        t.Fatalf("expected 2 NDJSON lines, got %d", count)
    }
}

