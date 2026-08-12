import { z } from "zod";
import { optionalStringSchema } from "./helpers";

export const carrierSettlementStatusSchema = z.enum([
  "Draft",
  "PendingApproval",
  "Approved",
  "Posted",
  "Paid",
  "Voided",
]);
export type CarrierSettlementStatus = z.infer<typeof carrierSettlementStatusSchema>;

export const carrierCostEventTypeSchema = z.enum([
  "LinehaulCost",
  "FuelSurcharge",
  "Accessorial",
  "Adjustment",
]);
export type CarrierCostEventType = z.infer<typeof carrierCostEventTypeSchema>;

export const carrierCostEventStatusSchema = z.enum(["Pending", "Attached", "Settled", "Voided"]);
export type CarrierCostEventStatus = z.infer<typeof carrierCostEventStatusSchema>;

export const carrierSettlementBatchStatusSchema = z.enum(["Open", "Completed", "Canceled"]);
export type CarrierSettlementBatchStatus = z.infer<typeof carrierSettlementBatchStatusSchema>;

export const carrierLedgerEntryTypeSchema = z.enum(["Bill", "Payment", "Adjustment"]);
export type CarrierLedgerEntryType = z.infer<typeof carrierLedgerEntryTypeSchema>;

export const carrierInvoiceMatchStatusSchema = z.enum([
  "Suggested",
  "Matched",
  "Variance",
  "Resolved",
  "Rejected",
]);
export type CarrierInvoiceMatchStatus = z.infer<typeof carrierInvoiceMatchStatusSchema>;

export const carrierInvoiceMatchViaSchema = z.enum(["Manual", "Auto"]);
export type CarrierInvoiceMatchVia = z.infer<typeof carrierInvoiceMatchViaSchema>;

export const generateCarrierBatchFormSchema = z.object({
  name: optionalStringSchema,
  notes: optionalStringSchema,
});
export type GenerateCarrierBatchFormValues = z.infer<typeof generateCarrierBatchFormSchema>;

export const carrierSettlementControlFormSchema = z.object({
  payTrigger: z.enum(["MoveCompleted", "ShipmentDelivered", "PODReceived", "ShipmentInvoiced"]),
  payPeriodFrequency: z.enum(["Weekly", "Biweekly", "Monthly"]),
  periodEndDayOfWeek: z.number().int().min(0).max(6),
  payDelayDays: z.number().int().min(0).max(60),
  autoGenerateBatches: z.boolean(),
  autoPostOnApprove: z.boolean(),
  varianceTolerance: z.number().min(0, "Variance tolerance cannot be negative"),
  autoMatchInboundInvoices: z.boolean(),
  autoAcceptWithinTolerance: z.boolean(),
  defaultApAccountId: z.string().optional().nullable(),
  defaultPurchasedTransportationAccountId: z.string().optional().nullable(),
});
export type CarrierSettlementControlFormValues = z.infer<typeof carrierSettlementControlFormSchema>;
