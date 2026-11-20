import { translate } from "@trenova/shared/i18n/runtime";
import { compareDecimalStrings, decimalPattern } from "@trenova/shared/types/decimal";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "@trenova/shared/types/helpers";
import { z } from "zod";

/** The server keeps the approval amounts as NUMERIC(19, 4). */
export const ADJUSTMENT_AMOUNT_SCALE = 4;
/** The server keeps the rerate tolerance as NUMERIC(9, 6). */
export const RERATE_TOLERANCE_SCALE = 6;
/** The largest rerate tolerance NUMERIC(9, 6) holds. */
export const RERATE_TOLERANCE_MAX = "999.999999";

export const adjustmentEligibilityPolicySchema = z.enum(["Disallow", "AllowWithApproval"]);
export type AdjustmentEligibilityPolicy = z.infer<typeof adjustmentEligibilityPolicySchema>;

export const adjustmentAccountingDatePolicySchema = z.enum([
  "UseOriginalIfOpenElseNextOpen",
  "AlwaysNextOpen",
]);
export type AdjustmentAccountingDatePolicy = z.infer<typeof adjustmentAccountingDatePolicySchema>;

export const closedPeriodAdjustmentPolicySchema = z.enum([
  "Disallow",
  "RequireReopen",
  "PostInNextOpenPeriodWithApproval",
]);
export type ClosedPeriodAdjustmentPolicy = z.infer<typeof closedPeriodAdjustmentPolicySchema>;

export const requirementPolicySchema = z.enum(["Optional", "Required"]);
export type RequirementPolicy = z.infer<typeof requirementPolicySchema>;

export const approvalPolicySchema = z.enum(["None", "Always", "AmountThreshold"]);
export type ApprovalPolicy = z.infer<typeof approvalPolicySchema>;

export const writeOffApprovalPolicySchema = z.enum([
  "Disallow",
  "AlwaysRequireApproval",
  "RequireApprovalAboveThreshold",
]);
export type WriteOffApprovalPolicy = z.infer<typeof writeOffApprovalPolicySchema>;

export const replacementInvoiceReviewPolicySchema = z.enum([
  "NoAdditionalReview",
  "RequireReviewWhenEconomicTermsChange",
  "AlwaysRequireReview",
]);
export type ReplacementInvoiceReviewPolicy = z.infer<typeof replacementInvoiceReviewPolicySchema>;

export const customerCreditBalancePolicySchema = z.enum(["Disallow", "AllowUnappliedCredit"]);
export type CustomerCreditBalancePolicy = z.infer<typeof customerCreditBalancePolicySchema>;

export const overCreditPolicySchema = z.enum(["Block", "AllowWithApproval"]);
export type OverCreditPolicy = z.infer<typeof overCreditPolicySchema>;

export const supersededInvoiceVisibilityPolicySchema = z.enum([
  "ShowCurrentOnlyExternally",
  "ShowCurrentAndSupersededExternally",
]);
export type SupersededInvoiceVisibilityPolicy = z.infer<
  typeof supersededInvoiceVisibilityPolicySchema
>;

/**
 * An organization's invoice adjustment control, as the server sends it. The amounts and the
 * tolerance stay decimal strings, as the server sends them, so a value is saved exactly as it
 * was typed. What each must hold depends on the policy that uses it, which the form checks
 * (`invoiceAdjustmentControlFormSchema`).
 */
export const invoiceAdjustmentControlSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  partiallyPaidInvoiceAdjustmentPolicy: adjustmentEligibilityPolicySchema,
  paidInvoiceAdjustmentPolicy: adjustmentEligibilityPolicySchema,
  disputedInvoiceAdjustmentPolicy: adjustmentEligibilityPolicySchema,
  adjustmentAccountingDatePolicy: adjustmentAccountingDatePolicySchema,
  closedPeriodAdjustmentPolicy: closedPeriodAdjustmentPolicySchema,
  adjustmentReasonRequirement: requirementPolicySchema,
  standardAdjustmentApprovalPolicy: approvalPolicySchema,
  standardAdjustmentApprovalThreshold: z.string(),
  writeOffApprovalPolicy: writeOffApprovalPolicySchema,
  writeOffApprovalThreshold: z.string(),
  requireSeparateApprover: z.boolean(),
  rerateVarianceTolerancePercent: z.string(),
  replacementInvoiceReviewPolicy: replacementInvoiceReviewPolicySchema,
  customerCreditBalancePolicy: customerCreditBalancePolicySchema,
  overCreditPolicy: overCreditPolicySchema,
  supersededInvoiceVisibilityPolicy: supersededInvoiceVisibilityPolicySchema,
});

export type InvoiceAdjustmentControl = z.infer<typeof invoiceAdjustmentControlSchema>;

/** Whether `value` is a decimal of at most `scale` places that is above zero. */
export function isPositiveDecimal(value: string, scale: number): boolean {
  return decimalPattern(scale).test(value.trim()) && compareDecimalStrings(value, "0") > 0;
}

/** Whether `value` is a decimal of at most `scale` places between zero and `max`. */
export function isDecimalWithin(value: string, scale: number, max: string): boolean {
  return (
    decimalPattern(scale).test(value.trim()) &&
    compareDecimalStrings(value, "0") >= 0 &&
    compareDecimalStrings(value, max) <= 0
  );
}

/**
 * The record as the form checks it before a save, following the server's validator: an
 * approval amount must be above zero while its policy approves above an amount, and the
 * rerate tolerance must be a percentage the server can keep while replacement invoices are
 * reviewed on a change in amount. A value the page hides is not checked here; it is sent as
 * the server requires (`prepareInvoiceAdjustmentControl`).
 */
export const invoiceAdjustmentControlFormSchema = invoiceAdjustmentControlSchema.superRefine(
  (values, ctx) => {
    if (
      values.standardAdjustmentApprovalPolicy === approvalPolicySchema.enum.AmountThreshold &&
      !isPositiveDecimal(values.standardAdjustmentApprovalThreshold, ADJUSTMENT_AMOUNT_SCALE)
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["standardAdjustmentApprovalThreshold"],
        message: translate("Enter an amount above zero"),
      });
    }

    if (
      values.writeOffApprovalPolicy ===
        writeOffApprovalPolicySchema.enum.RequireApprovalAboveThreshold &&
      !isPositiveDecimal(values.writeOffApprovalThreshold, ADJUSTMENT_AMOUNT_SCALE)
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["writeOffApprovalThreshold"],
        message: translate("Enter an amount above zero"),
      });
    }

    if (
      values.replacementInvoiceReviewPolicy ===
        replacementInvoiceReviewPolicySchema.enum.RequireReviewWhenEconomicTermsChange &&
      !isDecimalWithin(
        values.rerateVarianceTolerancePercent,
        RERATE_TOLERANCE_SCALE,
        RERATE_TOLERANCE_MAX,
      )
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["rerateVarianceTolerancePercent"],
        message: translate("Enter a percentage from 0 to 999.99"),
      });
    }
  },
);
