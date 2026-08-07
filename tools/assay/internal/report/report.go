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
	ImportPath string   `json:"importPath"`
	Reason     string   `json:"reason"`
	Tests      []string `json:"tests,omitempty"`
	FullReason string   `json:"fullReason,omitempty"`
}

func (p SelectedPackage) Mode() string {
	switch {
	case p.FullReason != "":
		return "full"
	case len(p.Tests) > 0:
		return "narrowed"
	default:
		return "dropped"
	}
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
	Note             string            `json:"note,omitempty"`

	NarrowingEnabled  bool     `json:"narrowingEnabled"`
	NarrowingDisabled string   `json:"narrowingDisabled,omitempty"`
	FullPackages      int      `json:"fullPackages"`
	NarrowedPackages  int      `json:"narrowedPackages"`
	DroppedPackages   int      `json:"droppedPackages"`
	SelectedTests     int      `json:"selectedTests"`
	BlockingFiles     []string `json:"blockingFiles,omitempty"`
	BaseCommit        string   `json:"baseCommit,omitempty"`
}

type Inputs struct {
	Narrowed         selection.Narrowed
	TotalPackages    int
	TestablePackages int
	GraphFromCache   bool
	Note             string
	BaseCommit       string
}

func NewSummary(in Inputs) Summary {
	res := in.Narrowed.Result

	selected := make([]SelectedPackage, 0, len(in.Narrowed.Plans))
	for _, plan := range in.Narrowed.Plans {
		selected = append(selected, SelectedPackage{
			ImportPath: plan.ImportPath,
			Reason:     string(plan.Reason),
			Tests:      plan.Tests,
			FullReason: plan.FullReason,
		})
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].ImportPath < selected[j].ImportPath })

	full := 0
	for _, plan := range selected {
		if plan.Mode() == "full" {
			full++
		}
	}

	effective := full + in.Narrowed.NarrowedPackages
	reduction := 0.0
	if in.TestablePackages > 0 {
		reduction = 100 * (1 - float64(effective)/float64(in.TestablePackages))
	}

	return Summary{
		TotalPackages:     in.TotalPackages,
		TestablePackages:  in.TestablePackages,
		SelectedPackages:  selected,
		ChangedPackages:   res.ChangedPackages,
		IgnoredFiles:      res.Unattributed,
		SelectAll:         res.SelectAll,
		SelectAllReason:   res.SelectAllReason,
		ReductionPercent:  reduction,
		GraphFromCache:    in.GraphFromCache,
		Note:              in.Note,
		NarrowingEnabled:  in.Narrowed.Enabled,
		NarrowingDisabled: in.Narrowed.DisabledReason,
		FullPackages:      full,
		NarrowedPackages:  in.Narrowed.NarrowedPackages,
		DroppedPackages:   in.Narrowed.DroppedPackages,
		SelectedTests:     in.Narrowed.SelectedTests,
		BlockingFiles:     in.Narrowed.Blocked,
		BaseCommit:        in.BaseCommit,
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
	mode     map[string]func(...any) string
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
		mode: map[string]func(...any) string{
			"narrowed": tint(color.FgCyan),
			"full":     tint(color.FgYellow),
			"dropped":  tint(color.Faint),
		},
	}
}

func (p *Printer) Summary(s Summary, verbose bool) error {
	if s.Note != "" {
		if err := p.line("%s %s", p.warn("warning:"), s.Note); err != nil {
			return err
		}
	}

	if s.SelectAll {
		if err := p.line("%s %s", p.warn("select-all:"), s.SelectAllReason); err != nil {
			return err
		}
	}

	origin := "graph loaded"
	if s.GraphFromCache {
		origin = "graph cached"
	}

	effective := s.FullPackages + s.NarrowedPackages
	if err := p.line("%d packages %s %d with tests %s %s selected %s %s skipped %s %s",
		s.TotalPackages, p.muted("·"),
		s.TestablePackages, p.muted("·"),
		p.emphasis(effective), p.muted("·"),
		p.emphasis(fmt.Sprintf("%.1f%%", s.ReductionPercent)), p.muted("·"),
		p.muted(origin),
	); err != nil {
		return err
	}

	if s.NarrowingEnabled {
		if err := p.line("%s %d full %s %d narrowed to %s tests %s %d dropped",
			p.muted("line-level:"),
			s.FullPackages, p.muted("·"),
			s.NarrowedPackages, p.emphasis(s.SelectedTests), p.muted("·"),
			s.DroppedPackages,
		); err != nil {
			return err
		}
	} else if s.NarrowingDisabled != "" {
		if err := p.line("%s %s", p.muted("line-level: off —"), s.NarrowingDisabled); err != nil {
			return err
		}
	}

	if len(s.ChangedPackages) > 0 {
		if err := p.line("%s %d", p.muted("changed packages:"), len(s.ChangedPackages)); err != nil {
			return err
		}
	}

	if len(s.BlockingFiles) > 0 {
		if err := p.line("%s %d files", p.muted("no line attribution:"), len(s.BlockingFiles)); err != nil {
			return err
		}
		if verbose {
			for _, f := range s.BlockingFiles {
				if err := p.line("  %s", p.muted(f)); err != nil {
					return err
				}
			}
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
		mode := pkg.Mode()
		paint := p.mode[mode]
		if paint == nil {
			paint = p.muted
		}
		detail := pkg.Reason
		switch mode {
		case "full":
			detail += " · " + pkg.FullReason
		case "narrowed":
			detail += fmt.Sprintf(" · %d tests", len(pkg.Tests))
		}
		if err := p.line("  %s %s %s",
			paint(fmt.Sprintf("%-9s", mode)), shorten(pkg.ImportPath), p.muted(detail)); err != nil {
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
