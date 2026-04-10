import { z } from "zod";
import { decimalStringSchema, nullableStringSchema, optionalStringSchema } from "./helpers";
import { shipmentSchema } from "./shipment";
import { userSchema } from "./user";

export const billingQueueStatusSchema = z.enum([
  "ReadyForReview",
  "InReview",
  "Approved",
  "Posted",
  "OnHold",
  "SentBackToOps",
  "Exception",
  "Canceled",
]);
export type BillingQueueStatus = z.infer<typeof billingQueueStatusSchema>;

export const billTypeSchema = z.enum(["Invoice", "CreditMemo", "DebitMemo"]);
export type BillType = z.infer<typeof billTypeSchema>;

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

export const billingQueueItemSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  shipmentId: z.string(),
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
  assignedBiller: userSchema.optional().nullable(),
  canceledBy: userSchema.optional().nullable(),
});

export type BillingQueueItem = z.infer<typeof billingQueueItemSchema>;

export const billingQueueTransferSchema = z.object({
  shipmentId: z.string(),
  billType: billTypeSchema.optional(),
});
export type BillingQueueTransferInput = z.infer<
  typeof billingQueueTransferSchema
>;

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
export type BillingQueueUpdateStatusInput = z.infer<
  typeof billingQueueUpdateStatusSchema
>;

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
});
export type BillingQueueUpdateChargesInput = z.infer<
  typeof billingQueueUpdateChargesSchema
>;

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
export type BillingQueueFilterPreset = z.infer<
  typeof billingQueueFilterPresetSchema
>;

export const billingQueueFilterPresetInputSchema = z.object({
  name: z.string().min(1).max(100),
  filters: z.record(z.string(), z.any()),
  isDefault: z.boolean().optional(),
});
export type BillingQueueFilterPresetInput = z.infer<
  typeof billingQueueFilterPresetInputSchema
>;
