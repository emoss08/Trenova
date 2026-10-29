package bankreceipttools

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/referenceutils"
)

const matchCandidateLimit = 10

const postPaymentBoundText = "Runs on its own only when it records an unmatched bank receipt " +
	"as the receipt has it (its day, reference and memo, by ACH), whose reference or memo names " +
	"every invoice it pays, each an open invoice of the same customer paid to its exact " +
	"balance, the whole payment is applied with nothing short-paid or left unapplied, and the " +
	"amount is within the organization's money limits for agents; any other call waits for a " +
	"person. The receipt it records, if the run read it, does not hold it, because the receipt " +
	"is read again from the organization's records; any other outside content does."

const matchReceiptBoundText = "Runs on its own only when the payment is the one posted payment " +
	"with the receipt's exact amount and reference, and the amount is within the organization's " +
	"money limits for agents; any other call waits for a person. The receipt it matches, if the " +
	"run read it, does not hold it, because the receipt is read again from the organization's " +
	"records; any other outside content does."

func (t *PostCustomerPaymentTool) withinBounds(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) agenttoolports.MoneyBound {
	if t.payments == nil || t.receipts == nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	args, err := t.arguments(ctx, params)
	if err != nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	posted, err := t.payments.PreviewPostAndApply(ctx, args.request, params.Actor)
	if err != nil || posted == nil || posted.Payment == nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}

	if args.receipt == nil || !recordsTheReceipt(args.request, args.receipt) ||
		!remittanceMatches(args.receipt, posted) {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyExactMatch)
	}

	return agenttoolports.WithinMoneyBound(args.request.AmountMinor, posted.Payment.CurrencyCode).
		Rereading(agent.TaintSourceBankReceipt, receiptRef(args.receipt.ID))
}

// recordsTheReceipt says the payment is recorded as the receipt has it: on the
// day it was received, under its reference, with its memo or none, by the
// method a bank receipt arrives by. What the call says otherwise is not taken
// on its own.
func recordsTheReceipt(
	request *customerpaymentports.PostCustomerPaymentRequest,
	receipt *bankreceipt.BankReceipt,
) bool {
	memo := strings.TrimSpace(request.Memo)

	return sameDay(request.PaymentDate, receipt.ReceiptDate) &&
		sameDay(request.AccountingDate, receipt.ReceiptDate) &&
		referenceutils.Same(request.ReferenceNumber, receipt.ReferenceNumber) &&
		(memo == "" || memo == strings.TrimSpace(receipt.Memo)) &&
		request.PaymentMethod == defaultPaymentMethod
}

func sameDay(a, b int64) bool {
	return time.Unix(a, 0).UTC().Format(time.DateOnly) ==
		time.Unix(b, 0).UTC().Format(time.DateOnly)
}

func receiptRef(id pulid.ID) agentrecord.RecordRef {
	return agentrecord.RecordRef{EntityType: agent.TaintEntityBankReceipt, ID: id.String()}
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
	params *agenttoolports.ToolExecuteParams,
) agenttoolports.MoneyBound {
	if t.receipts == nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	request, err := t.request(params)
	if err != nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	match, err := t.receipts.PreviewMatch(ctx, request, params.Actor)
	if err != nil || match == nil || match.ReceiptBefore == nil || match.Payment == nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}

	receipt, payment := match.ReceiptBefore, match.Payment
	if !exactReceiptMatch(receipt, payment) {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyExactMatch)
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
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	if !onlyExactCandidate(receipt, payment, candidates) {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyExactMatch)
	}

	return agenttoolports.WithinMoneyBound(receipt.AmountMinor, payment.CurrencyCode).
		Rereading(agent.TaintSourceBankReceipt, receiptRef(receipt.ID))
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
