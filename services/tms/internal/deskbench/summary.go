package deskbench

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

type CompareEntry struct {
	Key            string   `json:"key"`
	BeforePassRate float64  `json:"beforePassRate"`
	AfterPassRate  float64  `json:"afterPassRate"`
	BeforePassK    *float64 `json:"beforePassK,omitempty"`
	AfterPassK     *float64 `json:"afterPassK,omitempty"`
	BeforeRuns     int      `json:"beforeRuns"`
	AfterRuns      int      `json:"afterRuns"`
	BeforeRefused  float64  `json:"beforeRefusedPerRun"`
	AfterRefused   float64  `json:"afterRefusedPerRun"`
	BeforeTokens   float64  `json:"beforeTokensPerRun"`
	AfterTokens    float64  `json:"afterTokensPerRun"`
	BeforeCost     *float64 `json:"beforeCostPerRun,omitempty"`
	AfterCost      *float64 `json:"afterCostPerRun,omitempty"`
	Change         string   `json:"change"`
}

const (
	changeBetter = "better"
	changeWorse  = "worse"
	changeSame   = "same"
	changeNew    = "new"
	changeGone   = "gone"
)

func costPerRun(row *MatrixRow) *float64 {
	if row.Measures.CostUSD.N == 0 || row.Measures.CostUnpriced {
		return nil
	}
	mean := row.Measures.CostUSD.Mean

	return &mean
}

func Compare(before, after *Report) []CompareEntry {
	previous := make(map[string]MatrixRow, len(before.Summary.Matrix))
	for _, row := range before.Summary.Matrix {
		previous[row.Key()] = row
	}

	entries := make([]CompareEntry, 0, len(after.Summary.Matrix))
	seen := make(map[string]bool, len(after.Summary.Matrix))
	for idx := range after.Summary.Matrix {
		row := &after.Summary.Matrix[idx]
		key := row.Key()
		seen[key] = true
		entry := CompareEntry{
			Key:           key,
			AfterPassRate: row.PassRate(),
			AfterPassK:    row.PassK,
			AfterRuns:     row.Runs,
			AfterRefused:  row.refusedPerRun(after.SchemaVersion),
			AfterTokens:   row.tokensPerRun(after.SchemaVersion),
			AfterCost:     costPerRun(row),
		}
		old, ok := previous[key]
		if !ok {
			entry.Change = changeNew
			entries = append(entries, entry)
			continue
		}
		entry.BeforePassRate = old.PassRate()
		entry.BeforePassK = old.PassK
		entry.BeforeRuns = old.Runs
		entry.BeforeRefused = old.refusedPerRun(before.SchemaVersion)
		entry.BeforeTokens = old.tokensPerRun(before.SchemaVersion)
		entry.BeforeCost = costPerRun(&old)
		entry.Change = changeOf(&entry)
		entries = append(entries, entry)
	}
	for idx := range before.Summary.Matrix {
		row := &before.Summary.Matrix[idx]
		if seen[row.Key()] {
			continue
		}
		entries = append(entries, CompareEntry{
			Key:            row.Key(),
			BeforePassRate: row.PassRate(),
			BeforePassK:    row.PassK,
			BeforeRuns:     row.Runs,
			BeforeRefused:  row.refusedPerRun(before.SchemaVersion),
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
	case entry.AfterRefused < entry.BeforeRefused:
		return changeBetter
	case entry.AfterRefused > entry.BeforeRefused:
		return changeWorse
	default:
		return changeSame
	}
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
	if r.Meta.Threshold != nil {
		fmt.Fprintf(&b, "Tool disclosure threshold %d\n", *r.Meta.Threshold)
	}
	b.WriteString("\nMeans per trial with 95% intervals. By model, pass is the mean of each " +
		"scenario's pass rate with a t-interval over scenarios, since a scenario's trials rise " +
		"and fall together (report.json keeps the pooled Wilson rate as pass); by scenario it is " +
		"a Wilson interval over that scenario's trials. pass^" + strconv.Itoa(summary.PassK) +
		" is the chance every one of " + strconv.Itoa(summary.PassK) +
		" trials passes. Tokens: in is input the cache did not serve.\n")

	renderCells(&b, summary.Cells, summary.PassK)
	renderMatrix(&b, summary.Matrix, summary.PassK)
	renderComparison(&b, r.Comparison)
	renderTroubles(&b, r)

	return b.String()
}

func renderCells(b *strings.Builder, cells []CellRow, k int) {
	if len(cells) == 0 {
		return
	}

	b.WriteString("\nBy model:\n")
	table := tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "MODEL\tSPLIT\tSCEN\tTRIALS\tPASS\tPASS^%d\tS/TASK\tTTFT MS\t"+
		"MODEL CALLS\tTOOL CALLS\tREFUSED\tIN\tCACHE R\tCACHE W\tOUT\tCACHED\t$/TASK\t$/PASS\n", k)
	for idx := range cells {
		cell := &cells[idx]
		m := &cell.Measures
		fmt.Fprintf(table, strings.Repeat("%s\t", 2)+"%d\t%d\t"+strings.Repeat("%s\t", 13)+"%s\n",
			cell.Provider, splitLabel(cell.Split), cell.Scenarios, cell.Trials,
			percentStatCell(cell.ScenarioPass),
			statCell(cell.PassK, 2), statCell(m.Seconds, 1), statCell(m.FirstTokenMs, 0),
			statCell(m.ModelCalls, 1), statCell(m.ToolCalls, 1), statCell(m.Refused, 2),
			statCell(m.UncachedInput, 0), statCell(m.CacheRead, 0), statCell(m.CacheWrite, 0),
			statCell(m.Output, 0), percentCell(m.CacheReadShare), costCell(m), costPassCell(m))
	}
	_ = table.Flush()

	b.WriteString("\nTool selection and retrieval by model:\n")
	table = tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(table, "MODEL\tSPLIT\tPRECISION\tRECALL\tFIRST RIGHT\tFORBIDDEN\t"+
		"PRESELECTION R@8\tFIND RECALL\tFIND HITS\tREFUSED BY VERDICT\tRE-ASKS BY GUARD")
	for idx := range cells {
		cell := &cells[idx]
		m := &cell.Measures
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			cell.Provider, splitLabel(cell.Split), statCell(m.Precision, 2),
			statCell(m.Recall, 2), statCell(m.FirstCorrect, 1), statCell(m.ForbiddenCalls, 2),
			statCell(m.PreselectionRecall, 2), statCell(m.FindRecall, 2),
			percentCell(m.FindHitRate), perTrialCounts(m.RefusedByVerdict),
			perTrialCounts(m.ReasksByGuard))
	}
	_ = table.Flush()
}

func renderMatrix(b *strings.Builder, rows []MatrixRow, k int) {
	b.WriteString("\nBy scenario:\n")
	table := tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "SCENARIO\tMODEL\tSPLIT\tPASS\tPASS^%d\tAVG S\tMODEL CALLS\t"+
		"TOOL CALLS\tREFUSED\tIN/CACHE R/CACHE W/OUT\t$/TASK\tRECALL\n", k)
	for idx := range rows {
		row := &rows[idx]
		m := &row.Measures
		fmt.Fprintf(
			table,
			"%s\t%s\t%s\t%d/%d\t%s\t%.1f\t%.1f\t%.1f\t%.1f\t%.0f/%.0f/%.0f/%.0f\t%s\t%s\n",
			row.Scenario,
			row.Provider,
			splitLabel(row.Split),
			row.Passed,
			row.Runs,
			optionalFixed(row.PassK, 2),
			m.Seconds.Mean,
			m.ModelCalls.Mean,
			m.ToolCalls.Mean,
			m.Refused.Mean,
			m.UncachedInput.Mean,
			m.CacheRead.Mean,
			m.CacheWrite.Mean,
			m.Output.Mean,
			costCell(m),
			meanCell(m.Recall, 2),
		)
	}
	_ = table.Flush()
}

func renderComparison(b *strings.Builder, entries []CompareEntry) {
	if len(entries) == 0 {
		return
	}

	b.WriteString("\nAgainst the earlier run:\n")
	for idx := range entries {
		entry := &entries[idx]
		if entry.Change == changeSame {
			continue
		}
		fmt.Fprintf(
			b,
			"  %-6s %s: pass %.0f%% → %.0f%%, refused per run %.2f → %.2f, tokens per run %.0f → %.0f\n",
			strings.ToUpper(entry.Change),
			entry.Key,
			entry.BeforePassRate*100,
			entry.AfterPassRate*100,
			entry.BeforeRefused,
			entry.AfterRefused,
			entry.BeforeTokens,
			entry.AfterTokens,
		)
	}
}

func renderTroubles(b *strings.Builder, r *Report) {
	troubled := slices.DeleteFunc(
		slices.Clone(r.ToolHealth),
		func(h ToolHealth) bool { return h.Refused == 0 },
	)
	if len(troubled) > 0 {
		b.WriteString("\nTools the harness refused or that failed:\n")
		for _, health := range troubled {
			fmt.Fprintf(b, "  %s: %d of %d calls (%s)\n", health.Tool, health.Refused, health.Calls,
				verdictCounts(health.Verdicts))
			for _, reason := range health.Reasons {
				fmt.Fprintf(b, "    - %s\n", reason)
			}
		}
	}

	if len(r.ModelNotes) > 0 {
		b.WriteString("\nModel calls worth a look:\n")
		for _, note := range r.ModelNotes {
			fmt.Fprintf(b, "  %s turn %s call #%d (%s): %s\n", note.Case, note.TurnID, note.Seq,
				note.Model, note.Note)
		}
	}

	if failing := r.Summary.Failing; len(failing) > 0 {
		b.WriteString("\nFailing checks:\n")
		for _, item := range failing {
			fmt.Fprintf(b, "  %s step %d (%s): %s", item.Case, item.Step, item.Label, item.Check)
			if item.Detail != "" {
				fmt.Fprintf(b, " — %s", item.Detail)
			}
			b.WriteString("\n")
		}
	}
}

func splitLabel(split Split) string {
	if split == "" {
		return "all"
	}

	return string(split)
}

func statCell(stat Stat, places int) string {
	if stat.N == 0 {
		return "–"
	}
	if stat.N == 1 || stat.Low == stat.High {
		return strconv.FormatFloat(stat.Mean, 'f', places, 64)
	}

	return fmt.Sprintf("%s [%s, %s]", strconv.FormatFloat(stat.Mean, 'f', places, 64),
		strconv.FormatFloat(stat.Low, 'f', places, 64),
		strconv.FormatFloat(stat.High, 'f', places, 64))
}

func meanCell(stat Stat, places int) string {
	if stat.N == 0 {
		return "–"
	}

	return strconv.FormatFloat(stat.Mean, 'f', places, 64)
}

func percentStatCell(stat Stat) string {
	if stat.N == 0 {
		return "–"
	}
	if stat.N == 1 || stat.Low == stat.High {
		return fmt.Sprintf("%.0f%%", stat.Mean*100)
	}

	return fmt.Sprintf("%.0f%% [%.0f, %.0f]", stat.Mean*100, stat.Low*100, stat.High*100)
}

func percentCell(value *float64) string {
	if value == nil {
		return "–"
	}

	return fmt.Sprintf("%.0f%%", *value*100)
}

func optionalFixed(value *float64, places int) string {
	if value == nil {
		return "–"
	}

	return strconv.FormatFloat(*value, 'f', places, 64)
}

func costCell(m *Measures) string {
	if m.CostUnpriced || m.CostUSD.N == 0 {
		return "unpriced"
	}
	cell := "$" + strconv.FormatFloat(m.CostUSD.Mean, 'f', 4, 64)
	if m.CostEstimated {
		cell += "~"
	}

	return cell
}

func costPassCell(m *Measures) string {
	if m.CostPerPass == nil {
		return "–"
	}
	cell := "$" + strconv.FormatFloat(*m.CostPerPass, 'f', 4, 64)
	if m.CostEstimated {
		cell += "~"
	}

	return cell
}

func perTrialCounts(counts map[string]float64) string {
	if len(counts) == 0 {
		return "–"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s %.2f", key, counts[key]))
	}

	return strings.Join(parts, ", ")
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
