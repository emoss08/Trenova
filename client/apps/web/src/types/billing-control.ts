import { z } from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

const decimalNumberSchema = z.coerce.number().finite();

export const transferScheduleSchema = z.enum([
  "Continuous",
  "Hourly",
  "Daily",
  "Weekly",
]);
export type TransferSchedule = z.infer<typeof transferScheduleSchema>;

export const paymentTermSchema = z.enum([
  "Net10",
  "Net15",
  "Net30",
  "Net45",
  "Net60",
  "Net90",
  "DueOnReceipt",
]);
export type PaymentTerm = z.infer<typeof paymentTermSchema>;

export const enforcementLevelSchema = z.enum([
  "Ignore",
  "Warn",
  "RequireReview",
  "Block",
]);
export type EnforcementLevel = z.infer<typeof enforcementLevelSchema>;

export const billingExceptionDispositionSchema = z.enum([
  "RouteToBillingReview",
  "ReturnToOperations",
]);
export type BillingExceptionDisposition = z.infer<typeof billingExceptionDispositionSchema>;

export const readyToBillAssignmentModeSchema = z.enum([
  "ManualOnly",
  "AutomaticWhenEligible",
]);
export type ReadyToBillAssignmentMode = z.infer<typeof readyToBillAssignmentModeSchema>;

export const billingQueueTransferModeSchema = z.enum([
  "ManualOnly",
  "AutomaticWhenReady",
]);
export type BillingQueueTransferMode = z.infer<typeof billingQueueTransferModeSchema>;

export const invoiceDraftCreationModeSchema = z.enum([
  "ManualOnly",
  "AutomaticWhenTransferred",
]);
export type InvoiceDraftCreationMode = z.infer<typeof invoiceDraftCreationModeSchema>;

export const invoicePostingModeSchema = z.enum([
  "ManualReviewRequired",
  "AutomaticWhenNoBlockingExceptions",
]);
export type InvoicePostingMode = z.infer<typeof invoicePostingModeSchema>;

export const rateVarianceAutoResolutionModeSchema = z.enum([
  "Disabled",
  "BypassReviewWithinTolerance",
]);
export type RateVarianceAutoResolutionMode = z.infer<typeof rateVarianceAutoResolutionModeSchema>;

export const billingControlSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  defaultPaymentTerm: paymentTermSchema,
  defaultInvoiceTerms: z.string().optional().default(""),
  defaultInvoiceFooter: z.string().optional().default(""),
  showDueDateOnInvoice: z.boolean(),
  showBalanceDueOnInvoice: z.boolean(),

  readyToBillAssignmentMode: readyToBillAssignmentModeSchema,
  billingQueueTransferMode: billingQueueTransferModeSchema,
  billingQueueTransferSchedule: transferScheduleSchema.nullish(),
  billingQueueTransferBatchSize: z.coerce.number().int().nullish(),

  invoiceDraftCreationMode: invoiceDraftCreationModeSchema,
  invoicePostingMode: invoicePostingModeSchema,
  autoInvoiceBatchSize: z.coerce.number().int().nullish(),
  notifyOnAutoInvoiceCreation: z.boolean(),

  shipmentBillingRequirementEnforcement: enforcementLevelSchema,
  rateValidationEnforcement: enforcementLevelSchema,
  billingExceptionDisposition: billingExceptionDispositionSchema,
  notifyOnBillingExceptions: z.boolean(),

  rateVarianceTolerancePercent: decimalNumberSchema,
  rateVarianceAutoResolutionMode: rateVarianceAutoResolutionModeSchema,
});

export type BillingControl = z.infer<typeof billingControlSchema>;
