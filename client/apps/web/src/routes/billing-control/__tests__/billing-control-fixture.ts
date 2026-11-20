import type { BillingControl } from "@/types/billing-control";

/**
 * A saved billing control as the server sends it (the Go entity's JSON, decimals as text),
 * with every row that waits on another closed: transfers, drafts, posting and approval by
 * hand, both checks set to block, reminders and statements off.
 */
export const savedBillingControl = {
  id: "bc_01",
  version: 4,
  createdAt: 1_793_577_600,
  updatedAt: 1_793_577_600,
  organizationId: "org_01",
  businessUnitId: "bu_01",
  defaultPaymentTerm: "Net30",
  defaultInvoiceTerms: "",
  defaultInvoiceFooter: "",
  showDueDateOnInvoice: true,
  showBalanceDueOnInvoice: true,
  readyToBillAssignmentMode: "ManualOnly",
  billingQueueTransferMode: "ManualOnly",
  billingQueueTransferSchedule: "",
  billingQueueTransferBatchSize: 0,
  invoiceDraftCreationMode: "ManualOnly",
  invoicePostingMode: "ManualReviewRequired",
  autoInvoiceBatchSize: 0,
  notifyOnAutoInvoiceCreation: false,
  shipmentBillingRequirementEnforcement: "Block",
  rateValidationEnforcement: "Block",
  billingExceptionDisposition: "RouteToBillingReview",
  notifyOnBillingExceptions: true,
  rateVarianceTolerancePercent: "0",
  rateVarianceAutoResolutionMode: "Disabled",
  unratedShipmentDisposition: "ZeroAndFlag",
  fallbackFormulaTemplateId: null,
  requireRateOverrideReason: true,
  enforceMarginFloor: false,
  queueAutoApproveMode: "Off",
  queueAutoApproveMaxAmount: null,
  queueAgentReviewEnabled: true,
  lateChargeAssessmentMode: "Disabled",
  lateChargeMinimumAmount: "0",
  paymentRemindersEnabled: false,
  paymentReminderDays: [1, 15, 30],
  monthlyStatementsEnabled: false,
  monthlyStatementDay: 1,
} as const;

/** The same record as the form holds it, once the schema has read the server's JSON. */
export const billingControlValues: BillingControl = {
  ...savedBillingControl,
  billingQueueTransferSchedule: null,
  rateVarianceTolerancePercent: 0,
  lateChargeMinimumAmount: 0,
  paymentReminderDays: [1, 15, 30],
};
