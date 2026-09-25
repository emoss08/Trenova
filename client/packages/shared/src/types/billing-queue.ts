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
  detentionHolds: z.array(detentionHoldSchema).default([]),
});

export type BillingQueueItem = z.infer<typeof billingQueueItemSchema>;

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
