import { z } from "zod";
import { billTypeSchema } from "./bill-type";
import { billingQueueStatusSchema } from "./billing-queue-status";
import {
  decimalStringSchema,
  nullableEnumSchema,
  nullableStringSchema,
  optionalStringSchema,
} from "./helpers";
import { customerReferenceSchema } from "./customer";
import { billingHoldReasonSchema } from "./detention";
import {
  chargeAllocationKindSchema,
  chargeAllocationMethodSchema,
  shipmentSchema,
  stopTypeSchema,
} from "./shipment";
import { userSchema } from "./user";

export { billTypeSchema, defaultBillTypeSchema, type BillType } from "./bill-type";
export { billingQueueStatusSchema, type BillingQueueStatus } from "./billing-queue-status";

export const exceptionReasonCodeSchema = z.enum([
  "MissingDocumentation",
  "IncorrectRates",
  "WeightDiscrepancy",
  "AccessorialDispute",
  "DuplicateCharge",
  "MissingReferenceNumber",
  "CustomerInformationError",
  "ServiceFailure",
  "RateNotOnFile",
  "Other",
]);
export type ExceptionReasonCode = z.infer<typeof exceptionReasonCodeSchema>;

export const payerRefSchema = z.object({
  id: z.string(),
  name: z.string(),
  code: z.string().nullish(),
});
export type PayerRef = z.infer<typeof payerRefSchema>;

/** One payer's part of one charge. */
export const payerSharePartySchema = z.object({
  payerId: z.string(),
  payerName: z.string(),
  payerCode: z.string().nullish(),
  amount: decimalStringSchema,
  percent: decimalStringSchema,
});
export type PayerShareParty = z.infer<typeof payerSharePartySchema>;

/**
 * One charge as a single payer's bill sees it. `additionalChargeId` is null for
 * freight; `method` is null for a charge billed whole.
 */
export const payerShareLineSchema = z.object({
  kind: chargeAllocationKindSchema,
  additionalChargeId: nullableStringSchema,
  description: z.string(),
  chargeTotal: decimalStringSchema,
  amount: decimalStringSchema,
  percent: decimalStringSchema,
  method: nullableEnumSchema(chargeAllocationMethodSchema),
  partial: z.boolean().default(false),
  payers: z.array(payerSharePartySchema).default([]),
});
export type PayerShareLine = z.infer<typeof payerShareLineSchema>;

/**
 * What one billing queue item bills, resolved on the server by the same code
 * that builds the invoice lines. `otherPayerLines` are charges this payer owes
 * none of; `resolutionError` explains a split that cannot be divided right now.
 */
export const payerShareSchema = z.object({
  payerId: z.string(),
  isSplit: z.boolean().default(false),
  payers: z.array(payerRefSchema).default([]),
  lines: z.array(payerShareLineSchema).default([]),
  otherPayerLines: z.array(payerShareLineSchema).default([]),
  freightAmount: decimalStringSchema,
  accessorialAmount: decimalStringSchema,
  totalAmount: decimalStringSchema,
  shipmentTotal: decimalStringSchema,
  resolutionError: nullableStringSchema,
});
export type PayerShare = z.infer<typeof payerShareSchema>;

/**
 * A detention charge on the item's shipment that is still waiting on an
 * approver. While any is listed the server refuses to approve the item.
 */
export const detentionHoldSchema = z.object({
  occurrenceId: z.string(),
  stopId: z.string(),
  stopType: stopTypeSchema,
  locationName: z.string().default(""),
  clockStartAt: z.number(),
  billableAmount: decimalStringSchema,
  currency: z.string().default("USD"),
  reason: billingHoldReasonSchema,
});
export type DetentionHold = z.infer<typeof detentionHoldSchema>;

/** Why a biller set an item aside. */
export const billingQueueHoldReasonSchema = z.enum([
  "WaitingOnPaperwork",
  "CustomerDispute",
  "RateQuestion",
]);
export type BillingQueueHoldReason = z.infer<typeof billingQueueHoldReasonSchema>;

export const billingCheckKeySchema = z.enum(["biller", "charges", "pod", "terms", "duplicate"]);
export type BillingCheckKey = z.infer<typeof billingCheckKeySchema>;

/** What choosing an issue's option does: bill as is, remove, reprice, ask, or accept. */
export const billingIssueEffectSchema = z.object({
  kind: z.enum(["keep", "drop", "set", "request", "accept"]),
  chargeId: nullableStringSchema,
  amount: decimalStringSchema.nullish(),
  basis: z.string().nullish(),
});

export const billingIssueOptionSchema = z.object({
  key: z.string(),
  label: z.string(),
  amount: decimalStringSchema.nullish(),
  done: z.string().default(""),
  effect: billingIssueEffectSchema,
});
export type BillingIssueOption = z.infer<typeof billingIssueOptionSchema>;

/** Something a person settles before the item can be approved. */
export const billingIssueSchema = z.object({
  id: z.string(),
  itemId: z.string(),
  checkKey: billingCheckKeySchema,
  code: z.string(),
  subjectKey: z.string().default(""),
  summary: z.string(),
  reasoning: z.string().nullish(),
  source: z.string().default("Deterministic"),
  flaggedChargeId: nullableStringSchema,
  options: z
    .array(billingIssueOptionSchema)
    .nullish()
    .transform((value) => value ?? []),
  resolutionKey: nullableStringSchema,
  resolutionText: z.string().nullish(),
  resolvedById: nullableStringSchema,
  resolvedAt: z.number().nullish(),
  requestedAt: z.number().nullish(),
  undoable: z.boolean().default(false),
  createdAt: z.number().default(0),
});
export type BillingIssue = z.infer<typeof billingIssueSchema>;

/** One of the five checks, as the server decided it. */
export const billingCheckSchema = z.object({
  key: billingCheckKeySchema,
  state: z.enum(["ok", "warn", "fail"]),
  code: z.string(),
  detail: z.string().default(""),
  issueId: nullableStringSchema,
  facts: z.record(z.string(), z.unknown()).nullish(),
});
export type BillingCheck = z.infer<typeof billingCheckSchema>;

export const billingChargeLineSchema = z.object({
  key: z.string(),
  additionalChargeId: nullableStringSchema,
  label: z.string(),
  basis: z.string().default(""),
  expected: decimalStringSchema.nullish(),
  billed: decimalStringSchema,
  source: z.string().default("none"),
  flagged: z.boolean().default(false),
  removed: z.boolean().default(false),
  adjusted: z.boolean().default(false),
  issueId: nullableStringSchema,
});
export type BillingChargeLine = z.infer<typeof billingChargeLineSchema>;

export const billingChargeReviewSchema = z.object({
  lines: z
    .array(billingChargeLineSchema)
    .nullish()
    .transform((value) => value ?? []),
  expectedTotal: decimalStringSchema,
  billedTotal: decimalStringSchema,
  difference: decimalStringSchema,
  hasRateCon: z.boolean().default(false),
});
export type BillingChargeReview = z.infer<typeof billingChargeReviewSchema>;

export const billingInvoiceRefSchema = z.object({
  id: z.string(),
  number: z.string(),
  draftNumber: z.string().default(""),
  status: z.string(),
  invoiceDate: z.number().nullish(),
  dueDate: z.number().nullish(),
  postedAt: z.number().nullish(),
  posted: z.boolean().default(false),
});

export const billingTermsSchema = z.object({
  paymentTerm: z.string().default(""),
  netDays: z.number().default(0),
  dueDate: z.number().nullish(),
  recipients: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  creditHold: z.boolean().default(false),
});

export const billingDocumentTileSchema = z.object({
  code: z.string().default(""),
  name: z.string().default(""),
  documentId: nullableStringSchema,
  state: z.enum(["ok", "missing", "unsigned"]),
  required: z.boolean().default(false),
  signed: z.boolean().default(false),
  fileName: z.string().default(""),
});
export type BillingDocumentTile = z.infer<typeof billingDocumentTileSchema>;

/** The item as a biller reviews it; present when it was read with its shipment. */
export const billingReviewSchema = z.object({
  checks: z
    .array(billingCheckSchema)
    .nullish()
    .transform((value) => value ?? []),
  needsCount: z.number().default(0),
  ready: z.boolean().default(false),
  blocker: z.string().default(""),
  issues: z
    .array(billingIssueSchema)
    .nullish()
    .transform((value) => value ?? []),
  charges: billingChargeReviewSchema.nullish(),
  invoice: billingInvoiceRefSchema.nullish(),
  terms: billingTermsSchema.nullish(),
  documents: z
    .array(billingDocumentTileSchema)
    .nullish()
    .transform((value) => value ?? []),
  billerName: z.string().default(""),
});
export type BillingReview = z.infer<typeof billingReviewSchema>;

export const billingQueueItemSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  shipmentId: z.string(),
  /** The customer this item bills. A split shipment has one item per payer. */
  billToCustomerId: nullableStringSchema,
  /** This payer's share of the shipment's charges. */
  allocatedTotalAmount: decimalStringSchema.nullish(),
  assignedBillerId: nullableStringSchema,
  number: z.string().optional(),
  status: billingQueueStatusSchema,
  billType: billTypeSchema,
  exceptionReasonCode: nullableStringSchema,
  reviewNotes: optionalStringSchema,
  exceptionNotes: optionalStringSchema,
  reviewStartedAt: z.number().nullable().optional(),
  reviewCompletedAt: z.number().nullable().optional(),
  canceledById: nullableStringSchema,
  canceledAt: z.number().nullable().optional(),
  cancelReason: optionalStringSchema,
  isAdjustmentOrigin: z.boolean().default(false),
  sourceInvoiceId: nullableStringSchema,
  sourceInvoiceAdjustmentId: nullableStringSchema,
  sourceCreditMemoInvoiceId: nullableStringSchema,
  correctionGroupId: nullableStringSchema,
  rebillStrategy: nullableStringSchema,
  requiresReplacementReview: z.boolean().default(false),
  rerateVariancePercent: decimalStringSchema.nullish().default(0),
  adjustmentContext: z.record(z.string(), z.unknown()).default({}),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  shipment: shipmentSchema.optional(),
  billToCustomer: customerReferenceSchema.optional().nullable(),
  assignedBiller: userSchema.optional().nullable(),
  canceledBy: userSchema.optional().nullable(),
  payerShare: payerShareSchema.nullish(),
  detentionHolds: z
    .array(detentionHoldSchema)
    .nullish()
    .transform((value) => value ?? []),
  holdReasonCode: nullableEnumSchema(billingQueueHoldReasonSchema),
  heldAt: z.number().nullish(),
  heldById: nullableStringSchema,
  statusBeforeHold: nullableEnumSchema(billingQueueStatusSchema),
  poNumber: z.string().nullish(),
  review: billingReviewSchema.nullish(),
});

export type BillingQueueItem = z.infer<typeof billingQueueItemSchema>;

/** One line of an item's activity. */
export const billingQueueEventSchema = z.object({
  id: z.string(),
  kind: z.string(),
  text: z.string(),
  actorType: z.enum(["System", "Agent", "User"]),
  actorId: nullableStringSchema,
  actorName: z.string().nullish(),
  at: z.number(),
});
export type BillingQueueEvent = z.infer<typeof billingQueueEventSchema>;

export const billingQueueEventPageSchema = z.object({
  items: z
    .array(billingQueueEventSchema)
    .nullish()
    .transform((value) => value ?? []),
  hasMore: z.boolean().default(false),
  total: z.number().default(0),
});
export type BillingQueueEventPage = z.infer<typeof billingQueueEventPageSchema>;

/** An item's place in the queue, under the list's own order. */
export const billingQueueNeighborsSchema = z.object({
  prevId: nullableStringSchema,
  nextId: nullableStringSchema,
  position: z.number(),
  total: z.number(),
});
export type BillingQueueNeighbors = z.infer<typeof billingQueueNeighborsSchema>;

/** A queue row's live state. */
export const billingQueueSummarySchema = z.object({
  id: z.string(),
  number: z.string().default(""),
  status: billingQueueStatusSchema,
  holdReasonCode: nullableEnumSchema(billingQueueHoldReasonSchema),
  assignedBillerId: nullableStringSchema,
  allocatedTotalAmount: decimalStringSchema,
  needsCount: z.number().default(0),
  ready: z.boolean().default(false),
});
export type BillingQueueSummary = z.infer<typeof billingQueueSummarySchema>;

export const billingQueueApprovalRunItemSchema = z.object({
  itemId: z.string(),
  status: z.enum(["Pending", "Approved", "Failed", "Skipped"]),
  failureCode: z.string().nullish(),
  errorMessage: z.string().nullish(),
  invoiceNumber: z.string().nullish(),
});

/** One press of Approve over several items, run as a job behind an undo window. */
export const billingQueueApprovalRunSchema = z.object({
  id: z.string(),
  status: z.enum(["Scheduled", "Running", "Completed", "Undone", "Failed"]),
  totalCount: z.number(),
  approvedCount: z.number().default(0),
  failedCount: z.number().default(0),
  skippedCount: z.number().default(0),
  commitAt: z.number(),
  cancelRequestedAt: z.number().nullish(),
  failureMessage: z.string().nullish(),
  items: z
    .array(billingQueueApprovalRunItemSchema)
    .nullish()
    .transform((value) => value ?? []),
});
export type BillingQueueApprovalRun = z.infer<typeof billingQueueApprovalRunSchema>;

export const billingQueuePostResultSchema = z.object({
  item: billingQueueItemSchema,
  invoiceId: z.string(),
  invoiceNumber: z.string(),
  sentTo: z.string().nullish(),
  recipients: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
});
export type BillingQueuePostResult = z.infer<typeof billingQueuePostResultSchema>;

/**
 * The queue after a charge moved between payers: the item the change came from
 * (canceled when its payer no longer pays anything) and every active item.
 */
export const reassignChargeResultSchema = z.object({
  item: billingQueueItemSchema,
  items: z.array(billingQueueItemSchema).default([]),
  createdItemIds: z.array(z.string()).default([]),
  canceledItemIds: z.array(z.string()).default([]),
});
export type ReassignChargeResult = z.infer<typeof reassignChargeResultSchema>;

export type ReassignChargeAllocationInput = {
  id?: string;
  billToCustomerId: string;
  method: "Percent" | "Amount";
  percent: string | null;
  amount: string | null;
};

export type ReassignChargeInput = {
  chargeKind: "Freight" | "Accessorial";
  additionalChargeId?: string;
  allocations: ReassignChargeAllocationInput[];
};

export const billingQueueTransferSchema = z.object({
  shipmentId: z.string(),
  billType: billTypeSchema.optional(),
});
export type BillingQueueTransferInput = z.infer<typeof billingQueueTransferSchema>;

export const billingQueueAssignSchema = z.object({
  billerId: z.string(),
});
export type BillingQueueAssignInput = z.infer<typeof billingQueueAssignSchema>;

export const billingQueueUpdateStatusSchema = z.object({
  status: billingQueueStatusSchema,
  exceptionReasonCode: exceptionReasonCodeSchema.optional(),
  exceptionNotes: z.string().optional(),
  reviewNotes: z.string().optional(),
  cancelReason: z.string().optional(),
  holdReasonCode: billingQueueHoldReasonSchema.optional(),
});
export type BillingQueueUpdateStatusInput = z.infer<typeof billingQueueUpdateStatusSchema>;

export const billingQueueStatsSchema = z.object({
  readyForReview: z.number(),
  inReview: z.number(),
  approved: z.number(),
  posted: z.number(),
  onHold: z.number(),
  exception: z.number(),
  sentBackToOps: z.number(),
  canceled: z.number(),
  total: z.number(),
});
export type BillingQueueStats = z.infer<typeof billingQueueStatsSchema>;

export const billingQueueUpdateChargesSchema = z.object({
  formulaTemplateId: z.string().optional(),
  baseRate: z.string().optional(),
  additionalCharges: z
    .array(
      z.object({
        id: z.string().optional(),
        accessorialChargeId: z.string(),
        method: z.string(),
        amount: z.union([z.string(), z.number()]),
        unit: z.number().int().min(1).default(1),
      }),
    )
    .optional(),
  /** Retries an edit refused because it left an amount split that no longer adds up. */
  convertAmountSplitsToPercent: z.boolean().optional(),
});
export type BillingQueueUpdateChargesInput = z.infer<typeof billingQueueUpdateChargesSchema>;

export const billingQueueFilterPresetSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  userId: z.string(),
  name: z.string(),
  filters: z.record(z.string(), z.any()),
  isDefault: z.boolean(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
});
export type BillingQueueFilterPreset = z.infer<typeof billingQueueFilterPresetSchema>;

export const billingQueueFilterPresetInputSchema = z.object({
  name: z.string().min(1).max(100),
  filters: z.record(z.string(), z.any()),
  isDefault: z.boolean().optional(),
});
export type BillingQueueFilterPresetInput = z.infer<typeof billingQueueFilterPresetInputSchema>;
