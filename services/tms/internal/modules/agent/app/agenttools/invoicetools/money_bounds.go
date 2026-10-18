package invoicetools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/referenceutils"
	"github.com/shopspring/decimal"
)

const matchCandidateLimit = 10

const postPaymentBoundText = "Runs on its own only when it records an unmatched bank receipt " +
	"whose reference or memo names every invoice it pays, each an open invoice of the same " +
	"customer paid to its exact balance, the whole payment is applied with nothing short-paid " +
	"or left unapplied, and the amount is within the organization's money limits for agents; " +
	"any other call waits for a person. Bank receipts the run read do not hold it, because " +
	"the receipt is read again from the organization's records; any other outside content does."

const matchReceiptBoundText = "Runs on its own only when the payment is the one posted payment " +
	"with the receipt's exact amount and reference, and the amount is within the organization's " +
	"money limits for agents; any other call waits for a person. Bank receipts the run read do " +
	"not hold it, because the receipt is read again from the organization's records; any other " +
	"outside content does."

const correctChargeBoundText = "Runs on its own only while nothing has been invoiced from the " +
	"billing queue item and every charge it adds, removes or changes, counted in full, is " +
	"within the organization's money limits for agents; any other call waits for a person."

func (t *PostCustomerPaymentTool) withinBounds(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the TierCondition bounds pass params by value
) agentports.MoneyBound {
	if t.payments == nil || t.receipts == nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	args, err := t.arguments(ctx, params)
	if err != nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	posted, err := t.payments.PreviewPostAndApply(ctx, args.request, params.Actor)
	if err != nil || posted == nil || posted.Payment == nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}

	if args.receipt == nil || !remittanceMatches(args.receipt, posted) {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyExactMatch)
	}

	return agentports.WithinMoneyBound(args.request.AmountMinor, posted.Payment.CurrencyCode).
		Rereading(agent.TaintSourceBankReceipt)
}

func remittanceMatches(
	receipt *bankreceipt.BankReceipt,
	posted *customerpaymentports.CustomerPaymentPostPreview,
) bool {
	payment := posted.Payment
	if len(payment.Applications) == 0 ||
		len(payment.Applications) != len(posted.InvoicesBefore) ||
		payment.AmountMinor != receipt.AmountMinor ||
		payment.UnappliedAmountMinor != 0 {
		return false
	}

	var applied int64
	for idx, application := range payment.Applications {
		inv := posted.InvoicesBefore[idx]
		if application == nil || inv == nil || application.InvoiceID != inv.ID ||
			application.ShortPayAmountMinor != 0 ||
			inv.CustomerID != payment.CustomerID ||
			!inv.IsOpen() ||
			application.AppliedAmountMinor != inv.OpenBalanceMinor() ||
			!referenceutils.Mentions(inv.Number, receipt.ReferenceNumber, receipt.Memo) {
			return false
		}
		applied += application.AppliedAmountMinor
	}

	return applied == payment.AmountMinor
}

func (t *MatchBankReceiptTool) withinBounds(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the TierCondition bounds pass params by value
) agentports.MoneyBound {
	if t.receipts == nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	request, err := t.request(params)
	if err != nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	match, err := t.receipts.PreviewMatch(ctx, request, params.Actor)
	if err != nil || match == nil || match.ReceiptBefore == nil || match.Payment == nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}

	receipt, payment := match.ReceiptBefore, match.Payment
	if !exactReceiptMatch(receipt, payment) {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyExactMatch)
	}

	candidates, err := t.receipts.PaymentsOfReceiptAmount(
		ctx,
		&bankreceiptports.ReceiptAmountPaymentsRequest{
			ReceiptID:  request.ReceiptID,
			Limit:      matchCandidateLimit + 1,
			TenantInfo: request.TenantInfo,
		},
	)
	if err != nil || len(candidates) > matchCandidateLimit {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	if !onlyExactCandidate(receipt, payment, candidates) {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyExactMatch)
	}

	return agentports.WithinMoneyBound(receipt.AmountMinor, payment.CurrencyCode).
		Rereading(agent.TaintSourceBankReceipt)
}

func exactReceiptMatch(receipt *bankreceipt.BankReceipt, payment *customerpayment.Payment) bool {
	return payment.Status == customerpayment.StatusPosted &&
		receipt.AmountMinor == payment.AmountMinor &&
		referenceutils.Same(receipt.ReferenceNumber, payment.ReferenceNumber)
}

func onlyExactCandidate(
	receipt *bankreceipt.BankReceipt,
	payment *customerpayment.Payment,
	candidates []*customerpayment.Payment,
) bool {
	exact := 0
	for _, candidate := range candidates {
		if candidate == nil || candidate.AmountMinor != receipt.AmountMinor ||
			!referenceutils.Same(candidate.ReferenceNumber, receipt.ReferenceNumber) {
			continue
		}
		if candidate.ID != payment.ID {
			return false
		}
		exact++
	}

	return exact == 1
}

func (t *CorrectChargeCodeTool) withinBounds(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the TierCondition bounds pass params by value
) agentports.MoneyBound {
	if t.billing == nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	request, err := t.request(&params)
	if err != nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	plan, err := t.billing.PreviewUpdateCharges(ctx, request, params.Actor)
	if err != nil || plan == nil || plan.Item == nil || plan.Before == nil || plan.After == nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	if !uninvoiced(plan.Item) {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyRecordState)
	}
	if plan.Item.BillToCustomer == nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}

	return agentports.WithinMoneyBound(
		money.MinorUnits(grossChargeChange(plan.Before, plan.After)),
		plan.Item.BillToCustomer.BillingCurrencyCode(),
	)
}

func grossChargeChange(before, after *shipment.Shipment) decimal.Decimal {
	beforeFreight := before.FreightChargeAmount.Decimal
	afterFreight := after.FreightChargeAmount.Decimal
	gross := afterFreight.Sub(beforeFreight).Abs()

	kept := make(map[pulid.ID]*shipment.AdditionalCharge, len(before.AdditionalCharges))
	for _, charge := range before.AdditionalCharges {
		if charge != nil && charge.ID.IsNotNil() {
			kept[charge.ID] = charge
		}
	}
	for _, charge := range after.AdditionalCharges {
		if charge == nil {
			continue
		}
		total := charge.Total(afterFreight)
		previous, found := kept[charge.ID]
		if !found || charge.ID.IsNil() {
			gross = gross.Add(total.Abs())
			continue
		}
		delete(kept, charge.ID)
		gross = gross.Add(total.Sub(previous.Total(beforeFreight)).Abs())
	}
	for _, charge := range before.AdditionalCharges {
		if charge == nil {
			continue
		}
		if _, removed := kept[charge.ID]; removed || charge.ID.IsNil() {
			gross = gross.Add(charge.Total(beforeFreight).Abs())
		}
	}

	return gross
}

func uninvoiced(item *billingqueue.BillingQueueItem) bool {
	return item.Status == billingqueue.StatusInReview &&
		item.InvoiceID.IsNil() &&
		(item.SourceInvoiceID == nil || item.SourceInvoiceID.IsNil()) &&
		(item.SourceInvoiceAdjustmentID == nil || item.SourceInvoiceAdjustmentID.IsNil()) &&
		(item.SourceCreditMemoInvoiceID == nil || item.SourceCreditMemoInvoiceID.IsNil())
}
