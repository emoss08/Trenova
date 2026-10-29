package benchgate

import (
	"fmt"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/emoss08/trenova/internal/deskbench/benchreport"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
)

func figureCells(figure *GateFigure) (pass, cost string) {
	return fixedOrDash(figure.PassK), dollarsOrDash(figure.CostPerPass)
}

func dollarsOrDash(value *float64) string {
	if value == nil {
		return "-"
	}

	return fmt.Sprintf("$%.4f", *value)
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}

func (r *GateResult) Render() string {
	var b strings.Builder
	k := r.Spec.PassK
	fmt.Fprintf(&b, "# Desk bench gate\n\nRun %s", r.Run)
	if r.Commit != "" {
		fmt.Fprintf(&b, " at %s", r.Commit)
	}
	fmt.Fprintf(&b, ", %s to %s. Recommendations read development scenarios only; "+
		"held-out figures are kept for the confirming run.\n", r.StartedAt.Format("2006-01-02 15:04"),
		r.FinishedAt.Format("15:04"))

	b.WriteString("\n## Safety\n\n")
	table := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(table, "CELL\tTRIALS\tUNAUTHORIZED\tNOT AUDITED\tSAFE")
	for _, cell := range r.Spec.Cells() {
		safety := r.Safety[cell]
		fmt.Fprintf(table, "%s\t%d\t%d\t%d\t%s\n", cell, safety.Trials, safety.Unauthorized,
			safety.Unaudited, yesNo(safety.Safe()))
	}
	_ = table.Flush()

	fmt.Fprintf(&b, "\n## (c) Effort per route\n\nPass^%d and cost per completed task on each "+
		"route's development scenarios.\n\n", k)
	table = tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "ROUTE\tCANDIDATE\tSCEN\tTRIALS\tPASS^%d\t$/PASS\tCHOSEN\n", k)
	for _, decision := range r.Efforts {
		for idx := range decision.Candidates {
			figure := &decision.Candidates[idx]
			pass, cost := figureCells(figure)
			fmt.Fprintf(table, "%s\t%s\t%d\t%d\t%s\t%s\t%s\n", decision.Route, figure.Cell,
				figure.Scenarios, figure.Trials, pass, cost, yesNo(figure.Cell == decision.Chosen))
		}
	}
	_ = table.Flush()
	for _, decision := range r.Efforts {
		fmt.Fprintf(&b, "- %s: %s. %s\n", decision.Route, decision.Chosen, decision.Reason)
	}

	if len(r.Advisors) > 0 {
		b.WriteString("\n## (c) The advisor\n\n")
		for idx := range r.Advisors {
			decision := &r.Advisors[idx]
			fmt.Fprintf(&b, "- %s: offered %s. %s\n", decision.Route, yesNo(decision.Offered),
				decision.Reason)
			renderChecks(&b, decision.Checks)
		}
	}

	r.renderCodeMode(&b)

	b.WriteString("\n## The configuration run 2 confirms\n\n")
	for _, route := range benchscenario.AllRoutes() {
		fmt.Fprintf(&b, "- %s: %s\n", route, r.Configuration[route])
	}

	r.renderHeldOut(&b)
	renderRecommendations(&b, r.Recommendations())

	return b.String()
}

func (c *GateConfirmation) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Desk bench gate: held-out confirmation\n\nRun %s against gate %s",
		c.Run, c.Gate)
	if c.Commit != "" {
		fmt.Fprintf(&b, " at %s", c.Commit)
	}
	b.WriteString(".\n\n## Headline\n\n")
	table := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(table,
		"\tPASS^K\t$/PASS\tCACHED 2+\tFIRST RIGHT\tRECALL\tARGS VALID\tUNAUTHORIZED")
	for _, row := range []struct {
		name    string
		figures *HeadlineFigures
	}{{"before", &c.Before}, {"after", &c.After}} {
		pass, cost := figureCells(&row.figures.Figure)
		fmt.Fprintf(
			table,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\n",
			row.name,
			pass,
			cost,
			benchreport.PercentCell(
				row.figures.LaterCacheShare,
			),
			fixedOrDash(row.figures.FirstCorrect),
			fixedOrDash(row.figures.Recall),
			fixedOrDash(row.figures.ArgumentValidity),
			row.figures.Unauthorized,
		)
	}
	_ = table.Flush()

	b.WriteString("\n## Routes\n\n")
	for idx := range c.Routes {
		route := &c.Routes[idx]
		fmt.Fprintf(&b, "- %s on %s: holds %s. %s\n", route.Route, route.Cell,
			yesNo(route.Holds), route.Detail)
	}

	fmt.Fprintf(&b, "\n## Descriptions\n\n- %s: %s. %s\n", c.Descriptions.Name,
		yesNo(c.Descriptions.Passed), c.Descriptions.Detail)

	b.WriteString("\n## Configuration kept\n\n")
	for _, route := range benchscenario.AllRoutes() {
		fmt.Fprintf(&b, "- %s: %s\n", route, c.Configuration[route])
	}

	c.renderCodeMode(&b)
	renderRecommendations(&b, c.Recommendations())

	return b.String()
}

func (p *Projection) Render() string {
	var b strings.Builder
	b.WriteString("# Desk bench gate: budget probe\n\n")
	table := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(table, "GROUP\tCELL\tPROBED\t$/TRIAL\tS/TRIAL\tPLANNED\tCOST\tHOURS")
	for _, cell := range p.Cells {
		cost := fmt.Sprintf("$%.4f", cell.CostPerTrial)
		if cell.Unpriced {
			cost = "unpriced"
		}
		fmt.Fprintf(table, "%s\t%s\t%d\t%s\t%.1f\t%d\t$%.2f\t%.2f\n", cell.Group, cell.Cell,
			cell.ProbeTrials, cost, cell.SecondsPerRun, cell.Planned, cell.Cost, cell.Hours)
	}
	_ = table.Flush()
	fmt.Fprintf(&b, "\nProjected: $%.2f and %.1f hours at %d cases at once, against a budget of "+
		"$%.2f and %.1f hours. Within budget: %s. The run stops at $%.2f.\n",
		p.Cost, p.Hours, p.Parallel, p.MaxUSD, p.MaxHours, yesNo(p.Within), p.Cap)

	return b.String()
}

func (r *GateResult) renderHeldOut(b *strings.Builder) {
	k := r.Spec.PassK
	b.WriteString("\n## Held out, as run 1 measured it\n\n")
	beforePass, beforeCost := figureCells(&r.HeldOut.Before)
	chosenPass, chosenCost := figureCells(&r.HeldOut.Chosen)
	fmt.Fprintf(b, "- before (%s): pass^%d %s, %s per completed task, %d scenarios\n",
		r.Spec.Before, k, beforePass, beforeCost, r.HeldOut.Before.Scenarios)
	fmt.Fprintf(b, "- chosen configuration, without rewrites: pass^%d %s, %s per completed "+
		"task\n", k, chosenPass, chosenCost)
}

func RenderFigures(r *benchreport.Report) string {
	var b strings.Builder
	k := r.Summary.PassK
	fmt.Fprintf(&b, "# Desk bench gate: figures\n\nCells only, by split. Scenario detail for the "+
		"development split is in development/; the held-out and adversarial splits' detail stays "+
		"under sealed/ and is never handed over.\n\n")
	table := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "CELL\tSPLIT\tSCEN\tTRIALS\tPASS^%d\t$/PASS\tUNAUTHORIZED\n", k)
	for idx := range r.Summary.Cells {
		cell := &r.Summary.Cells[idx]
		fmt.Fprintf(
			table,
			"%s\t%s\t%d\t%d\t%s\t%s\t%s\n",
			cell.Provider,
			benchreport.SplitLabel(
				cell.Split,
			),
			cell.Scenarios,
			cell.Trials,
			benchreport.StatCell(cell.PassK, 2),
			benchreport.CostPassCell(&cell.Measures),
			benchreport.UnauthorizedCell(&cell.Measures),
		)
	}
	_ = table.Flush()

	return b.String()
}

func (r *GateResult) renderCodeMode(b *strings.Builder) {
	if len(r.CodeMode) == 0 {
		return
	}
	k := r.Spec.PassK
	fmt.Fprintf(b, "\n## (a) Code mode, per model\n\nHeld-out pairs, discrete against script "+
		"mode; the adversarial pairs for safety. A recommended model is re-checked in run 2; "+
		"what holds is a recommendation to turn on Scripts for that model's provider.\n\n")
	table := tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "CELL\tKIND\tPAIRS\tDISCRETE PASS^%d\tSCRIPT PASS^%d\tDISCRETE $/PASS\t"+
		"SCRIPT $/PASS\n", k, k)
	for idx := range r.CodeMode {
		decision := &r.CodeMode[idx]
		rows := append(slices.Clone(decision.Work), decision.AllWork, decision.Controls)
		rows[len(rows)-2].Kind = "all work"
		rows[len(rows)-1].Kind = "controls"
		for at := range rows {
			writePairRow(table, decision.Cell, &rows[at])
		}
	}
	_ = table.Flush()
	for idx := range r.CodeMode {
		decision := &r.CodeMode[idx]
		fmt.Fprintf(b, "- %s: recommended %s; scripts may propose writes %s. %s\n", decision.Cell,
			yesNo(decision.Recommended), yesNo(decision.WritesRecommended), decision.Reason)
		renderChecks(b, decision.Checks)
	}
}

func writePairRow(table *tabwriter.Writer, cell string, row *PairFigures) {
	discretePass, discreteCost := figureCells(&row.Discrete)
	scriptPass, scriptCost := figureCells(&row.Script)
	fmt.Fprintf(table, "%s\t%s\t%d\t%s\t%s\t%s\t%s\n", cell, row.Kind, row.Pairs,
		discretePass, scriptPass, discreteCost, scriptCost)
}

func renderChecks(b *strings.Builder, checks []GateCheck) {
	for _, check := range checks {
		fmt.Fprintf(b, "  - %s: %s (%s)\n", check.Name, yesNo(check.Passed), check.Detail)
	}
}

func (c *GateConfirmation) renderCodeMode(b *strings.Builder) {
	if len(c.CodeMode) == 0 {
		return
	}
	b.WriteString("\n## (a) Code mode, re-checked\n\nEach model run 1 recommended, on its " +
		"configured cell where the configuration runs it. A model that holds is recommended " +
		"under \"Recommended settings\" below.\n\n")
	for idx := range c.CodeMode {
		recheck := &c.CodeMode[idx]
		fmt.Fprintf(b, "- %s %s on %s (run 1: %s): holds %s. %s\n", recheck.Model.Provider,
			recheck.Model.Model, recheck.Cell, recheck.Run1, yesNo(recheck.Holds),
			recheck.Run2.Reason)
		renderChecks(b, recheck.Run2.Checks)
	}
}
