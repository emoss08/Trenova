import {
  BillingExceptionHandling,
  PaymentTerm,
  TransferSchedule,
} from "@/types/billing";
import { z } from "zod";

export const billingControlSchema = z
  .object({
    id: z.string(),
    version: z.number().optional(),
    createdAt: z.number().optional(),
    updatedAt: z.number().optional(),

    // * Core Fields
    creditMemoNumberPrefix: z
      .string({
        required_error: "Credit memo number prefix is required",
      })
      .min(3, "Credit memo number prefix must be at least 3 characters")
      .max(10, "Credit memo number prefix must be less than 10 characters"),
    invoiceNumberPrefix: z
      .string({
        required_error: "Invoice number prefix is required",
      })
      .min(3, "Invoice number prefix must be at least 3 characters")
      .max(10, "Invoice number prefix must be less than 10 characters"),
    // * Invoice terms
    paymentTerm: z.nativeEnum(PaymentTerm),
    showInvoiceDueDate: z.boolean(),
    invoiceTerms: z.string().optional(),
    invoiceFooter: z.string().optional(),
    showAmountDue: z.boolean(),
    autoTransfer: z.boolean(),
    transferSchedule: z.nativeEnum(TransferSchedule),
    transferBatchSize: z.preprocess((val) => {
      if (val === "" || val === null || val === undefined) {
        return undefined;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? undefined : parsed;
    }, z.number().optional()),
    autoMarkReadyToBill: z.boolean(),
    enforceCustomerBillingReq: z.boolean(),
    validateCustomerRates: z.boolean(),
    autoBill: z.boolean(),
    sendAutoBillNotifications: z.boolean(),
    autoBillBatchSize: z.preprocess((val) => {
      if (val === "" || val === null || val === undefined) {
        return undefined;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? undefined : parsed;
    }, z.number().optional()),
    billingExceptionHandling: z.nativeEnum(BillingExceptionHandling),
    rateDiscrepancyThreshold: z.preprocess((val) => {
      if (val === "" || val === null || val === undefined) {
        return undefined;
      }
      const parsed = parseFloat(String(val));
      return isNaN(parsed) ? undefined : parsed;
    }, z.number().optional()),
    autoResolveMinorDiscrepancies: z.boolean(),
    allowInvoiceConsolidation: z.boolean(),
    consolidationPeriodDays: z.preprocess((val) => {
      if (val === "" || val === null || val === undefined) {
        return undefined;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? undefined : parsed;
    }, z.number().optional()),
    groupConsolidatedInvoices: z.boolean(),
  })
  .refine(
    (data) => {
      // If allowInvoiceConsolidation is true, consolidationPeriodDays must be provided
      if (data.allowInvoiceConsolidation && !data.consolidationPeriodDays) {
        return false;
      }
      return true;
    },
    {
      message:
        "Consolidation period days is required when invoice consolidation is allowed",
      path: ["consolidationPeriodDays"],
    },
  );

export type BillingControlSchema = z.infer<typeof billingControlSchema>;
