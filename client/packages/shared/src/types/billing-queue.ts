import { linkedRecordSchema } from "@trenova/shared/lib/linked-record";
import { z } from "zod";
import { billTypeSchema } from "./bill-type";
import { billingQueueStatusSchema } from "./billing-queue-status";
import { invoiceMemoKindSchema } from "./invoice-memo-kind";
import {
  decimalStringSchema,
  nullableEnumSchema,
  nullableStringSchema,
  optionalStringSchema,
} from "./helpers";
import { customerReferenceSchema } from "./customer";
import { tariffFormInputValuesSchema } from "./tariff-form";
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

/**
 * The reviews a tariff asked for on the item's shipment, keeping the item from being approved
 * until a person approves the rating.
 */
export const ratingReviewHoldSchema = z.object({
  shipmentId: z.string(),
  reasons: z.array(z.string()).default([]),
  ratedAt: z.number().nullish(),
  formulaTemplateName: z.string().nullish(),
});
export type RatingReviewHold = z.infer<typeof ratingReviewHoldSchema>;

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
  kind: z.enum(["keep", "drop", "set", "request", "accept", "add"]),
  chargeId: nullableStringSchema,
  accessorialChargeId: nullableStringSchema,
  /** The charge an add effect bills. */
  label: z.string().nullish(),
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

/** Where an item sits in the queue's working order, stored by the server. */
export const billingQueueLaneSchema = z.enum([
  "Checking",
  "NeedsYou",
  "Waiting",
  "Ready",
  "Held",
  "Approved",
  "Closed",
]);
export type BillingQueueLane = z.infer<typeof billingQueueLaneSchema>;

export const BILLING_QUEUE_OPEN_LANES = [
  "NeedsYou",
  "Ready",
  "Waiting",
  "Checking",
  "Held",
  "Approved",
] as const satisfies readonly BillingQueueLane[];

/** What Nova made of an item it reviewed; it never blocks approval. */
export const agentVerdictSchema = z.enum(["Agrees", "Concern"]);
export type AgentVerdict = z.infer<typeof agentVerdictSchema>;

/**
 * One thing Nova read to review an item: what the review was given and what it read
 * further. A document carries how many of its parts were read (`part` of `parts`); one
 * with no text has neither. A charge looked up carries how many of the payer's invoices
 * billed it (`count` of `of`).
 */
export const agentReadSchema = z.object({
  kind: z.enum([
    "Invoice",
    "Stops",
    "Terms",
    "BillingNotes",
    "Comments",
    "PastInvoices",
    "ChargeHistory",
    "Document",
  ]),
  count: z.number().nullish(),
  code: z.string().nullish(),
  documentId: z.string().nullish(),
  name: z.string().nullish(),
  fileName: z.string().nullish(),
  part: z.number().nullish(),
  parts: z.number().nullish(),
  of: z.number().nullish(),
});
export type AgentRead = z.infer<typeof agentReadSchema>;

/** The open issue a list row leads with, and how many are open. */
export const billingQueueHeadlineSchema = z.object({
  code: z.string(),
  checkKey: billingCheckKeySchema,
  summary: z.string(),
  open: z.number().default(0),
  waiting: z.boolean().default(false),
});
export type BillingQueueHeadline = z.infer<typeof billingQueueHeadlineSchema>;

/**
 * What priced an item's shipment, for the biller to open: the rate agreement when one
 * did, or the formula template.
 */
export const billingQueueRatedBySchema = z.object({
  kind: z.enum(["Agreement", "Template"]),
  id: z.string(),
  code: z.string().nullish(),
  name: z.string(),
});
export type BillingQueueRatedBy = z.infer<typeof billingQueueRatedBySchema>;

/**
 * The replacement invoice a credit-and-rebill prices. The REST item carries its replacement
 * lines and the GraphQL action fields only their count; both read as `lineCount`.
 */
const billingQueueAdjustmentRebillSchema = z
  .object({
    replacementLines: z.array(z.unknown()).nullish(),
    replacementLineCount: z.number().int().nullish(),
    subtotalAmount: decimalStringSchema,
    otherAmount: decimalStringSchema,
    totalAmount: decimalStringSchema,
    accountingDate: z.number(),
  })
  .transform(({ replacementLines, replacementLineCount, ...rebill }) => ({
    ...rebill,
    lineCount: replacementLineCount ?? replacementLines?.length ?? 0,
  }));

/** A memo a queue item bills, and who raised it. */
const billingQueueAdjustmentMemoSchema = z.object({
  kind: invoiceMemoKindSchema,
  raisedById: nullableStringSchema,
});

/** What an adjustment or memo item bills from; each part is null when it does not apply. */
export const billingQueueAdjustmentContextSchema = z.object({
  rebill: billingQueueAdjustmentRebillSchema.nullish().transform((value) => value ?? null),
  memo: billingQueueAdjustmentMemoSchema.nullish().transform((value) => value ?? null),
});
export type BillingQueueAdjustmentContext = z.infer<typeof billingQueueAdjustmentContextSchema>;

/** A quarter turn a biller gave one page of a packet document, kept by document and page. */
export const billingQueuePacketPageRotationSchema = z.object({
  documentId: z.string(),
  page: z.number().int().min(1),
  rotation: z.number().int(),
});
export type BillingQueuePacketPageRotation = z.infer<typeof billingQueuePacketPageRotationSchema>;

export const billingQueueItemSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  /** Absent on a credit or debit memo's item, which bills a customer rather than a shipment. */
  shipmentId: nullableStringSchema,
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
  adjustmentContext: billingQueueAdjustmentContextSchema.default({ rebill: null, memo: null }),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  shipment: linkedRecordSchema("shipment", shipmentSchema).optional(),
  billToCustomer: customerReferenceSchema.optional().nullable(),
  assignedBiller: linkedRecordSchema("user", userSchema).optional().nullable(),
  canceledBy: linkedRecordSchema("user", userSchema).optional().nullable(),
  payerShare: payerShareSchema.nullish(),
  ratedBy: billingQueueRatedBySchema.nullish(),
  detentionHolds: z
    .array(detentionHoldSchema)
    .nullish()
    .transform((value) => value ?? []),
  ratingReviewHold: ratingReviewHoldSchema.nullish().transform((value) => value ?? null),
  holdReasonCode: nullableEnumSchema(billingQueueHoldReasonSchema),
  heldAt: z.number().nullish(),
  heldById: nullableStringSchema,
  statusBeforeHold: nullableEnumSchema(billingQueueStatusSchema),
  lane: billingQueueLaneSchema.default("Checking"),
  checkedAt: z.number().nullish(),
  poNumber: z.string().nullish(),
  invoiceMemo: z.string().nullish(),
  paymentTerm: z.string().nullish(),
  invoiceDate: z.number().nullish(),
  packetDocumentIds: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  /** The biller has picked the packet, so it is exactly packetDocumentIds, even when empty. */
  packetPicked: z
    .boolean()
    .nullish()
    .transform((value) => value ?? false),
  packetPageRotations: z
    .array(billingQueuePacketPageRotationSchema)
    .nullish()
    .transform((value) => value ?? []),
  agentVerdict: nullableEnumSchema(agentVerdictSchema),
  agentSummary: z.string().nullish(),
  agentReviewedAt: z.number().nullish(),
  agentReads: z
    .array(agentReadSchema)
    .nullish()
    .transform((value) => value ?? []),
  headline: billingQueueHeadlineSchema.nullish(),
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

export const bulkRunActionSchema = z.enum([
  "Approve",
  "Post",
  "Hold",
  "Assign",
  "Cancel",
  "RequestPaperwork",
  "ResolveIssue",
]);
export type BulkRunAction = z.infer<typeof bulkRunActionSchema>;

export type BulkRunParams = {
  assignApprover?: boolean;
  billerId?: string;
  holdReasonCode?: BillingQueueHoldReason;
  reason?: string;
  issueCode?: string;
  optionKey?: string;
};

export const billingQueueApprovalRunItemSchema = z.object({
  itemId: z.string(),
  status: z.enum(["Pending", "Succeeded", "Failed", "Skipped"]),
  failureCode: z.string().nullish(),
  errorMessage: z.string().nullish(),
  invoiceNumber: z.string().nullish(),
});

/** One press of Approve over several items, run as a job behind an undo window. */
export const billingQueueApprovalRunSchema = z.object({
  id: z.string(),
  action: bulkRunActionSchema.default("Approve"),
  status: z.enum(["Scheduled", "Running", "Completed", "Undone", "Failed"]),
  totalCount: z.number(),
  succeededCount: z.number().default(0),
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

export const podSignatureRedetectResultSchema = z.object({
  shipments: z.number(),
  documents: z.number(),
  queued: z.number(),
  alreadyRead: z.number(),
  setByPerson: z.number(),
  notReadByMachine: z.number(),
  deferred: z.number(),
  failed: z.number(),
});

export type PODSignatureRedetectResult = z.infer<typeof podSignatureRedetectResultSchema>;

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

export const billingQueueLaneStatSchema = z.object({
  lane: billingQueueLaneSchema,
  count: z.number(),
  total: decimalStringSchema,
});
export type BillingQueueLaneStat = z.infer<typeof billingQueueLaneStatSchema>;

/** The day's figures for the queue, computed against the caller's day. */
export const billingQueuePulseSchema = z.object({
  oldestWaitingAt: z.number().nullish(),
  deliveryToInvoiceDays: decimalStringSchema.nullish(),
  priorDeliveryToInvoiceDays: decimalStringSchema.nullish(),
  approvedTodayCount: z.number().default(0),
  approvedTodayTotal: decimalStringSchema.default(0),
  medianReviewSecondsToday: z.number().nullish(),
});
export type BillingQueuePulse = z.infer<typeof billingQueuePulseSchema>;

export const billingQueueStatsSchema = z.object({
  lanes: z
    .array(billingQueueLaneStatSchema)
    .nullish()
    .transform((value) => value ?? []),
  openCount: z.number().default(0),
  openTotal: decimalStringSchema.default(0),
  pulse: billingQueuePulseSchema.nullish(),
});
export type BillingQueueStats = z.infer<typeof billingQueueStatsSchema>;

/** One part of a packet: the invoice, or a document it carries, and the pages it fills. */
export const billingQueuePacketPartSchema = z.object({
  kind: z.enum(["Invoice", "Document"]),
  documentId: z.string().nullish(),
  name: z.string(),
  firstPage: z.number().int().min(1),
  lastPage: z.number().int().min(1),
  unsigned: z.boolean(),
});
export type BillingQueuePacketPart = z.infer<typeof billingQueuePacketPartSchema>;

/** The packet approving an item would send, assembled for viewing a page at a time. */
export const billingQueuePacketPreviewSchema = z.object({
  fileName: z.string(),
  pageCount: z.number().int().min(1),
  documents: z.number().int().min(0),
  build: z.string().min(1),
  parts: z
    .array(billingQueuePacketPartSchema)
    .nullish()
    .transform((value) => value ?? []),
});
export type BillingQueuePacketPreview = z.infer<typeof billingQueuePacketPreviewSchema>;

/** The checks one assembly of a packet passed: which of its document pages look blank. */
export const billingQueuePacketChecksSchema = z.object({
  build: z.string().min(1),
  blankPages: z
    .array(z.number().int().min(1))
    .nullish()
    .transform((value) => value ?? []),
});
export type BillingQueuePacketChecks = z.infer<typeof billingQueuePacketChecksSchema>;

export const recommendationKindSchema = z.enum([
  "ApproveReady",
  "AssignToMe",
  "PostApproved",
  "RequestPaperwork",
  "ReviewUnbilled",
  "HoldDuplicates",
  "ApproveAgentAgreed",
  "ReviewAgentConcerns",
]);
export type RecommendationKind = z.infer<typeof recommendationKindSchema>;

/** A batch the queue can clear in one action, naming exactly the items it covers. */
export const billingQueueRecommendationSchema = z.object({
  kind: recommendationKindSchema,
  source: z.enum(["Rules", "Agent"]),
  count: z.number(),
  total: decimalStringSchema,
  itemIds: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
});
export type BillingQueueRecommendation = z.infer<typeof billingQueueRecommendationSchema>;

export const billingQueueAssistSchema = z.object({
  review: z.boolean().default(false),
  brief: z.boolean().default(false),
  search: z.boolean().default(false),
});
export type BillingQueueAssist = z.infer<typeof billingQueueAssistSchema>;

export const billingQueueBriefSchema = z.object({
  available: z.boolean().default(false),
  text: z.string().default(""),
  generatedAt: z.number().default(0),
  /** Written for a queue that has since moved; a rewrite is on its way. */
  stale: z.boolean().default(false),
});
export type BillingQueueBrief = z.infer<typeof billingQueueBriefSchema>;

/**
 * Nova writing, in bulk, the reviews the queue is missing and then its brief;
 * null when nothing is generating. One runs per organization, shared by everyone
 * who has the queue open.
 */
export const billingQueueNovaGenerationSchema = z
  .object({
    phase: z.enum(["Reviewing", "Briefing", "Done", "Left"]),
    total: z.number().default(0),
    done: z.number().default(0),
    /** When opening the queue next tries what the last generation left ("Left"). */
    retryAt: z.number().nullish(),
  })
  .nullable();
export type BillingQueueNovaGeneration = z.infer<typeof billingQueueNovaGenerationSchema>;

/** The filters Nova read out of a sentence typed into the queue's search. */
/**
 * A window of days a queue search names: the last N days, or a range of calendar days
 * (YYYY-MM-DD, both ends kept) in the person's own calendar.
 */
export const billingQueueSearchDatesSchema = z.object({
  lastDays: z.number().int().positive().nullish(),
  from: z.string().nullish(),
  to: z.string().nullish(),
});
export type BillingQueueSearchDates = z.infer<typeof billingQueueSearchDatesSchema>;

export const billingQueueSearchIntentSchema = z.object({
  lanes: z
    .array(billingQueueLaneSchema)
    .nullish()
    .transform((value) => value ?? []),
  search: z.string().default(""),
  payerId: nullableStringSchema,
  payerName: z.string().nullish(),
  shipmentCustomerId: nullableStringSchema,
  shipmentCustomerName: z.string().nullish(),
  minAmount: decimalStringSchema.nullish(),
  maxAmount: decimalStringSchema.nullish(),
  createdBefore: z.number().nullish(),
  delivered: billingQueueSearchDatesSchema.nullish(),
  pickedUp: billingQueueSearchDatesSchema.nullish(),
  queued: billingQueueSearchDatesSchema.nullish(),
  unresolved: z.string().nullish(),
});
export type BillingQueueSearchIntent = z.infer<typeof billingQueueSearchIntentSchema>;

export type BillingQueueInvoiceDetailsInput = {
  version: number;
  poNumber: string;
  invoiceMemo: string;
  paymentTerm: string;
  invoiceDate: number | null;
  packetDocumentIds: string[];
  packetPicked: boolean;
  packetPageRotations: BillingQueuePacketPageRotation[];
};

export const billingQueueUpdateChargesSchema = z.object({
  formulaTemplateId: z.string().optional(),
  /** The tariff's input values for the shipment, replacing what it carries. */
  tariffInputs: tariffFormInputValuesSchema.optional(),
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
  /** Answers with the saved item as `getById` expands it, so the caller can show it. */
  expandShipmentDetails: z.boolean().optional(),
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
