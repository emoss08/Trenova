package settlementreleaseports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/shared/i18n"
)

func BlockWords(ctx context.Context, block driversettlement.ReleaseBlock) string {
	switch block {
	case driversettlement.ReleaseBlockHeld:
		return i18n.T(ctx, "On hold")
	case driversettlement.ReleaseBlockCriticalException:
		return i18n.T(ctx, "Has a critical exception")
	case driversettlement.ReleaseBlockDisputeOpen:
		return i18n.T(ctx, "Dispute open")
	}

	return ""
}

func DisputeAbout(ctx context.Context, category driversettlement.DisputeCategory) string {
	switch category {
	case driversettlement.DisputeCategoryMissingPay:
		return i18n.T(ctx, "about missing pay")
	case driversettlement.DisputeCategoryIncorrectRate:
		return i18n.T(ctx, "about the rate")
	case driversettlement.DisputeCategoryIncorrectDeduction:
		return i18n.T(ctx, "about a deduction")
	case driversettlement.DisputeCategoryMissingReimbursement:
		return i18n.T(ctx, "about a missing reimbursement")
	case driversettlement.DisputeCategoryOther:
	}

	return ""
}

func HoldReasonWords(ctx context.Context, reason driversettlement.HoldReason) string {
	switch reason {
	case driversettlement.HoldReasonRateDispute:
		return i18n.T(ctx, "Rate dispute")
	case driversettlement.HoldReasonMissingPaperwork:
		return i18n.T(ctx, "Missing paperwork")
	case driversettlement.HoldReasonNegativeNet:
		return i18n.T(ctx, "Negative net")
	case driversettlement.HoldReasonDeductionReview:
		return i18n.T(ctx, "Deduction under review")
	case driversettlement.HoldReasonOther:
		return i18n.T(ctx, "Other")
	}

	return ""
}

func HoldWords(ctx context.Context, reason driversettlement.HoldReason, note string) string {
	label := HoldReasonWords(ctx, reason)
	if label == "" {
		return i18n.T(ctx, "On hold")
	}
	if note == "" {
		return i18n.T(ctx, "On hold: {0}", label)
	}

	return i18n.T(ctx, "On hold: {0}, {1}", label, note)
}
