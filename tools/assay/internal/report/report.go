package report

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"

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

func WriteText(w io.Writer, s Summary, verbose bool) error {
	if s.SelectAll {
		if _, err := fmt.Fprintf(w, "select-all: %s\n", s.SelectAllReason); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w,
		"packages: %d total, %d with tests, %d selected (%.1f%% skipped)\n",
		s.TotalPackages, s.TestablePackages, len(s.SelectedPackages), s.ReductionPercent,
	); err != nil {
		return err
	}

	if len(s.ChangedPackages) > 0 {
		if _, err := fmt.Fprintf(w, "changed packages: %d\n", len(s.ChangedPackages)); err != nil {
			return err
		}
	}

	if len(s.IgnoredFiles) > 0 {
		if _, err := fmt.Fprintf(w, "ignored (no owning package): %d files\n", len(s.IgnoredFiles)); err != nil {
			return err
		}
		if verbose {
			for _, f := range s.IgnoredFiles {
				if _, err := fmt.Fprintf(w, "  - %s\n", f); err != nil {
					return err
				}
			}
		}
	}

	if !verbose {
		return nil
	}

	for _, pkg := range s.SelectedPackages {
		if _, err := fmt.Fprintf(w, "  %-10s %s\n", pkg.Reason, shorten(pkg.ImportPath)); err != nil {
			return err
		}
	}

	return nil
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
