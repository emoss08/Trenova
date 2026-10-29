package masterdatatools

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const moneyLineRunTotal = "Total"

type runView struct {
	Status         string `json:"status"`
	PeriodStart    int64  `json:"periodStart"`
	PeriodEnd      int64  `json:"periodEnd"`
	InvoiceDate    int64  `json:"invoiceDate"`
	GroupCount     int    `json:"groupCount"`
	ItemCount      int    `json:"itemCount"`
	ExcludedCount  int    `json:"excludedCount"`
	FailureReason  string `json:"failureReason,omitempty"`
	OffCycleReason string `json:"offCycleReason,omitempty"`
}

type proposedInvoiceView struct {
	CustomerID    pulid.ID `json:"customerId"`
	GroupLabel    string   `json:"groupLabel"`
	ShipmentCount int      `json:"shipmentCount"`
	Outcome       string   `json:"outcome,omitempty"`
	Reason        string   `json:"reason,omitempty"`
}

var runDateTypes = map[string]assistantartifact.DisplayType{
	toolparams.ParamPeriodStart: assistantartifact.DisplayDate,
	toolparams.ParamPeriodEnd:   assistantartifact.DisplayDate,
	toolparams.ParamInvoiceDate: assistantartifact.DisplayDate,
}

var proposedInvoiceRefs = map[string]permission.Resource{
	toolparams.ParamCustomerID: permission.ResourceCustomer,
}

func viewOfRun(run *invoicerun.InvoiceRun) *runView {
	return &runView{
		Status:         string(run.Status),
		PeriodStart:    run.PeriodStart,
		PeriodEnd:      run.PeriodEnd,
		InvoiceDate:    run.InvoiceDate,
		GroupCount:     run.GroupCount,
		ItemCount:      run.ItemCount,
		ExcludedCount:  run.ExcludedCount,
		FailureReason:  run.FailureReason,
		OffCycleReason: run.OffCycleReason,
	}
}

func runTotalMoney(change *agentpreview.RecordChange, before, after *invoicerun.InvoiceRun) {
	line := agentpreview.MoneyLine{
		Label: moneyLineRunTotal,
		After: changepreview.KnownAmount(after.TotalAmount),
	}
	if before != nil {
		line.Before = changepreview.KnownAmount(before.TotalAmount)
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(after.CurrencyCode, line),
		toolpreview.SensitiveAs("totalAmount"))
}

func proposedInvoiceChange(
	plan *invoicerunports.InvoiceRunGroupPlan,
	operation agentpreview.PreviewOperation,
	before decimal.NullDecimal,
) (*agentpreview.RecordChange, error) {
	view := &proposedInvoiceView{
		CustomerID:    plan.CustomerID,
		GroupLabel:    plan.GroupLabel,
		ShipmentCount: plan.ShipmentCount,
		Outcome:       string(plan.Outcome),
		Reason:        plan.Reason,
	}
	resource := permission.ResourceInvoice
	label := "New invoice: " + plan.GroupLabel
	if plan.Outcome != invoicerunports.InvoiceRunGroupBills {
		resource = permission.ResourceInvoiceRun
		label = "Not invoiced: " + plan.GroupLabel
	}

	change, err := toolpreview.Create(
		toolpreview.Record{Resource: resource, Label: label},
		view,
		toolpreview.WithRefs(proposedInvoiceRefs),
	)
	if err != nil {
		return nil, err
	}
	change.Operation = operation
	toolpreview.AttachMoney(
		change,
		toolpreview.MoneyBlock(plan.CurrencyCode, agentpreview.MoneyLine{
			Label:  moneyLineRunTotal,
			Before: before,
			After:  changepreview.KnownAmount(plan.Total),
		}),
		toolpreview.SensitiveAs("totalAmount"),
	)

	return change, nil
}

func groupPlansFromRun(run *invoicerun.InvoiceRun) []invoicerunports.InvoiceRunGroupPlan {
	plans := make([]invoicerunports.InvoiceRunGroupPlan, 0, len(run.Groups))
	for _, group := range run.Groups {
		if group == nil {
			continue
		}
		plans = append(plans, invoicerunports.InvoiceRunGroupPlan{
			GroupID:       group.ID,
			GroupLabel:    group.GroupLabel,
			CustomerID:    group.CustomerID,
			ShipmentCount: group.ItemCount,
			Total:         group.TotalAmount,
			CurrencyCode:  group.CurrencyCode,
			Outcome:       invoicerunports.InvoiceRunGroupBills,
		})
	}

	return plans
}

func RenderBuiltRun(
	_ *invoicerunports.PreviewInvoiceRunRequest,
	run *invoicerun.InvoiceRun,
) (*agentpreview.ToolPreview, error) {
	header, err := toolpreview.Create(invoicepreview.RunRecord(run), viewOfRun(run),
		toolpreview.Types(runDateTypes))
	if err != nil {
		return nil, err
	}
	runTotalMoney(header, nil, run)

	changes := []*agentpreview.RecordChange{header}
	plans := groupPlansFromRun(run)
	for idx := range plans {
		change, changeErr := proposedInvoiceChange(
			&plans[idx],
			agentpreview.PreviewOperationCreate,
			decimal.NullDecimal{},
		)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, change)
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would build an invoice run proposing %s for %s, %s in all; nothing is invoiced until "+
			"it is committed.",
		changepreview.CountOf(run.GroupCount, "invoice"),
		changepreview.CountOf(run.ItemCount, "shipment"),
		run.TotalAmount.StringFixed(2),
	), changes...), nil
}

func RenderMembership(
	_ *invoicerunports.AdjustInvoiceRunMembershipRequest,
	plan *invoicerunports.InvoiceRunChangePreview,
) (*agentpreview.ToolPreview, error) {
	header, err := toolpreview.Changed(
		invoicepreview.RunRecord(plan.Before),
		viewOfRun(plan.Before),
		viewOfRun(plan.After),
		toolpreview.Types(runDateTypes),
	)
	if err != nil {
		return nil, err
	}
	runTotalMoney(header, plan.Before, plan.After)

	changes := []*agentpreview.RecordChange{header}
	before := groupPlansFromRun(plan.Before)
	afterPlans := groupPlansFromRun(plan.After)
	for idx := range afterPlans {
		after := &afterPlans[idx]
		if idx < len(before) && before[idx].ShipmentCount == after.ShipmentCount &&
			before[idx].Total.Equal(after.Total) {
			continue
		}
		previous := decimal.NullDecimal{}
		if idx < len(before) {
			previous = changepreview.KnownAmount(before[idx].Total)
		}
		change, changeErr := proposedInvoiceChange(
			after,
			agentpreview.PreviewOperationUpdate,
			previous,
		)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, change)
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would change %s from %s to %s: %s billed, %s held back.",
		invoicepreview.RunLabel(plan.Before),
		plan.Before.TotalAmount.StringFixed(2),
		plan.After.TotalAmount.StringFixed(2),
		changepreview.CountOf(plan.After.ItemCount, "shipment"),
		changepreview.CountOf(plan.After.ExcludedCount, "shipment"),
	), changes...), nil
}

func outcomeSummary(
	plans []invoicerunports.InvoiceRunGroupPlan,
) (billed int, total decimal.Decimal, notes []string) {
	total = decimal.Zero
	notes = make([]string, 0, len(plans))
	for idx := range plans {
		plan := &plans[idx]
		switch plan.Outcome {
		case invoicerunports.InvoiceRunGroupBills:
			billed++
			total = total.Add(plan.Total)
		case invoicerunports.InvoiceRunGroupSkips, invoicerunports.InvoiceRunGroupAlreadySkipped:
			notes = append(notes, fmt.Sprintf("%s is skipped: %s", plan.GroupLabel, plan.Reason))
		case invoicerunports.InvoiceRunGroupAlreadyCommitted:
			notes = append(notes, plan.GroupLabel+" was already invoiced")
		}
	}

	return billed, total, notes
}

func groupChanges(
	plans []invoicerunports.InvoiceRunGroupPlan,
) ([]*agentpreview.RecordChange, error) {
	changes := make([]*agentpreview.RecordChange, 0, len(plans))
	for idx := range plans {
		plan := &plans[idx]
		operation := agentpreview.PreviewOperationCreate
		if plan.Outcome != invoicerunports.InvoiceRunGroupBills {
			operation = agentpreview.PreviewOperationRun
		}
		change, err := proposedInvoiceChange(plan, operation, decimal.NullDecimal{})
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	return changes, nil
}

func RenderCommit(
	_ *invoicerunports.CommitInvoiceRunRequest,
	plan *invoicerunports.InvoiceRunCommitPlan,
) (*agentpreview.ToolPreview, error) {
	if plan.AlreadyCommitted {
		return toolpreview.Build(
			invoicepreview.RunLabel(plan.Run) + " is already committed; committing it " +
				"again changes nothing.",
		), nil
	}

	after := *plan.Run
	after.Status = invoicerun.StatusCommitted
	header, err := toolpreview.Changed(invoicepreview.RunRecord(plan.Run), viewOfRun(plan.Run),
		viewOfRun(&after), toolpreview.Types(runDateTypes))
	if err != nil {
		return nil, err
	}
	groups, err := groupChanges(plan.Groups)
	if err != nil {
		return nil, err
	}

	billed, total, notes := outcomeSummary(plan.Groups)
	summary := fmt.Sprintf(
		"Would commit %s, making %s for %s as drafts.",
		invoicepreview.RunLabel(
			plan.Run,
		),
		changepreview.CountOf(billed, "invoice"),
		total.StringFixed(2),
	)

	return toolpreview.Build(changepreview.WithNotes(summary, notes),
		append([]*agentpreview.RecordChange{header}, groups...)...), nil
}

func RenderStatementBill(
	req *invoicerunports.BillStatementNowRequest,
	plan *invoicerunports.StatementBillPlan,
) (*agentpreview.ToolPreview, error) {
	header, err := toolpreview.Create(invoicepreview.RunRecord(plan.Run), viewOfRun(plan.Run),
		toolpreview.Types(runDateTypes))
	if err != nil {
		return nil, err
	}
	groups, err := groupChanges(plan.Groups)
	if err != nil {
		return nil, err
	}

	billed, total, notes := outcomeSummary(plan.Groups)
	summary := fmt.Sprintf("Would bill %s's open statement now, off cycle, making %s for %s "+
		"as drafts; %s stay on the statement.",
		plan.Statement.CustomerName, changepreview.CountOf(billed, "invoice"), total.StringFixed(2),
		changepreview.CountOf(len(req.Exclude)-len(plan.UnmatchedExclusions), "held-back shipment"))
	if len(plan.UnmatchedExclusions) > 0 {
		notes = append(notes, fmt.Sprintf("%s to hold back are no longer on the statement",
			changepreview.CountOf(len(plan.UnmatchedExclusions), "shipment")))
	}

	return toolpreview.Build(changepreview.WithNotes(summary, notes),
		append([]*agentpreview.RecordChange{header}, groups...)...), nil
}
