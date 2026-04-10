import { z } from "zod";
import { invoiceAdjustmentSupportingDocumentPolicySchema } from "./customer";
import { documentSchema } from "./document";
import { decimalStringSchema, nullableStringSchema, tenantInfoSchema } from "./helpers";
import { invoiceLineSchema, invoiceSchema } from "./invoice";

export const invoiceAdjustmentKindSchema = z.enum([
  "CreditOnly",
  "CreditAndRebill",
  "FullReversal",
  "WriteOff",
]);
export type InvoiceAdjustmentKind = z.infer<typeof invoiceAdjustmentKindSchema>;

export const invoiceAdjustmentStatusSchema = z.enum([
  "Draft",
  "PendingApproval",
  "Approved",
  "Rejected",
  "Executing",
  "Executed",
  "ExecutionFailed",
]);
export type InvoiceAdjustmentStatus = z.infer<typeof invoiceAdjustmentStatusSchema>;

export const approvalStatusSchema = z.enum(["NotRequired", "Pending", "Approved", "Rejected"]);
export type ApprovalStatus = z.infer<typeof approvalStatusSchema>;

export const replacementReviewStatusSchema = z.enum(["NotRequired", "Required", "Completed"]);
export type ReplacementReviewStatus = z.infer<typeof replacementReviewStatusSchema>;
export const reconciliationExceptionStatusSchema = z.enum(["Open", "Resolved"]);
export type ReconciliationExceptionStatus = z.infer<typeof reconciliationExceptionStatusSchema>;
export const rebillStrategySchema = z.enum(["CloneExact", "Rerate", "Manual"]);
export type RebillStrategy = z.infer<typeof rebillStrategySchema>;

const supportingDocumentPolicyValueSchema = z.preprocess((value) => {
  if (typeof value !== "string" || value.length === 0) {
    return "Inherit";
  }

  return value;
}, invoiceAdjustmentSupportingDocumentPolicySchema.catch("Inherit"));

export const invoiceAdjustmentLineInputSchema = z.object({
  originalLineId: z.string(),
  creditQuantity: decimalStringSchema,
  creditAmount: decimalStringSchema,
  rebillQuantity: decimalStringSchema,
  rebillAmount: decimalStringSchema,
  description: z.string().optional().default(""),
  replacementPayload: z.record(z.string(), z.unknown()).optional().default({}),
});
export type InvoiceAdjustmentLineInput = z.infer<typeof invoiceAdjustmentLineInputSchema>;

export const invoiceAdjustmentPreviewLineSchema = z.object({
  lineNumber: z.number().int(),
  originalLineId: z.string(),
  description: z.string(),
  eligibleAmount: decimalStringSchema,
  alreadyCreditedAmount: decimalStringSchema,
  requestedCreditAmount: decimalStringSchema,
  requestedRebillAmount: decimalStringSchema,
  remainingEligibleAmount: decimalStringSchema,
  hasEligibilityError: z.boolean().default(false),
  eligibilityOverageAmount: decimalStringSchema.default(0),
  eligibilityMessage: z.string().default(""),
});
export type InvoiceAdjustmentPreviewLine = z.infer<typeof invoiceAdjustmentPreviewLineSchema>;

export const invoiceAdjustmentPreviewSchema = z.object({
  invoiceId: z.string(),
  correctionGroupId: z.string(),
  kind: invoiceAdjustmentKindSchema,
  rebillStrategy: rebillStrategySchema,
  accountingDate: z.number(),
  creditTotalAmount: decimalStringSchema,
  rebillTotalAmount: decimalStringSchema,
  netDeltaAmount: decimalStringSchema,
  rerateVariancePercent: decimalStringSchema,
  wouldCreateUnappliedCredit: z.boolean(),
  requiresApproval: z.boolean(),
  requiresReplacementInvoiceReview: z.boolean(),
  requiresReconciliationException: z.boolean(),
  customerSupportingDocumentPolicy: supportingDocumentPolicyValueSchema,
  supportingDocumentsRequired: z.boolean().default(false),
  supportingDocumentPolicySource: z.string().default("OrganizationControl"),
  warnings: z.array(z.string()).default([]),
  errors: z.record(z.string(), z.array(z.string())).default({}),
  lines: z.array(invoiceAdjustmentPreviewLineSchema).default([]),
});
export type InvoiceAdjustmentPreview = z.infer<typeof invoiceAdjustmentPreviewSchema>;

export const invoiceAdjustmentLineSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  adjustmentId: z.string(),
  originalInvoiceId: z.string(),
  originalLineId: z.string(),
  creditMemoLineId: nullableStringSchema,
  replacementLineId: nullableStringSchema,
  lineNumber: z.number().int(),
  description: z.string(),
  creditQuantity: decimalStringSchema,
  creditAmount: decimalStringSchema,
  remainingEligibleAmount: decimalStringSchema,
  rebillQuantity: decimalStringSchema,
  rebillAmount: decimalStringSchema,
  replacementPayload: z.record(z.string(), z.unknown()).default({}),
  createdAt: z.number(),
  updatedAt: z.number(),
});
export type InvoiceAdjustmentLine = z.infer<typeof invoiceAdjustmentLineSchema>;

export const invoiceAdjustmentDocumentReferenceSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  adjustmentId: z.string(),
  documentId: z.string(),
  selectedById: nullableStringSchema,
  selectedAt: z.number().nullish(),
  snapshotFileName: z.string(),
  snapshotOriginalName: z.string(),
  snapshotFileType: z.string(),
  snapshotResourceType: z.string(),
  snapshotResourceId: z.string(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  document: documentSchema.optional(),
});
export type InvoiceAdjustmentDocumentReference = z.infer<
  typeof invoiceAdjustmentDocumentReferenceSchema
>;

export const invoiceAdjustmentSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  correctionGroupId: z.string(),
  originalInvoiceId: z.string(),
  creditMemoInvoiceId: nullableStringSchema,
  replacementInvoiceId: nullableStringSchema,
  rebillQueueItemId: nullableStringSchema,
  batchId: nullableStringSchema,
  kind: invoiceAdjustmentKindSchema,
  status: invoiceAdjustmentStatusSchema,
  approvalStatus: approvalStatusSchema,
  replacementReviewStatus: replacementReviewStatusSchema,
  rebillStrategy: rebillStrategySchema.nullish(),
  reason: z.string().nullish().default(""),
  policyReason: z.string().nullish().default(""),
  idempotencyKey: z.string(),
  accountingDate: z.number(),
  creditTotalAmount: decimalStringSchema,
  rebillTotalAmount: decimalStringSchema,
  netDeltaAmount: decimalStringSchema,
  rerateVariancePercent: decimalStringSchema,
  wouldCreateUnappliedCredit: z.boolean(),
  requiresReconciliationException: z.boolean(),
  approvalRequired: z.boolean(),
  submittedById: nullableStringSchema,
  submittedAt: z.number().nullish(),
  approvedById: nullableStringSchema,
  approvedAt: z.number().nullish(),
  rejectedById: nullableStringSchema,
  rejectedAt: z.number().nullish(),
  rejectionReason: z.string().nullish().default(""),
  executionError: z.string().nullish().default(""),
  metadata: z.record(z.string(), z.unknown()).default({}),
  customerSupportingDocumentPolicy: supportingDocumentPolicyValueSchema,
  supportingDocumentsRequired: z.boolean().default(false),
  supportingDocumentPolicySource: z.string().default("OrganizationControl"),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  lines: z.array(invoiceAdjustmentLineSchema).default([]),
  referencedDocuments: z.array(invoiceAdjustmentDocumentReferenceSchema).default([]),
  adjustmentDocuments: z.array(documentSchema).default([]),
});
export type InvoiceAdjustment = z.infer<typeof invoiceAdjustmentSchema>;

export const invoiceAdjustmentRequestSchema = z.object({
  adjustmentId: z.string().optional(),
  invoiceId: z.string(),
  kind: invoiceAdjustmentKindSchema,
  rebillStrategy: rebillStrategySchema.default("CloneExact"),
  reason: z.string().default(""),
  idempotencyKey: z.string(),
  attachmentIds: z.array(z.string()).default([]),
  lines: z.array(invoiceAdjustmentLineInputSchema).default([]),
});
export type InvoiceAdjustmentRequest = z.infer<typeof invoiceAdjustmentRequestSchema>;

export const createDraftInvoiceAdjustmentRequestSchema = z.object({
  invoiceId: z.string(),
});
export type CreateDraftInvoiceAdjustmentRequest = z.infer<
  typeof createDraftInvoiceAdjustmentRequestSchema
>;

export const updateDraftInvoiceAdjustmentRequestSchema = z.object({
  kind: invoiceAdjustmentKindSchema,
  rebillStrategy: rebillStrategySchema.default("CloneExact"),
  reason: z.string().default(""),
  referencedDocumentIds: z.array(z.string()).default([]),
  lines: z.array(invoiceAdjustmentLineInputSchema).default([]),
});
export type UpdateDraftInvoiceAdjustmentRequest = z.infer<
  typeof updateDraftInvoiceAdjustmentRequestSchema
>;

export const correctionGroupSchema = z.object({
  ...tenantInfoSchema.shape,
  rootInvoiceId: z.string(),
  currentInvoiceId: nullableStringSchema,
  metadata: z.record(z.string(), z.unknown()).default({}),
});
export type CorrectionGroup = z.infer<typeof correctionGroupSchema>;

export const invoiceAdjustmentLineageSchema = z.object({
  correctionGroup: correctionGroupSchema,
  invoices: z.array(invoiceSchema).default([]),
  adjustments: z.array(invoiceAdjustmentSchema).default([]),
});
export type InvoiceAdjustmentLineage = z.infer<typeof invoiceAdjustmentLineageSchema>;

export const invoiceApprovalQueueItemSchema = z.object({
  adjustmentId: z.string(),
  correctionGroupId: z.string(),
  originalInvoiceId: z.string(),
  originalInvoiceNumber: z.string(),
  originalInvoiceStatus: z.string(),
  customerName: z.string(),
  kind: invoiceAdjustmentKindSchema,
  status: invoiceAdjustmentStatusSchema,
  approvalStatus: approvalStatusSchema,
  rebillStrategy: rebillStrategySchema.nullish(),
  reason: z.string().default(""),
  policyReason: z.string().default(""),
  policySource: z.string().default(""),
  creditTotalAmount: decimalStringSchema,
  rebillTotalAmount: decimalStringSchema,
  netDeltaAmount: decimalStringSchema,
  rerateVariancePercent: decimalStringSchema,
  wouldCreateUnappliedCredit: z.boolean(),
  requiresReconciliationException: z.boolean(),
  requiresReplacementInvoiceReview: z.boolean(),
  submittedById: nullableStringSchema,
  submittedByName: z.string().default(""),
  submittedAt: z.number().nullish(),
  approvedById: nullableStringSchema,
  approvedByName: z.string().default(""),
  approvedAt: z.number().nullish(),
  rejectedById: nullableStringSchema,
  rejectedByName: z.string().default(""),
  rejectedAt: z.number().nullish(),
  rejectionReason: z.string().default(""),
  creditMemoInvoiceId: nullableStringSchema,
  creditMemoInvoiceNumber: z.string().default(""),
  replacementInvoiceId: nullableStringSchema,
  replacementInvoiceNumber: z.string().default(""),
  rebillQueueItemId: nullableStringSchema,
  rebillQueueNumber: z.string().default(""),
  batchId: nullableStringSchema,
  createdAt: z.number(),
  updatedAt: z.number(),
});
export type InvoiceApprovalQueueItem = z.infer<typeof invoiceApprovalQueueItemSchema>;

export const invoiceReconciliationQueueItemSchema = z.object({
  exceptionId: z.string(),
  adjustmentId: z.string(),
  correctionGroupId: z.string(),
  status: reconciliationExceptionStatusSchema,
  reason: z.string(),
  amount: decimalStringSchema,
  originalInvoiceId: z.string(),
  originalInvoiceNumber: z.string(),
  originalInvoiceStatus: z.string(),
  creditMemoInvoiceId: nullableStringSchema,
  creditMemoInvoiceNumber: z.string().default(""),
  replacementInvoiceId: nullableStringSchema,
  replacementInvoiceNumber: z.string().default(""),
  rebillQueueItemId: nullableStringSchema,
  rebillQueueNumber: z.string().default(""),
  customerName: z.string(),
  adjustmentKind: invoiceAdjustmentKindSchema,
  adjustmentStatus: invoiceAdjustmentStatusSchema,
  policySource: z.string().default(""),
  submittedById: nullableStringSchema,
  submittedByName: z.string().default(""),
  submittedAt: z.number().nullish(),
  financeNotes: z.string().default(""),
  createdAt: z.number(),
  updatedAt: z.number(),
});
export type InvoiceReconciliationQueueItem = z.infer<typeof invoiceReconciliationQueueItemSchema>;

export const invoiceAdjustmentBatchItemStatusSchema = z.enum([
  "Pending",
  "Previewed",
  "Submitted",
  "PendingApproval",
  "Executing",
  "Executed",
  "Rejected",
  "Failed",
]);
export const invoiceAdjustmentBatchStatusSchema = z.enum([
  "Pending",
  "Running",
  "Completed",
  "Failed",
  "PartialSuccess",
  "Submitted",
  "Queued",
]);

export const invoiceAdjustmentBatchItemSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  batchId: z.string(),
  adjustmentId: nullableStringSchema,
  invoiceId: z.string(),
  idempotencyKey: z.string(),
  status: invoiceAdjustmentBatchItemStatusSchema,
  errorMessage: z.string().nullish().default(""),
  requestPayload: z.record(z.string(), z.unknown()).default({}),
  resultPayload: z.record(z.string(), z.unknown()).default({}),
  createdAt: z.number(),
  updatedAt: z.number(),
});
export type InvoiceAdjustmentBatchItem = z.infer<typeof invoiceAdjustmentBatchItemSchema>;

export const invoiceAdjustmentBatchSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  idempotencyKey: z.string(),
  status: invoiceAdjustmentBatchStatusSchema,
  totalCount: z.number().int(),
  processedCount: z.number().int(),
  succeededCount: z.number().int(),
  failedCount: z.number().int(),
  submittedById: nullableStringSchema,
  submittedAt: z.number().nullish(),
  metadata: z.record(z.string(), z.unknown()).default({}),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  items: z.array(invoiceAdjustmentBatchItemSchema).default([]),
});
export type InvoiceAdjustmentBatch = z.infer<typeof invoiceAdjustmentBatchSchema>;

export const invoiceAdjustmentBatchQueueItemSchema = z.object({
  batchId: z.string(),
  idempotencyKey: z.string(),
  status: invoiceAdjustmentBatchStatusSchema,
  totalCount: z.number().int(),
  processedCount: z.number().int(),
  succeededCount: z.number().int(),
  failedCount: z.number().int(),
  pendingCount: z.number().int(),
  submittedById: nullableStringSchema,
  submittedByName: z.string().default(""),
  submittedAt: z.number().nullish(),
  lastFailure: z.string().default(""),
  lastFailureCount: z.number().int(),
  createdAt: z.number(),
  updatedAt: z.number(),
});
export type InvoiceAdjustmentBatchQueueItem = z.infer<typeof invoiceAdjustmentBatchQueueItemSchema>;

export const invoiceAdjustmentSummaryCountSchema = z.object({
  label: z.string(),
  count: z.number().int(),
});
export type InvoiceAdjustmentSummaryCount = z.infer<typeof invoiceAdjustmentSummaryCountSchema>;

export const repeatedAdjustmentSummarySchema = z.object({
  entityId: z.string(),
  entityType: z.string(),
  label: z.string(),
  count: z.number().int(),
});
export type RepeatedAdjustmentSummary = z.infer<typeof repeatedAdjustmentSummarySchema>;

export const invoiceAdjustmentOperationsSummarySchema = z.object({
  adjustmentsByStatus: z.array(invoiceAdjustmentSummaryCountSchema).default([]),
  approvalsPending: z.number().int(),
  reconciliationPending: z.number().int(),
  writeOffPending: z.number().int(),
  batchesInFlight: z.number().int(),
  failedBatchItems: z.number().int(),
  reasonDistribution: z.array(invoiceAdjustmentSummaryCountSchema).default([]),
  repeatedAdjustments: z.array(repeatedAdjustmentSummarySchema).default([]),
  repeatedCustomerAdjustments: z.array(repeatedAdjustmentSummarySchema).default([]),
});
export type InvoiceAdjustmentOperationsSummary = z.infer<
  typeof invoiceAdjustmentOperationsSummarySchema
>;

export const replacementLineSchema = invoiceLineSchema.omit({
  id: true,
  invoiceId: true,
  organizationId: true,
  businessUnitId: true,
});
