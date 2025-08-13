package main

import (
    "bufio"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"

    "github.com/bytedance/sonic"
)

func TestCLI_Multi_NDJSON(t *testing.T) {
    tmp := t.TempDir()
    bin := filepath.Join(tmp, "edi-cli-testbin")
    cmd := exec.Command("go", "build", "-o", bin)
    cmd.Env = append(os.Environ(), "GOWORK=off")
    if out, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("build edi-cli failed: %v\n%s", err, string(out))
    }
    edi := filepath.Join("..", "..", "testdata", "204", "multi_tx.edi")
    run := exec.Command(bin, "--format", "shipment", "--multi", "ndjson", edi)
    out, err := run.CombinedOutput()
    if err != nil {
        t.Fatalf("edi-cli run failed: %v\n%s", err, string(out))
    }
    // Expect two NDJSON lines, each parseable JSON with shipment
    scanner := bufio.NewScanner(strings.NewReader(string(out)))
    count := 0
    for scanner.Scan() {
        line := scanner.Bytes()
        var obj map[string]any
        if err := sonic.Unmarshal(line, &obj); err != nil {
            t.Fatalf("invalid json line: %v\n%s", err, string(line))
        }
        if _, ok := obj["shipment"]; !ok {
            t.Fatalf("line missing shipment: %s", string(line))
        }
        count++
    }
    if count != 2 {
        t.Fatalf("expected 2 lines, got %d; output:%s", count, string(out))
    }
}

