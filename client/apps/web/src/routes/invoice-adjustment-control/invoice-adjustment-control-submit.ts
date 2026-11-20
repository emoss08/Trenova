import {
  ADJUSTMENT_AMOUNT_SCALE,
  adjustmentAccountingDatePolicySchema,
  approvalPolicySchema,
  closedPeriodAdjustmentPolicySchema,
  customerCreditBalancePolicySchema,
  isDecimalWithin,
  overCreditPolicySchema,
  replacementInvoiceReviewPolicySchema,
  RERATE_TOLERANCE_MAX,
  RERATE_TOLERANCE_SCALE,
  writeOffApprovalPolicySchema,
  type AdjustmentAccountingDatePolicy,
  type ApprovalPolicy,
  type ClosedPeriodAdjustmentPolicy,
  type InvoiceAdjustmentControl,
} from "@/types/invoice-adjustment-control";
import { decimalPattern } from "@trenova/shared/types/decimal";

/**
 * Whether the closed-period policy is ever consulted: only while an adjustment books on its
 * invoice's date, since one that always books in the next open period never meets a closed
 * one.
 */
export function closedPeriodPolicyApplies(datePolicy: AdjustmentAccountingDatePolicy): boolean {
  return datePolicy === adjustmentAccountingDatePolicySchema.enum.UseOriginalIfOpenElseNextOpen;
}

/**
 * Whether a closed-period adjustment booked in the next open period asks for an approval
 * that the approval policy never gives. The server refuses that pair.
 */
export function nextPeriodNeedsApproval({
  datePolicy,
  closedPeriodPolicy,
}: {
  datePolicy: AdjustmentAccountingDatePolicy;
  closedPeriodPolicy: ClosedPeriodAdjustmentPolicy;
}): boolean {
  return (
    closedPeriodPolicyApplies(datePolicy) &&
    closedPeriodPolicy === closedPeriodAdjustmentPolicySchema.enum.PostInNextOpenPeriodWithApproval
  );
}

/** Whether no adjustment ever waits for approval. */
export function approvesNothing(policy: ApprovalPolicy): boolean {
  return policy === approvalPolicySchema.enum.None;
}

/**
 * The record as it is saved: a setting the page hides because another one makes it
 * meaningless is sent the way the server requires it then, so a save is never refused for a
 * field the person cannot see.
 */
export function prepareInvoiceAdjustmentControl(
  values: InvoiceAdjustmentControl,
): InvoiceAdjustmentControl {
  const next = { ...values };

  // A record that books every adjustment in the next open period never reads the
  // closed-period policy, but the server still refuses reopening with that date policy, and
  // next-period posting with approval while nothing needs approval.
  if (!closedPeriodPolicyApplies(next.adjustmentAccountingDatePolicy)) {
    const refused =
      next.closedPeriodAdjustmentPolicy === closedPeriodAdjustmentPolicySchema.enum.RequireReopen ||
      (next.closedPeriodAdjustmentPolicy ===
        closedPeriodAdjustmentPolicySchema.enum.PostInNextOpenPeriodWithApproval &&
        approvesNothing(next.standardAdjustmentApprovalPolicy));
    if (refused) {
      next.closedPeriodAdjustmentPolicy = closedPeriodAdjustmentPolicySchema.enum.Disallow;
    }
  }

  if (next.customerCreditBalancePolicy === customerCreditBalancePolicySchema.enum.Disallow) {
    next.overCreditPolicy = overCreditPolicySchema.enum.Block;
  }

  const amount = decimalPattern(ADJUSTMENT_AMOUNT_SCALE);
  if (
    next.standardAdjustmentApprovalPolicy !== approvalPolicySchema.enum.AmountThreshold &&
    !amount.test(next.standardAdjustmentApprovalThreshold.trim())
  ) {
    next.standardAdjustmentApprovalThreshold = "0";
  }
  if (
    next.writeOffApprovalPolicy !==
      writeOffApprovalPolicySchema.enum.RequireApprovalAboveThreshold &&
    !amount.test(next.writeOffApprovalThreshold.trim())
  ) {
    next.writeOffApprovalThreshold = "0";
  }

  if (
    next.replacementInvoiceReviewPolicy !==
      replacementInvoiceReviewPolicySchema.enum.RequireReviewWhenEconomicTermsChange &&
    !isDecimalWithin(
      next.rerateVarianceTolerancePercent,
      RERATE_TOLERANCE_SCALE,
      RERATE_TOLERANCE_MAX,
    )
  ) {
    next.rerateVarianceTolerancePercent = "0";
  }

  return next;
}
