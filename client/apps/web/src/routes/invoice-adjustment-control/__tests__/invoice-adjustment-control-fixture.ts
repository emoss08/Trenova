import type { InvoiceAdjustmentControl } from "@/types/invoice-adjustment-control";

/**
 * A new organization's invoice adjustment control as the server sends it: the Go entity's
 * JSON from `NewDefaultInvoiceAdjustmentControl`, with its decimals as quoted strings.
 */
export const savedInvoiceAdjustmentControl: InvoiceAdjustmentControl = {
  id: "iac_01",
  version: 2,
  createdAt: 1_793_577_600,
  updatedAt: 1_793_577_600,
  organizationId: "org_01",
  businessUnitId: "bu_01",
  partiallyPaidInvoiceAdjustmentPolicy: "AllowWithApproval",
  paidInvoiceAdjustmentPolicy: "Disallow",
  disputedInvoiceAdjustmentPolicy: "AllowWithApproval",
  adjustmentAccountingDatePolicy: "UseOriginalIfOpenElseNextOpen",
  closedPeriodAdjustmentPolicy: "PostInNextOpenPeriodWithApproval",
  adjustmentReasonRequirement: "Required",
  standardAdjustmentApprovalPolicy: "AmountThreshold",
  standardAdjustmentApprovalThreshold: "0.01",
  writeOffApprovalPolicy: "RequireApprovalAboveThreshold",
  writeOffApprovalThreshold: "0.01",
  requireSeparateApprover: false,
  rerateVarianceTolerancePercent: "0",
  replacementInvoiceReviewPolicy: "RequireReviewWhenEconomicTermsChange",
  customerCreditBalancePolicy: "AllowUnappliedCredit",
  overCreditPolicy: "Block",
  supersededInvoiceVisibilityPolicy: "ShowCurrentOnlyExternally",
};

/**
 * A saved control where every setting that waits on another is closed: adjustments always
 * book in the next open period, approval and write-offs take no amount, no unapplied credit
 * and replacement invoices are always reviewed.
 */
export const closedInvoiceAdjustmentControl: InvoiceAdjustmentControl = {
  ...savedInvoiceAdjustmentControl,
  adjustmentAccountingDatePolicy: "AlwaysNextOpen",
  closedPeriodAdjustmentPolicy: "Disallow",
  standardAdjustmentApprovalPolicy: "Always",
  standardAdjustmentApprovalThreshold: "0",
  writeOffApprovalPolicy: "AlwaysRequireApproval",
  writeOffApprovalThreshold: "0",
  replacementInvoiceReviewPolicy: "AlwaysRequireReview",
  customerCreditBalancePolicy: "Disallow",
};
