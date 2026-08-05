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

func packageResult() selection.Result {
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

func narrowedInputs() Inputs {
	return Inputs{
		Narrowed: selection.Narrowed{
			Result:  packageResult(),
			Enabled: true,
			Plans: []selection.PackagePlan{
				{
					ImportPath: "example.com/repo",
					Reason:     selection.ReasonDirect,
					Tests:      []string{"TestGet", "TestPut"},
					AllTests:   []string{"TestGet", "TestOther", "TestPut"},
				},
				{
					ImportPath: "example.com/app",
					Reason:     selection.ReasonDependent,
					FullReason: "no index record for this package",
				},
			},
			NarrowedPackages: 1,
			SelectedTests:    2,
			Blocked:          []string{"/repo/svc/gen.go"},
		},
		TotalPackages:    40,
		TestablePackages: 10,
	}
}

func render(t *testing.T, s Summary, verbose bool) string {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, NewPrinter(&buf, false).Summary(s, verbose))

	return buf.String()
}

func TestNewSummaryCountsModes(t *testing.T) {
	s := NewSummary(narrowedInputs())

	assert.Equal(t, 1, s.FullPackages)
	assert.Equal(t, 1, s.NarrowedPackages)
	assert.Equal(t, 2, s.SelectedTests)
	assert.True(t, s.NarrowingEnabled)
}

func TestNewSummaryReductionCountsEffectivePackages(t *testing.T) {
	s := NewSummary(narrowedInputs())

	assert.InDelta(t, 80.0, s.ReductionPercent, 0.001,
		"one full plus one narrowed package out of ten testable is an 80% reduction")
}

func TestNewSummaryDroppedPackagesAreNotRun(t *testing.T) {
	in := narrowedInputs()
	in.Narrowed.Plans = append(in.Narrowed.Plans, selection.PackagePlan{
		ImportPath: "example.com/untouched",
		Reason:     selection.ReasonDependent,
		AllTests:   []string{"TestNothing"},
	})
	in.Narrowed.DroppedPackages = 1

	s := NewSummary(in)

	assert.Equal(t, 1, s.DroppedPackages)
	assert.Equal(t, "dropped", s.SelectedPackages[len(s.SelectedPackages)-1].Mode())
}

func TestNewSummaryHandlesNoTestablePackages(t *testing.T) {
	s := NewSummary(Inputs{})

	assert.Zero(t, s.ReductionPercent)
	assert.Empty(t, s.SelectedPackages)
}

func TestPrinterSummarizesCounts(t *testing.T) {
	out := render(t, NewSummary(narrowedInputs()), false)

	for _, want := range []string{
		"40 packages", "10 with tests", "2 selected", "80.0% skipped",
		"line-level:", "1 full", "1 narrowed to 2 tests",
		"no line attribution: 1 files",
	} {
		assert.Contains(t, out, want)
	}
	assert.NotContains(t, out, "example.com/app", "non-verbose output must not list packages")
}

func TestPrinterVerboseShowsModeAndReason(t *testing.T) {
	out := render(t, NewSummary(narrowedInputs()), true)

	for _, want := range []string{
		"narrowed", "example.com/repo", "direct", "2 tests",
		"full", "example.com/app", "dependent", "no index record",
		"design.md", "gen.go",
	} {
		assert.Contains(t, out, want)
	}
}

func TestPrinterReportsNarrowingDisabled(t *testing.T) {
	in := narrowedInputs()
	in.Narrowed.Enabled = false
	in.Narrowed.DisabledReason = "no index available"

	out := render(t, NewSummary(in), false)

	assert.Contains(t, out, "line-level: off — no index available")
}

func TestPrinterReportsSelectAllAndNote(t *testing.T) {
	in := Inputs{
		Narrowed: selection.Narrowed{
			Result: selection.Result{SelectAll: true, SelectAllReason: "module definition changed: go.mod"},
		},
		Note:             "no merge base with \"main\"; comparing directly, which over-selects",
		TotalPackages:    40,
		TestablePackages: 10,
	}

	out := render(t, NewSummary(in), false)

	assert.Contains(t, out, "select-all: module definition changed: go.mod")
	assert.Contains(t, out, "warning: no merge base")
}

func TestPrinterEmitsAnsiOnlyWhenColorEnabled(t *testing.T) {
	s := NewSummary(narrowedInputs())

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

func TestWriteJSONCarriesNarrowingDetail(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, WriteJSON(&buf, NewSummary(narrowedInputs())))

	var decoded Summary
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))

	require.Len(t, decoded.SelectedPackages, 2)
	assert.True(t, decoded.NarrowingEnabled)
	assert.Equal(t, 2, decoded.SelectedTests)

	byPath := make(map[string]SelectedPackage, len(decoded.SelectedPackages))
	for _, pkg := range decoded.SelectedPackages {
		byPath[pkg.ImportPath] = pkg
	}
	assert.Equal(t, []string{"TestGet", "TestPut"}, byPath["example.com/repo"].Tests)
	assert.Equal(t, "no index record for this package", byPath["example.com/app"].FullReason)
}

func TestSelectedPackageMode(t *testing.T) {
	assert.Equal(t, "full", SelectedPackage{FullReason: "x"}.Mode())
	assert.Equal(t, "narrowed", SelectedPackage{Tests: []string{"TestA"}}.Mode())
	assert.Equal(t, "dropped", SelectedPackage{}.Mode())
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
