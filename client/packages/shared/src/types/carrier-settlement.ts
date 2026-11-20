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
