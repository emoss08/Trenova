package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/emoss08/assay/internal/selection"
)

func sampleResult() selection.Result {
	return selection.Result{
		Packages: []string{"example.com/app", "example.com/repo"},
		Reasons: map[string]selection.Reason{
			"example.com/app":  selection.ReasonDependent,
			"example.com/repo": selection.ReasonDirect,
		},
		ChangedPackages: []string{"example.com/repo"},
		Unattributed:    []string{"/repo/docs/design.md"},
	}
}

func TestNewSummaryComputesReduction(t *testing.T) {
	s := NewSummary(sampleResult(), 40, 10)

	if len(s.SelectedPackages) != 2 {
		t.Fatalf("SelectedPackages = %d, want 2", len(s.SelectedPackages))
	}
	if s.ReductionPercent != 80 {
		t.Errorf("ReductionPercent = %v, want 80", s.ReductionPercent)
	}
	if s.TotalPackages != 40 || s.TestablePackages != 10 {
		t.Errorf("counts = (%d, %d), want (40, 10)", s.TotalPackages, s.TestablePackages)
	}
}

func TestNewSummaryHandlesNoTestablePackages(t *testing.T) {
	s := NewSummary(selection.Result{}, 0, 0)

	if s.ReductionPercent != 0 {
		t.Errorf("ReductionPercent = %v, want 0", s.ReductionPercent)
	}
	if len(s.SelectedPackages) != 0 {
		t.Errorf("SelectedPackages = %v, want empty", s.SelectedPackages)
	}
}

func TestNewSummaryCarriesReasons(t *testing.T) {
	s := NewSummary(sampleResult(), 40, 10)

	byPath := make(map[string]string, len(s.SelectedPackages))
	for _, pkg := range s.SelectedPackages {
		byPath[pkg.ImportPath] = pkg.Reason
	}

	if byPath["example.com/repo"] != string(selection.ReasonDirect) {
		t.Errorf("repo reason = %q, want %q", byPath["example.com/repo"], selection.ReasonDirect)
	}
	if byPath["example.com/app"] != string(selection.ReasonDependent) {
		t.Errorf("app reason = %q, want %q", byPath["example.com/app"], selection.ReasonDependent)
	}
}

func TestWriteTextSummarizesCounts(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteText(&buf, NewSummary(sampleResult(), 40, 10), false); err != nil {
		t.Fatalf("WriteText: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"40 total", "10 with tests", "2 selected", "80.0% skipped", "1 files"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "example.com/app") {
		t.Error("non-verbose output must not list individual packages")
	}
}

func TestWriteTextVerboseListsPackagesAndIgnoredFiles(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteText(&buf, NewSummary(sampleResult(), 40, 10), true); err != nil {
		t.Fatalf("WriteText: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"example.com/app", "example.com/repo", "dependent", "direct", "design.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("verbose output missing %q:\n%s", want, out)
		}
	}
}

func TestWriteTextReportsSelectAll(t *testing.T) {
	res := selection.Result{SelectAll: true, SelectAllReason: "module definition changed: go.mod"}

	var buf bytes.Buffer
	if err := WriteText(&buf, NewSummary(res, 40, 10), false); err != nil {
		t.Fatalf("WriteText: %v", err)
	}

	if !strings.Contains(buf.String(), "select-all: module definition changed: go.mod") {
		t.Errorf("output missing select-all explanation:\n%s", buf.String())
	}
}

func TestWriteJSONRoundTrips(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, NewSummary(sampleResult(), 40, 10)); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var decoded Summary
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.SelectedPackages) != 2 || decoded.ReductionPercent != 80 {
		t.Errorf("decoded = %+v, want 2 packages and 80%% reduction", decoded)
	}
}

func TestShortenTrimsLongImportPaths(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"example.com/app", "example.com/app"},
		{"a/b/c/d/e", "a/b/c/d/e"},
		{"github.com/emoss08/trenova/internal/core/services/shipment", ".../trenova/internal/core/services/shipment"},
	}

	for _, tc := range cases {
		if got := shorten(tc.in); got != tc.want {
			t.Errorf("shorten(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
