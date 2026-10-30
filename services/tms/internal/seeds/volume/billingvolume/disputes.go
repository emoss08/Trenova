package billingvolume

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute/invoicedisputedb"
	"github.com/emoss08/trenova/internal/seeds/volume/ledgervolume"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/shared/money"
)

const (
	bilDisputeAfterDays  = 15
	bilOpenDisputeShare  = 0.25
	bilClosedDisputeRate = 0.015
)

var bilDisputeReasons = []struct {
	code invoicedispute.DisputeReasonCode
	note string
}{
	{invoicedispute.DisputeReasonRateDiscrepancy, "Linehaul billed above the contract rate"},
	{invoicedispute.DisputeReasonAccessorialDisputed, "Detention not approved by the consignee"},
	{invoicedispute.DisputeReasonMissingDocumentation, "Customer asked for the signed POD"},
	{invoicedispute.DisputeReasonServiceFailure, "Delivery missed the appointment window"},
	{invoicedispute.DisputeReasonWrongBillTo, "Invoice belongs to the shipper's account"},
	{invoicedispute.DisputeReasonDuplicateBilling, "Customer believes this load was billed twice"},
}

var bilDisputeShares = []int{30, 25, 20, 12, 8, 5}

func (g *billingPhase) bilDispute(
	inv *invoice.Invoice,
	plan *ledgervolume.PaymentPlan,
) *invoicedispute.InvoiceDispute {
	open := inv.OpenBalanceMinor()
	if open <= 0 || inv.IsCreditMemo() || inv.FactoringCompanyID != nil {
		return nil
	}
	nowUnix := volumegen.Unix(g.Now)
	due := inv.InvoiceDate
	if inv.DueDate != nil {
		due = *inv.DueDate
	}

	if plan != nil && plan.Amount >= open {
		return g.bilClosedDispute(inv, plan, open)
	}
	if (nowUnix-due)/volumegen.Day < bilDisputeAfterDays || !g.Fake.Chance(bilOpenDisputeShare) {
		return nil
	}

	unpaid := open
	if plan != nil {
		unpaid = open - plan.Amount
	}
	reason := bilDisputeReasons[g.Fake.Weighted(bilDisputeShares)]
	amount := unpaid
	switch {
	case reason.code == invoicedispute.DisputeReasonAccessorialDisputed &&
		inv.OtherAmountMinor > 0:
		amount = min(unpaid, inv.OtherAmountMinor)
	case reason.code == invoicedispute.DisputeReasonRateDiscrepancy:
		amount = max(unpaid*int64(g.Fake.Between(10, 40))/100, 1)
	}
	openedAt := g.bilClamp(
		inv.InvoiceDate+g.Fake.Between64(5*volumegen.Day, 45*volumegen.Day),
		inv.InvoiceDate+volumegen.Day,
	)
	if plan != nil {
		openedAt = max(openedAt, plan.Date+volumegen.Hour)
		if openedAt >= nowUnix {
			return nil
		}
	}

	inv.DisputeStatus = invoice.DisputeStatusDisputed
	inv.UpdatedAt = max(inv.UpdatedAt, openedAt)

	return g.bilNewDispute(inv, reason.code, reason.note, amount, openedAt)
}

func (g *billingPhase) bilClosedDispute(
	inv *invoice.Invoice,
	plan *ledgervolume.PaymentPlan,
	amount int64,
) *invoicedispute.InvoiceDispute {
	if plan.Date-inv.InvoiceDate < 5*volumegen.Day || !g.Fake.Chance(bilClosedDisputeRate) {
		return nil
	}
	reason := bilDisputeReasons[g.Fake.Weighted(bilDisputeShares)]
	openedAt := inv.InvoiceDate + g.Fake.Between64(volumegen.Day, plan.Date-inv.InvoiceDate-
		3*volumegen.Day)
	resolvedAt := openedAt + g.Fake.Between64(volumegen.Hour, plan.Date-openedAt-volumegen.Hour)

	dispute := g.bilNewDispute(inv, reason.code, reason.note, amount, openedAt)
	dispute.ResolvedByID = g.PickUser(g.Refs.UserIDs)
	dispute.ResolvedAt = volumegen.Ptr(resolvedAt)
	dispute.UpdatedAt = resolvedAt
	if g.Fake.Chance(0.3) {
		dispute.Status = invoicedispute.DisputeCaseStatusWithdrawn
		dispute.ResolutionNotes = "Customer withdrew the dispute after reviewing the paperwork"
		return dispute
	}
	dispute.Status = invoicedispute.DisputeCaseStatusResolved
	dispute.Resolution = invoicedispute.DisputeResolutionInvoiceUpheld
	dispute.ResolutionNotes = "Sent the rate confirmation and signed POD; customer agreed to pay"

	return dispute
}

func (g *billingPhase) bilNewDispute(
	inv *invoice.Invoice,
	reason invoicedispute.DisputeReasonCode,
	note string,
	amount int64,
	openedAt int64,
) *invoicedispute.InvoiceDispute {
	return &invoicedispute.InvoiceDispute{
		ID:                  volumegen.IDAt("idsp_", openedAt),
		OrganizationID:      inv.OrganizationID,
		BusinessUnitID:      inv.BusinessUnitID,
		InvoiceID:           inv.ID,
		CustomerID:          inv.CustomerID,
		Status:              invoicedispute.DisputeCaseStatusOpen,
		ReasonCode:          reason,
		DisputedAmount:      money.DecimalFromMinor(amount),
		DisputedAmountMinor: amount,
		Notes:               note,
		OpenedByID:          g.PickUser(g.Refs.UserIDs),
		OpenedAt:            openedAt,
		CreatedAt:           openedAt,
		UpdatedAt:           openedAt,
	}
}

func (g *billingPhase) bilWriteDisputes(
	ctx context.Context,
	tx *keel.DB,
	disputes []*invoicedispute.InvoiceDispute,
) error {
	return ledgervolume.Insert(ctx, tx, invoicedisputedb.InvoiceDisputes.Handle, disputes,
		func(r *invoicedispute.InvoiceDispute) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false)
}
