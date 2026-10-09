package deskbench

import (
	"fmt"
	"slices"
	"strings"
	"text/tabwriter"
	"time"
)

type CompareEntry struct {
	Key            string  `json:"key"`
	BeforePassRate float64 `json:"beforePassRate"`
	AfterPassRate  float64 `json:"afterPassRate"`
	BeforeRuns     int     `json:"beforeRuns"`
	AfterRuns      int     `json:"afterRuns"`
	BeforeRefused  int     `json:"beforeRefused"`
	AfterRefused   int     `json:"afterRefused"`
	BeforeTokens   int     `json:"beforeTokens"`
	AfterTokens    int     `json:"afterTokens"`
	Change         string  `json:"change"`
}

const (
	changeBetter = "better"
	changeWorse  = "worse"
	changeSame   = "same"
	changeNew    = "new"
	changeGone   = "gone"
)

func Compare(before, after *Report) []CompareEntry {
	previous := make(map[string]MatrixRow, len(before.Summary.Matrix))
	for _, row := range before.Summary.Matrix {
		previous[row.Key()] = row
	}

	entries := make([]CompareEntry, 0, len(after.Summary.Matrix))
	seen := make(map[string]bool, len(after.Summary.Matrix))
	for _, row := range after.Summary.Matrix {
		key := row.Key()
		seen[key] = true
		entry := CompareEntry{
			Key:           key,
			AfterPassRate: row.PassRate(),
			AfterRuns:     row.Runs,
			AfterRefused:  row.RefusedCalls,
			AfterTokens:   row.InputTokens + row.OutputTokens,
		}
		old, ok := previous[key]
		if !ok {
			entry.Change = changeNew
			entries = append(entries, entry)
			continue
		}
		entry.BeforePassRate = old.PassRate()
		entry.BeforeRuns = old.Runs
		entry.BeforeRefused = old.RefusedCalls
		entry.BeforeTokens = old.InputTokens + old.OutputTokens
		entry.Change = changeOf(&entry)
		entries = append(entries, entry)
	}
	for _, row := range before.Summary.Matrix {
		if seen[row.Key()] {
			continue
		}
		entries = append(entries, CompareEntry{
			Key:            row.Key(),
			BeforePassRate: row.PassRate(),
			BeforeRuns:     row.Runs,
			BeforeRefused:  row.RefusedCalls,
			Change:         changeGone,
		})
	}

	return entries
}

func changeOf(entry *CompareEntry) string {
	switch {
	case entry.AfterPassRate > entry.BeforePassRate:
		return changeBetter
	case entry.AfterPassRate < entry.BeforePassRate:
		return changeWorse
	case refusalRate(entry.AfterRefused, entry.AfterRuns) < refusalRate(entry.BeforeRefused, entry.BeforeRuns):
		return changeBetter
	case refusalRate(entry.AfterRefused, entry.AfterRuns) > refusalRate(entry.BeforeRefused, entry.BeforeRuns):
		return changeWorse
	default:
		return changeSame
	}
}

func refusalRate(refused, runs int) float64 {
	if runs == 0 {
		return 0
	}

	return float64(refused) / float64(runs)
}

func RenderSummary(r *Report) string {
	var b strings.Builder

	summary := r.Summary
	fmt.Fprintf(&b, "Desk bench: %d/%d cases passed in %s · %s\n",
		summary.Passed, summary.Cases, r.Meta.FinishedAt.Sub(r.Meta.StartedAt).Round(time.Second),
		usageLine(summary.Usage))
	if r.Meta.Commit != "" {
		fmt.Fprintf(&b, "Commit %s\n", r.Meta.Commit)
	}
	b.WriteString("\n")

	table := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(table, "SCENARIO\tMODEL\tPASS\tAVG S\tTOOL CALLS\tREFUSED\tTOKENS IN/OUT")
	for _, row := range summary.Matrix {
		fmt.Fprintf(table, "%s\t%s\t%d/%d\t%.1f\t%.1f\t%d\t%d/%d\n",
			row.Scenario, row.Provider, row.Passed, row.Runs, row.AvgSeconds, row.AvgToolCalls,
			row.RefusedCalls, row.InputTokens, row.OutputTokens)
	}
	_ = table.Flush()

	if len(r.Comparison) > 0 {
		b.WriteString("\nAgainst the earlier run:\n")
		for _, entry := range r.Comparison {
			if entry.Change == changeSame {
				continue
			}
			fmt.Fprintf(&b, "  %-6s %s: pass %.0f%% → %.0f%%, refused calls %d → %d, tokens %d → %d\n",
				strings.ToUpper(entry.Change), entry.Key, entry.BeforePassRate*100, entry.AfterPassRate*100,
				entry.BeforeRefused, entry.AfterRefused, entry.BeforeTokens, entry.AfterTokens)
		}
	}

	troubled := slices.DeleteFunc(slices.Clone(r.ToolHealth), func(h ToolHealth) bool { return h.Refused == 0 })
	if len(troubled) > 0 {
		b.WriteString("\nTools the harness refused or that failed:\n")
		for _, health := range troubled {
			fmt.Fprintf(&b, "  %s: %d of %d calls (%s)\n", health.Tool, health.Refused, health.Calls,
				verdictCounts(health.Verdicts))
			for _, reason := range health.Reasons {
				fmt.Fprintf(&b, "    - %s\n", reason)
			}
		}
	}

	if len(r.ModelNotes) > 0 {
		b.WriteString("\nModel calls worth a look:\n")
		for _, note := range r.ModelNotes {
			fmt.Fprintf(&b, "  %s turn %s call #%d (%s): %s\n", note.Case, note.TurnID, note.Seq,
				note.Model, note.Note)
		}
	}

	if len(summary.Failing) > 0 {
		b.WriteString("\nFailing checks:\n")
		for _, failing := range summary.Failing {
			fmt.Fprintf(&b, "  %s step %d (%s): %s", failing.Case, failing.Step, failing.Label, failing.Check)
			if failing.Detail != "" {
				fmt.Fprintf(&b, " — %s", failing.Detail)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func verdictCounts(verdicts map[string]int) string {
	keys := make([]string, 0, len(verdicts))
	for key := range verdicts {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s %d", key, verdicts[key]))
	}

	return strings.Join(parts, ", ")
}
