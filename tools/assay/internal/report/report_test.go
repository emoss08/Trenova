package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func render(t *testing.T, s Summary, verbose bool) string {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, NewPrinter(&buf, false).Summary(s, verbose))

	return buf.String()
}

func TestNewSummaryComputesReduction(t *testing.T) {
	s := NewSummary(sampleResult(), 40, 10)

	require.Len(t, s.SelectedPackages, 2)
	assert.InDelta(t, 80.0, s.ReductionPercent, 0.001)
	assert.Equal(t, 40, s.TotalPackages)
	assert.Equal(t, 10, s.TestablePackages)
}

func TestNewSummaryHandlesNoTestablePackages(t *testing.T) {
	s := NewSummary(selection.Result{}, 0, 0)

	assert.Zero(t, s.ReductionPercent)
	assert.Empty(t, s.SelectedPackages)
}

func TestNewSummaryCarriesReasons(t *testing.T) {
	s := NewSummary(sampleResult(), 40, 10)

	byPath := make(map[string]string, len(s.SelectedPackages))
	for _, pkg := range s.SelectedPackages {
		byPath[pkg.ImportPath] = pkg.Reason
	}

	assert.Equal(t, string(selection.ReasonDirect), byPath["example.com/repo"])
	assert.Equal(t, string(selection.ReasonDependent), byPath["example.com/app"])
}

func TestPrinterSummarizesCounts(t *testing.T) {
	out := render(t, NewSummary(sampleResult(), 40, 10), false)

	for _, want := range []string{"40 packages", "10 with tests", "2 selected", "80.0% skipped", "1 files"} {
		assert.Contains(t, out, want)
	}
	assert.NotContains(t, out, "example.com/app", "non-verbose output must not list packages")
}

func TestPrinterVerboseListsPackagesAndIgnoredFiles(t *testing.T) {
	out := render(t, NewSummary(sampleResult(), 40, 10), true)

	for _, want := range []string{"example.com/app", "example.com/repo", "dependent", "direct", "design.md"} {
		assert.Contains(t, out, want)
	}
}

func TestPrinterReportsSelectAll(t *testing.T) {
	res := selection.Result{SelectAll: true, SelectAllReason: "module definition changed: go.mod"}

	out := render(t, NewSummary(res, 40, 10), false)

	assert.Contains(t, out, "select-all: module definition changed: go.mod")
}

func TestPrinterEmitsAnsiOnlyWhenColorEnabled(t *testing.T) {
	s := NewSummary(sampleResult(), 40, 10)

	var plain, tinted bytes.Buffer
	require.NoError(t, NewPrinter(&plain, false).Summary(s, true))
	require.NoError(t, NewPrinter(&tinted, true).Summary(s, true))

	assert.NotContains(t, plain.String(), "\x1b[")
	assert.Contains(t, tinted.String(), "\x1b[")
	assert.Equal(t, stripANSI(tinted.String()), plain.String())
}

func stripANSI(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			out.WriteByte(s[i])

			continue
		}
		for i < len(s) && s[i] != 'm' {
			i++
		}
	}

	return out.String()
}

func TestWriteJSONRoundTrips(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, WriteJSON(&buf, NewSummary(sampleResult(), 40, 10)))

	var decoded Summary
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Len(t, decoded.SelectedPackages, 2)
	assert.InDelta(t, 80.0, decoded.ReductionPercent, 0.001)
}

func TestShortenTrimsLongImportPaths(t *testing.T) {
	cases := map[string]string{
		"example.com/app": "example.com/app",
		"a/b/c/d/e":       "a/b/c/d/e",
		"github.com/emoss08/trenova/internal/core/services/shipment": ".../trenova/internal/core/services/shipment",
	}

	for in, want := range cases {
		assert.Equal(t, want, shorten(in), "shorten(%q)", in)
	}
}
