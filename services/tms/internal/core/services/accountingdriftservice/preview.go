package accountingdriftservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/money"
)

func describeFinding(finding *accountingsync.AccountingDriftFinding) string {
	label := finding.ObjectNumber
	if label == "" {
		label = finding.PartyName
	}
	switch finding.Kind {
	case accountingsync.DriftAmountMismatch:
		return "the difference in the total of " + label
	case accountingsync.DriftStatusMismatch:
		return "the difference in the status of " + label
	case accountingsync.DriftDeletedInProvider:
		return "the deletion of " + label
	case accountingsync.DriftVoidedInProvider:
		return "the void of " + label
	case accountingsync.DriftCustomerBalanceMismatch:
		return "the difference in " + label + "'s balance"
	default:
		return "the difference in " + label
	}
}

func valueOf(minor *int64, currency string) string {
	if minor == nil {
		return "nothing"
	}
	return money.FormatMinor(*minor, currency)
}

func (s *Service) preview(
	ctx context.Context,
	plan *fixPlan,
) (*services.AccountingDriftFixPreview, error) {
	tolerance, err := s.toleranceMinor(ctx, plan.finding.TenantInfo())
	if err != nil {
		return nil, err
	}
	return &services.AccountingDriftFixPreview{
		Finding:         plan.finding,
		Direction:       plan.direction,
		FixObject:       plan.fixObject,
		Operation:       plan.operation,
		AmountMinor:     plan.amountMinor,
		CurrencyCode:    plan.finding.CurrencyCode,
		ToleranceMinor:  tolerance,
		WithinTolerance: plan.finding.WithinTolerance(tolerance),
		Summary:         planSummary(plan),
	}, nil
}

func planSummary(plan *fixPlan) string {
	finding := plan.finding
	provider := accountingsync.ProviderName(plan.conn.IntegrationType)
	currency := finding.CurrencyCode
	switch plan.fixObject {
	case accountingsync.DriftFixSyncRecord:
		return pushSummary(plan, provider)
	case accountingsync.DriftFixCreditMemo:
		return "Post a credit memo of " + money.FormatMinor(plan.amountMinor, currency) +
			" against " + finding.ObjectNumber + ", bringing it to " +
			valueOf(finding.ProviderMinor, currency) + " as in " + provider +
			". The memo is not sent to " + provider + "."
	case accountingsync.DriftFixDebitMemo:
		return "Post a debit memo of " + money.FormatMinor(plan.amountMinor, currency) +
			" against " + finding.ObjectNumber + ", bringing it to " +
			valueOf(finding.ProviderMinor, currency) + " as in " + provider +
			". The memo is not sent to " + provider + "."
	case accountingsync.DriftFixInvoiceVoid:
		return "Void " + finding.ObjectNumber + " in Trenova without rebilling, as it was " +
			goneWord(finding.Kind) + " in " + provider + ". The reversal is not sent to " +
			provider + "."
	case accountingsync.DriftFixPaymentReversal:
		return "Reverse payment " + finding.ObjectNumber + " of " +
			money.FormatMinor(plan.amountMinor, currency) + " in Trenova, as it was " +
			goneWord(finding.Kind) + " in " + provider + ". The reversal is not sent to " +
			provider + "."
	default:
		return ""
	}
}

func pushSummary(plan *fixPlan, provider string) string {
	finding := plan.finding
	switch plan.operation {
	case accountingsync.SyncOperationRecreate:
		return "Create " + finding.ObjectNumber + " in " + provider +
			" again from Trenova, as a new document."
	case accountingsync.SyncOperationVoid:
		return "Void " + finding.ObjectNumber + " in " + provider + ", as it is in Trenova."
	case accountingsync.SyncOperationCreate, accountingsync.SyncOperationUpdate:
		return "Send Trenova's " + finding.ObjectNumber + " to " + provider + ": " +
			valueOf(finding.TrenovaMinor, finding.CurrencyCode) + " replaces " +
			valueOf(finding.ProviderMinor, finding.CurrencyCode) + "."
	default:
		return ""
	}
}
