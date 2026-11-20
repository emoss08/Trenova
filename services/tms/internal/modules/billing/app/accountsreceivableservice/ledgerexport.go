package accountsreceivableservice

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel/analytics/formula"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	LedgerReportCategory    = "Customer ledgers"
	LedgerReportDescription = "A customer's ledger exported from the customer ledger page. Each " +
		"balance starts from what the customer owed before the first day, as it stood at the export."

	ledgerReportEntity = "customer_ledger_entry"
	ledgerLineOffset   = 1_000_000
)

var (
	ledgerDateDisplay  = json.RawMessage(`{"style":"date","dateStyle":"date"}`)
	ledgerMoneyDisplay = json.RawMessage(
		`{"style":"currency","decimals":2,"negative":"parentheses"}`,
	)
	ledgerMinorFactor = 0.01
)

type ledgerReportWindow struct {
	req          customerledgerports.ARLedgerPageRequest
	openingMinor int64
}

func (s *Service) CustomerLedgerReport(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req customerledgerports.ARLedgerPageRequest,
) (*ir.Definition, error) {
	req.First, req.After = 0, ""
	if err := checkLedgerWindow(req); err != nil {
		return nil, err
	}
	if req.EndDate == 0 {
		req.EndDate = timeutils.NowUnix()
		if err := checkLedgerWindow(req); err != nil {
			return nil, err
		}
	}

	window := ledgerReportWindow{req: req}
	if req.StartDate > 0 {
		opening, err := s.repo.GetCustomerLedgerBalance(
			ctx,
			customerledgerports.GetCustomerLedgerBalanceRequest{
				TenantInfo: tenantInfo,
				CustomerID: req.CustomerID,
				Before:     req.StartDate,
			},
		)
		if err != nil {
			return nil, err
		}
		window.openingMinor = opening
	}

	return ledgerReport(&window), nil
}

func ledgerReport(window *ledgerReportWindow) *ir.Definition {
	filters := []ir.Filter{
		{Field: "customer.id", Op: ir.OpEq, Value: window.req.CustomerID.String()},
		{Field: "transactionDate", Op: ir.OpLte, Value: window.req.EndDate},
	}
	if window.req.StartDate > 0 {
		filters = append(filters, ir.Filter{
			Field: "transactionDate", Op: ir.OpGte, Value: window.req.StartDate,
		})
	}

	return &ir.Definition{
		Version: ir.Version,
		Entity:  ledgerReportEntity,
		Columns: []ir.Column{
			{
				ID:        "transaction_date",
				Label:     "Date",
				Dimension: &ir.Dimension{Field: "transactionDate"},
				Display:   ledgerDateDisplay,
			},
			{
				ID:        "document_number",
				Label:     "Document",
				Dimension: &ir.Dimension{Field: "documentNumber"},
			},
			{ID: "event", Label: "Event", Calc: formula.Of(eventLabelFormula())},
			ledgerMoneyColumn("debit", "Debit", "nullif(greatest({amountMinor}, 0), 0)"),
			ledgerMoneyColumn("credit", "Credit", "nullif(greatest(0 - {amountMinor}, 0), 0)"),
			{ID: "amount_minor", Hidden: true, Calc: formula.Of("{amountMinor}")},
			{
				ID:     "position",
				Hidden: true,
				Calc: formula.Of(fmt.Sprintf(
					"concat({transactionDate}, {lineNumber} + %d, {id})",
					ledgerLineOffset,
				)),
			},
			{
				ID:     "running_minor",
				Hidden: true,
				Window: &ir.Window{Of: "amount_minor", Fn: ir.WindowRunningTotal, Over: "position"},
			},
			ledgerMoneyColumn("balance", "Balance", openingFormula(window.openingMinor)),
		},
		Filters: ir.And(filters...),
		Sort:    []ir.Sort{{Column: "position"}},
	}
}

func ledgerMoneyColumn(id, label, calc string) ir.Column {
	return ir.Column{
		ID:        id,
		Label:     label,
		Calc:      formula.Of(calc),
		Transform: &ir.Transform{Op: ir.TransformScale, Factor: &ledgerMinorFactor},
		Display:   ledgerMoneyDisplay,
	}
}

func openingFormula(opening int64) string {
	switch {
	case opening > 0:
		return fmt.Sprintf("[running_minor] + %d", opening)
	case opening < 0:
		return fmt.Sprintf("[running_minor] - %d", -opening)
	}
	return "[running_minor]"
}

func eventLabelFormula() string {
	labels := customerledger.EventLabels()
	events := make([]string, 0, len(labels))
	for event, label := range labels {
		if strings.ContainsRune(event, '\'') || strings.ContainsRune(label, '\'') {
			continue
		}
		events = append(events, event)
	}
	slices.Sort(events)

	var b strings.Builder
	b.WriteString("case {sourceEventType}")
	for _, event := range events {
		fmt.Fprintf(&b, " when '%s' then '%s'", event, labels[event])
	}
	b.WriteString(" else {sourceEventType} end")
	return b.String()
}
