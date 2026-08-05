package report

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"

	"github.com/fatih/color"

	"github.com/emoss08/assay/internal/selection"
)

type SelectedPackage struct {
	ImportPath string `json:"importPath"`
	Reason     string `json:"reason"`
}

type Summary struct {
	TotalPackages    int               `json:"totalPackages"`
	TestablePackages int               `json:"testablePackages"`
	SelectedPackages []SelectedPackage `json:"selectedPackages"`
	ChangedPackages  []string          `json:"changedPackages"`
	IgnoredFiles     []string          `json:"ignoredFiles,omitempty"`
	SelectAll        bool              `json:"selectAll"`
	SelectAllReason  string            `json:"selectAllReason,omitempty"`
	ReductionPercent float64           `json:"reductionPercent"`
	GraphFromCache   bool              `json:"graphFromCache"`
}

func NewSummary(res selection.Result, totalPackages, testablePackages int) Summary {
	selected := make([]SelectedPackage, 0, len(res.Packages))
	for _, importPath := range res.Packages {
		selected = append(selected, SelectedPackage{
			ImportPath: importPath,
			Reason:     string(res.Reasons[importPath]),
		})
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].ImportPath < selected[j].ImportPath })

	reduction := 0.0
	if testablePackages > 0 {
		reduction = 100 * (1 - float64(len(selected))/float64(testablePackages))
	}

	return Summary{
		TotalPackages:    totalPackages,
		TestablePackages: testablePackages,
		SelectedPackages: selected,
		ChangedPackages:  res.ChangedPackages,
		IgnoredFiles:     res.Unattributed,
		SelectAll:        res.SelectAll,
		SelectAllReason:  res.SelectAllReason,
		ReductionPercent: reduction,
	}
}

func WriteJSON(w io.Writer, s Summary) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(s)
}

type Printer struct {
	out      io.Writer
	warn     func(...any) string
	muted    func(...any) string
	emphasis func(...any) string
	reason   map[string]func(...any) string
}

func NewPrinter(out io.Writer, useColor bool) *Printer {
	tint := func(attrs ...color.Attribute) func(...any) string {
		c := color.New(attrs...)
		if useColor {
			c.EnableColor()
		} else {
			c.DisableColor()
		}

		return c.SprintFunc()
	}

	return &Printer{
		out:      out,
		warn:     tint(color.FgYellow),
		muted:    tint(color.Faint),
		emphasis: tint(color.Bold),
		reason: map[string]func(...any) string{
			string(selection.ReasonDirect):    tint(color.FgCyan),
			string(selection.ReasonDependent): tint(color.FgBlue),
			string(selection.ReasonFallback):  tint(color.FgYellow),
		},
	}
}

func (p *Printer) Summary(s Summary, verbose bool) error {
	if s.SelectAll {
		if err := p.line("%s %s", p.warn("select-all:"), s.SelectAllReason); err != nil {
			return err
		}
	}

	origin := "graph loaded"
	if s.GraphFromCache {
		origin = "graph cached"
	}

	if err := p.line("%d packages %s %d with tests %s %s selected %s %s skipped %s %s",
		s.TotalPackages, p.muted("·"),
		s.TestablePackages, p.muted("·"),
		p.emphasis(len(s.SelectedPackages)), p.muted("·"),
		p.emphasis(fmt.Sprintf("%.1f%%", s.ReductionPercent)), p.muted("·"),
		p.muted(origin),
	); err != nil {
		return err
	}

	if len(s.ChangedPackages) > 0 {
		if err := p.line("%s %d", p.muted("changed packages:"), len(s.ChangedPackages)); err != nil {
			return err
		}
	}

	if len(s.IgnoredFiles) > 0 {
		if err := p.line("%s %d files", p.muted("ignored (no owning package):"), len(s.IgnoredFiles)); err != nil {
			return err
		}
		if verbose {
			for _, f := range s.IgnoredFiles {
				if err := p.line("  %s", p.muted(f)); err != nil {
					return err
				}
			}
		}
	}

	if !verbose {
		return nil
	}

	for _, pkg := range s.SelectedPackages {
		paint := p.reason[pkg.Reason]
		if paint == nil {
			paint = p.muted
		}
		if err := p.line("  %s %s", paint(fmt.Sprintf("%-10s", pkg.Reason)), shorten(pkg.ImportPath)); err != nil {
			return err
		}
	}

	return nil
}

func (p *Printer) line(format string, args ...any) error {
	_, err := fmt.Fprintf(p.out, format+"\n", args...)

	return err
}

func shorten(importPath string) string {
	const maxSegments = 5
	segments := splitPath(importPath)
	if len(segments) <= maxSegments {
		return importPath
	}

	return ".../" + path.Join(segments[len(segments)-maxSegments:]...)
}

func splitPath(p string) []string {
	var out []string
	for p != "" && p != "." && p != "/" {
		dir, file := path.Split(p)
		if file != "" {
			out = append([]string{file}, out...)
		}
		p = path.Clean(dir)
		if p == "." || p == "/" {
			break
		}
	}

	return out
}
