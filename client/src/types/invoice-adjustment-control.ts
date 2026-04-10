import { z } from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

const decimalNumberSchema = z.coerce.number().finite();

export const adjustmentEligibilityPolicySchema = z.enum([
  "Disallow",
  "AllowWithApproval",
  "AllowWithoutApproval",
]);
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

export const adjustmentAttachmentPolicySchema = z.enum([
  "Optional",
  "RequiredForCreditOrWriteOff",
  "RequiredForAll",
]);
export type AdjustmentAttachmentPolicy = z.infer<typeof adjustmentAttachmentPolicySchema>;

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

export const customerCreditBalancePolicySchema = z.enum([
  "Disallow",
  "AllowUnappliedCredit",
]);
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
  adjustmentAttachmentRequirement: adjustmentAttachmentPolicySchema,
  standardAdjustmentApprovalPolicy: approvalPolicySchema,
  standardAdjustmentApprovalThreshold: decimalNumberSchema,
  writeOffApprovalPolicy: writeOffApprovalPolicySchema,
  writeOffApprovalThreshold: decimalNumberSchema,
  rerateVarianceTolerancePercent: decimalNumberSchema,
  replacementInvoiceReviewPolicy: replacementInvoiceReviewPolicySchema,
  customerCreditBalancePolicy: customerCreditBalancePolicySchema,
  overCreditPolicy: overCreditPolicySchema,
  supersededInvoiceVisibilityPolicy: supersededInvoiceVisibilityPolicySchema,
});

export type InvoiceAdjustmentControl = z.infer<typeof invoiceAdjustmentControlSchema>;
