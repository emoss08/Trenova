import { z } from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const transferScheduleSchema = z.enum([
  "Continuous",
  "Hourly",
  "Daily",
  "Weekly",
]);

export type TransferSchedule = z.infer<typeof transferScheduleSchema>;

export const exceptionHandlingSchema = z.enum([
  "Queue",
  "Notify",
  "AutoResolve",
  "Reject",
]);

export type ExceptionHandling = z.infer<typeof exceptionHandlingSchema>;

export const paymentTermSchema = z.enum([
  "Net15",
  "Net30",
  "Net45",
  "Net60",
  "Net90",
  "DueOnReceipt",
]);

export type PaymentTerm = z.infer<typeof paymentTermSchema>;

export const billingControlSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  invoiceNumberPrefix: z
    .string()
    .min(3, {
      message: "Invoice number prefix must be between 3 and 10 characters",
    })
    .max(10, {
      message: "Invoice number prefix must be between 3 and 10 characters",
    }),
  creditMemoNumberPrefix: z
    .string()
    .min(3, {
      message: "Credit memo number prefix must be between 3 and 10 characters",
    })
    .max(10, {
      message: "Credit memo number prefix must be between 3 and 10 characters",
    }),
  invoiceTerms: z.string().optional().default(""),
  invoiceFooter: z.string().optional().default(""),

  transferSchedule: transferScheduleSchema,
  billingExceptionHandling: exceptionHandlingSchema,
  paymentTerm: paymentTermSchema,

  showInvoiceDueDate: z.boolean(),
  showAmountDue: z.boolean(),
  autoTransfer: z.boolean(),
  autoMarkReadyToBill: z.boolean(),
  enforceCustomerBillingReq: z.boolean(),
  validateCustomerRates: z.boolean(),
  autoBill: z.boolean(),
  autoResolveMinorDiscrepancies: z.boolean(),
  allowInvoiceConsolidation: z.boolean(),
  groupConsolidatedInvoices: z.boolean(),
  sendAutoBillNotifications: z.boolean(),

  transferBatchSize: z
    .number()
    .int()
    .min(1, { message: "Transfer batch size must be greater than 0" }),
  autoBillBatchSize: z
    .number()
    .int()
    .min(1, { message: "Auto bill batch size must be greater than 0" }),
  consolidationPeriodDays: z
    .number()
    .int()
    .min(1, { message: "Consolidation period days must be greater than 0" }),
  rateDiscrepancyThreshold: z
    .number()
    .min(0, { message: "Rate discrepancy threshold must be at least 0" }),
});

export type BillingControl = z.infer<typeof billingControlSchema>;
