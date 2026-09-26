package agenttoolservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
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

var (
	runDateTypes = map[string]assistantartifact.DisplayType{
		paramPeriodStart: assistantartifact.DisplayDate,
		paramPeriodEnd:   assistantartifact.DisplayDate,
		paramInvoiceDate: assistantartifact.DisplayDate,
	}
	proposedInvoiceRefs = map[string]permission.Resource{
		paramCustomerID: permission.ResourceCustomer,
	}
)

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

func runLabel(run *invoicerun.InvoiceRun) string {
	if run.ID.IsNil() || run.Number == "" {
		return "New invoice run"
	}

	return "Invoice run " + run.Number
}

func runRecord(run *invoicerun.InvoiceRun) toolpreview.Record {
	record := toolpreview.Record{Resource: permission.ResourceInvoiceRun, Label: runLabel(run)}
	if run.ID.IsNotNil() {
		record.ID = run.ID
		record.Version = pinnedVersion(run.Version)
	}

	return record
}

func runTotalMoney(change *agent.RecordChange, before, after *invoicerun.InvoiceRun) {
	line := agent.MoneyLine{Label: moneyLineRunTotal, After: knownAmount(after.TotalAmount)}
	if before != nil {
		line.Before = knownAmount(before.TotalAmount)
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(after.CurrencyCode, line),
		toolpreview.SensitiveAs("totalAmount"))
}

func proposedInvoiceChange(
	plan serviceports.InvoiceRunGroupPlan,
	operation agent.PreviewOperation,
	before decimal.NullDecimal,
) (*agent.RecordChange, error) {
	view := &proposedInvoiceView{
		CustomerID:    plan.CustomerID,
		GroupLabel:    plan.GroupLabel,
		ShipmentCount: plan.ShipmentCount,
		Outcome:       string(plan.Outcome),
		Reason:        plan.Reason,
	}
	resource := permission.ResourceInvoice
	label := "New invoice: " + plan.GroupLabel
	if plan.Outcome != serviceports.InvoiceRunGroupBills {
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
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(plan.CurrencyCode, agent.MoneyLine{
		Label:  moneyLineRunTotal,
		Before: before,
		After:  knownAmount(plan.Total),
	}), toolpreview.SensitiveAs("totalAmount"))

	return change, nil
}

func groupPlansFromRun(run *invoicerun.InvoiceRun) []serviceports.InvoiceRunGroupPlan {
	plans := make([]serviceports.InvoiceRunGroupPlan, 0, len(run.Groups))
	for _, group := range run.Groups {
		if group == nil {
			continue
		}
		plans = append(plans, serviceports.InvoiceRunGroupPlan{
			GroupID:       group.ID,
			GroupLabel:    group.GroupLabel,
			CustomerID:    group.CustomerID,
			ShipmentCount: group.ItemCount,
			Total:         group.TotalAmount,
			CurrencyCode:  group.CurrencyCode,
			Outcome:       serviceports.InvoiceRunGroupBills,
		})
	}

	return plans
}

func renderBuiltRun(
	_ *serviceports.PreviewInvoiceRunRequest,
	run *invoicerun.InvoiceRun,
) (*agent.ToolPreview, error) {
	header, err := toolpreview.Create(runRecord(run), viewOfRun(run),
		toolpreview.Types(runDateTypes))
	if err != nil {
		return nil, err
	}
	runTotalMoney(header, nil, run)

	changes := []*agent.RecordChange{header}
	for _, plan := range groupPlansFromRun(run) {
		change, changeErr := proposedInvoiceChange(
			plan,
			agent.PreviewOperationCreate,
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
		countOf(run.GroupCount, "invoice"),
		countOf(run.ItemCount, "shipment"),
		run.TotalAmount.StringFixed(2),
	), changes...), nil
}

func renderMembership(
	_ *serviceports.AdjustInvoiceRunMembershipRequest,
	plan *serviceports.InvoiceRunChangePreview,
) (*agent.ToolPreview, error) {
	header, err := toolpreview.Changed(runRecord(plan.Before), viewOfRun(plan.Before),
		viewOfRun(plan.After), toolpreview.Types(runDateTypes))
	if err != nil {
		return nil, err
	}
	runTotalMoney(header, plan.Before, plan.After)

	changes := []*agent.RecordChange{header}
	before := groupPlansFromRun(plan.Before)
	for idx, after := range groupPlansFromRun(plan.After) {
		if idx < len(before) && before[idx].ShipmentCount == after.ShipmentCount &&
			before[idx].Total.Equal(after.Total) {
			continue
		}
		previous := decimal.NullDecimal{}
		if idx < len(before) {
			previous = knownAmount(before[idx].Total)
		}
		change, changeErr := proposedInvoiceChange(after, agent.PreviewOperationUpdate, previous)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, change)
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would change %s from %s to %s: %s billed, %s held back.",
		runLabel(plan.Before),
		plan.Before.TotalAmount.StringFixed(2),
		plan.After.TotalAmount.StringFixed(2),
		countOf(plan.After.ItemCount, "shipment"),
		countOf(plan.After.ExcludedCount, "shipment"),
	), changes...), nil
}

func outcomeSummary(plans []serviceports.InvoiceRunGroupPlan) (int, decimal.Decimal, []string) {
	billed := 0
	total := decimal.Zero
	notes := make([]string, 0, len(plans))
	for _, plan := range plans {
		switch plan.Outcome {
		case serviceports.InvoiceRunGroupBills:
			billed++
			total = total.Add(plan.Total)
		case serviceports.InvoiceRunGroupSkips, serviceports.InvoiceRunGroupAlreadySkipped:
			notes = append(notes, fmt.Sprintf("%s is skipped: %s", plan.GroupLabel, plan.Reason))
		case serviceports.InvoiceRunGroupAlreadyCommitted:
			notes = append(notes, plan.GroupLabel+" was already invoiced")
		}
	}

	return billed, total, notes
}

func groupChanges(plans []serviceports.InvoiceRunGroupPlan) ([]*agent.RecordChange, error) {
	changes := make([]*agent.RecordChange, 0, len(plans))
	for _, plan := range plans {
		operation := agent.PreviewOperationCreate
		if plan.Outcome != serviceports.InvoiceRunGroupBills {
			operation = agent.PreviewOperationRun
		}
		change, err := proposedInvoiceChange(plan, operation, decimal.NullDecimal{})
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	return changes, nil
}

func withNotes(summary string, notes []string) string {
	if len(notes) == 0 {
		return summary
	}

	return summary + " " + strings.Join(notes, "; ") + "."
}

func renderCommit(
	_ *serviceports.CommitInvoiceRunRequest,
	plan *serviceports.InvoiceRunCommitPlan,
) (*agent.ToolPreview, error) {
	if plan.AlreadyCommitted {
		return toolpreview.Build(runLabel(plan.Run) + " is already committed; committing it " +
			"again changes nothing."), nil
	}

	after := *plan.Run
	after.Status = invoicerun.StatusCommitted
	header, err := toolpreview.Changed(runRecord(plan.Run), viewOfRun(plan.Run),
		viewOfRun(&after), toolpreview.Types(runDateTypes))
	if err != nil {
		return nil, err
	}
	groups, err := groupChanges(plan.Groups)
	if err != nil {
		return nil, err
	}

	billed, total, notes := outcomeSummary(plan.Groups)
	summary := fmt.Sprintf("Would commit %s, making %s for %s as drafts.",
		runLabel(plan.Run), countOf(billed, "invoice"), total.StringFixed(2))

	return toolpreview.Build(withNotes(summary, notes),
		append([]*agent.RecordChange{header}, groups...)...), nil
}

func renderCancel(
	_ *serviceports.CancelInvoiceRunRequest,
	plan *serviceports.InvoiceRunChangePreview,
) (*agent.ToolPreview, error) {
	change, err := toolpreview.Changed(runRecord(plan.Before), plan.Before, plan.After,
		toolpreview.Only(fieldStatus, "failureReason", "canceledById", "canceledAt"),
		toolpreview.WithRefs(map[string]permission.Resource{"canceledById": permission.ResourceUser}),
		toolpreview.Volatile("canceledAt"),
	)
	if err != nil {
		return nil, err
	}
	change.Operation = agent.PreviewOperationArchive

	return toolpreview.Build(fmt.Sprintf(
		"Would cancel %s (now %s); its %s stay approved for the next run.",
		runLabel(plan.Before), plan.Before.Status, countOf(plan.Before.ItemCount, "shipment"),
	), change), nil
}

func renderStatementBill(
	req *serviceports.BillStatementNowRequest,
	plan *serviceports.StatementBillPlan,
) (*agent.ToolPreview, error) {
	header, err := toolpreview.Create(runRecord(plan.Run), viewOfRun(plan.Run),
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
		plan.Statement.CustomerName, countOf(billed, "invoice"), total.StringFixed(2),
		countOf(len(req.Exclude)-len(plan.UnmatchedExclusions), "held-back shipment"))
	if len(plan.UnmatchedExclusions) > 0 {
		notes = append(notes, fmt.Sprintf("%s to hold back are no longer on the statement",
			countOf(len(plan.UnmatchedExclusions), "shipment")))
	}

	return toolpreview.Build(withNotes(summary, notes),
		append([]*agent.RecordChange{header}, groups...)...), nil
}
