package deskbench

import (
	"fmt"
	"strings"
	"text/tabwriter"
)

func figureCells(figure *GateFigure) (string, string) {
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
	fmt.Fprintf(&b, ", %s to %s. Decisions read development scenarios only; held-out figures "+
		"are kept for the confirming run.\n", r.StartedAt.Format("2006-01-02 15:04"),
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
		for _, decision := range r.Advisors {
			fmt.Fprintf(&b, "- %s: offered %s. %s\n", decision.Route, yesNo(decision.Offered),
				decision.Reason)
			for _, check := range decision.Checks {
				fmt.Fprintf(&b, "  - %s: %s (%s)\n", check.Name, yesNo(check.Passed), check.Detail)
			}
		}
	}

	b.WriteString("\n## (a) Code mode\n\n")
	if len(r.CodeMode) == 0 {
		b.WriteString("No code-mode pair is in the spec, so code mode stays off for every model.\n")
	}
	for idx := range r.CodeMode {
		decision := &r.CodeMode[idx]
		fmt.Fprintf(&b, "- %s (%s against %s): enabled %s. %s\n", decision.Model, decision.Code,
			decision.Discrete, yesNo(decision.Enabled), decision.Reason)
		for _, check := range decision.Checks {
			fmt.Fprintf(&b, "  - %s: %s (%s)\n", check.Name, yesNo(check.Passed), check.Detail)
		}
	}

	b.WriteString("\n## The configuration run 2 confirms\n\n")
	for _, route := range AllRoutes() {
		fmt.Fprintf(&b, "- %s: %s\n", route, r.Configuration[route])
	}
	for idx := range r.CodeMode {
		if r.CodeMode[idx].Enabled {
			fmt.Fprintf(&b, "- code mode on for %s\n", r.CodeMode[idx].Model)
		}
	}

	b.WriteString("\n## Held out, as run 1 measured it\n\n")
	beforePass, beforeCost := figureCells(&r.HeldOut.Before)
	chosenPass, chosenCost := figureCells(&r.HeldOut.Chosen)
	fmt.Fprintf(&b, "- before (%s): pass^%d %s, %s per completed task, %d scenarios\n",
		r.Spec.Before, k, beforePass, beforeCost, r.HeldOut.Before.Scenarios)
	fmt.Fprintf(&b, "- chosen configuration, without rewrites: pass^%d %s, %s per completed "+
		"task\n", k, chosenPass, chosenCost)

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
	fmt.Fprintln(table, "\tPASS^K\t$/PASS\tCACHED 2+\tFIRST RIGHT\tRECALL\tARGS VALID\tUNAUTHORIZED")
	for _, row := range []struct {
		name    string
		figures *HeadlineFigures
	}{{"before", &c.Before}, {"after", &c.After}} {
		pass, cost := figureCells(&row.figures.Figure)
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\n", row.name, pass, cost,
			percentCell(row.figures.LaterCacheShare), fixedOrDash(row.figures.FirstCorrect),
			fixedOrDash(row.figures.Recall), fixedOrDash(row.figures.ArgumentValidity),
			row.figures.Unauthorized)
	}
	_ = table.Flush()

	b.WriteString("\n## Routes\n\n")
	for _, route := range c.Routes {
		fmt.Fprintf(&b, "- %s on %s: holds %s. %s\n", route.Route, route.Cell,
			yesNo(route.Holds), route.Detail)
	}

	fmt.Fprintf(&b, "\n## Descriptions\n\n- %s: %s. %s\n", c.Descriptions.Name,
		yesNo(c.Descriptions.Passed), c.Descriptions.Detail)

	b.WriteString("\n## Code mode\n\n")
	if len(c.CodeMode) == 0 {
		b.WriteString("No model had code mode enabled by run 1.\n")
	}
	for _, check := range c.CodeMode {
		fmt.Fprintf(&b, "- %s: holds %s. %s\n", check.Model, yesNo(check.Holds), check.Detail)
	}

	b.WriteString("\n## Configuration kept\n\n")
	for _, route := range AllRoutes() {
		fmt.Fprintf(&b, "- %s: %s\n", route, c.Configuration[route])
	}

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
