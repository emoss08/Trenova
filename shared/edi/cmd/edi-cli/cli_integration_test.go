package main

import (
    "encoding/json"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

func TestCLI_Shipment_IncludesSegments_WhenProfileEnabled(t *testing.T) {
    tmp := t.TempDir()
    bin := filepath.Join(tmp, "edi-cli-testbin")
    cmd := exec.Command("go", "build", "-o", bin)
    cmd.Env = append(os.Environ(), "GOWORK=off")
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("build edi-cli failed: %v\n%s", err, string(out))
    }

    profPath := filepath.Join(tmp, "profile.json")
    if err := os.WriteFile(profPath, []byte(`{"include_segments": true}`), 0644); err != nil {
        t.Fatalf("write profile: %v", err)
    }
    edi := filepath.Join("..", "..", "testdata", "204", "sample1.edi")
    run := exec.Command(bin, "--format", "shipment", "--profile", profPath, edi)
    out, err = run.CombinedOutput()
    if err != nil {
        t.Fatalf("edi-cli run failed: %v\n%s", err, string(out))
    }
    // Output should be a JSON object with shipment and segments
    var payload struct {
        Segments []any `json:"segments"`
    }
    if err := json.Unmarshal(out, &payload); err != nil {
        t.Fatalf("json parse: %v\n%s", err, string(out))
    }
    if len(payload.Segments) == 0 {
        t.Fatalf("expected segments in output, got none. JSON: %s", string(out))
    }
}

func TestCLI_FailOnError_ExitCode(t *testing.T) {
    tmp := t.TempDir()
    bin := filepath.Join(tmp, "edi-cli-testbin")
    cmd := exec.Command("go", "build", "-o", bin)
    cmd.Env = append(os.Environ(), "GOWORK=off")
    if out, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("build edi-cli failed: %v\n%s", err, string(out))
    }
    edi := filepath.Join("..", "..", "testdata", "204", "invalid_dtm_format.edi")
    run := exec.Command(bin, "--format", "shipment", "--validate", "--fail-on-error", edi)
    if err := run.Run(); err == nil {
        t.Fatalf("expected non-zero exit code due to validation errors")
    }
}

func TestCLI_Profile_ISODatetime(t *testing.T) {
    tmp := t.TempDir()
    bin := filepath.Join(tmp, "edi-cli-testbin")
    cmd := exec.Command("go", "build", "-o", bin)
    cmd.Env = append(os.Environ(), "GOWORK=off")
    if out, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("build edi-cli failed: %v\n%s", err, string(out))
    }
    profPath := filepath.Join(tmp, "profile.json")
    if err := os.WriteFile(profPath, []byte(`{"emit_iso_datetime": true, "timezone": "UTC"}`), 0644); err != nil {
        t.Fatalf("write profile: %v", err)
    }
    edi := filepath.Join("..", "..", "testdata", "204", "sample1.edi")
    run := exec.Command(bin, "--format", "shipment", "--profile", profPath, edi)
    out, err := run.CombinedOutput()
    if err != nil {
        t.Fatalf("edi-cli run failed: %v\n%s", err, string(out))
    }
    // Support both wrapped and bare shipment shapes
    var anyObj map[string]any
    if err := json.Unmarshal(out, &anyObj); err != nil {
        t.Fatalf("json parse: %v\n%s", err, string(out))
    }
    var stops any
    if shp, ok := anyObj["shipment"].(map[string]any); ok {
        stops = shp["stops"]
    } else {
        stops = anyObj["stops"]
    }
    arr, ok := stops.([]any)
    if !ok || len(arr) == 0 {
        t.Fatalf("expected stops present; json=%s", string(out))
    }
    first := arr[0].(map[string]any)
    appts, ok := first["appointments"].([]any)
    if !ok || len(appts) == 0 {
        t.Fatalf("expected appointments present; json=%s", string(out))
    }
    a0 := appts[0].(map[string]any)
    if dt, _ := a0["datetime"].(string); dt != "2024-01-02T08:00:00Z" {
        t.Fatalf("expected ISO datetime, got %q", dt)
    }
}

func TestCLI_AckJSON_WrapsAckAndJSON(t *testing.T) {
    tmp := t.TempDir()
    bin := filepath.Join(tmp, "edi-cli-testbin")
    cmd := exec.Command("go", "build", "-o", bin)
    cmd.Env = append(os.Environ(), "GOWORK=off")
    if out, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("build edi-cli failed: %v\n%s", err, string(out))
    }
    ediRel := filepath.Join("testdata", "204", "sample1.edi")
    run := exec.Command(bin, "--format", "204", "--validate", "--ack", "--ack-json", ediRel)
    // Run from repo root so relative schema path resolves
    run.Dir = filepath.Join("..", "..")
    out, err := run.CombinedOutput()
    if err != nil {
        t.Fatalf("edi-cli run failed: %v\n%s", err, string(out))
    }
    var payload struct {
        Ack string `json:"ack"`
    }
    if err := json.Unmarshal(out, &payload); err != nil {
        t.Fatalf("json parse: %v\n%s", err, string(out))
    }
    if payload.Ack == "" {
        t.Fatalf("expected ack content in json")
    }
}
