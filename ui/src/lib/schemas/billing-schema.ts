import {
  AutoBillCriteria,
  BillingExceptionHandling,
  PaymentTerm,
  TransferCriteria,
  TransferSchedule,
} from "@/types/billing";
import { z } from "zod";

export const billingControlSchema = z
  .object({
    id: z.string().optional(),
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
    transferCriteria: z.nativeEnum(TransferCriteria),
    transferSchedule: z.nativeEnum(TransferSchedule),
    transferBatchSize: z
      .number()
      .min(1, "Transfer batch size must be greater than 0"),
    autoMarkReadyToBill: z.boolean(),
    enforceCustomerBillingReq: z.boolean(),
    validateCustomerRates: z.boolean(),
    autoBill: z.boolean(),
    autoBillCriteria: z.nativeEnum(AutoBillCriteria).optional(),
    sendAutoBillNotifications: z.boolean(),
    autoBillBatchSize: z
      .number()
      .min(1, "Auto bill batch size must be greater than 0"),
    billingExceptionHandling: z.nativeEnum(BillingExceptionHandling),
    rateDiscrepancyThreshold: z
      .number()
      .min(0, "Rate discrepancy threshold must be greater than 0"),
    autoResolveMinorDiscrepancies: z.boolean(),
    allowInvoiceConsolidation: z.boolean(),
    consolidationPeriodDays: z
      .number()
      .min(1, "Consolidation period days must be greater than 0")
      .optional(),
    groupConsolidatedInvoices: z.boolean(),
  })
  .refine(
    (data) => {
      // * If autoBill is true, autoBillCriteria must be provided
      if (data.autoBill && !data.autoBillCriteria) {
        return false;
      }
      return true;
    },
    {
      message: "Auto bill criteria is required when auto bill is enabled",
      path: ["autoBillCriteria"],
    },
  )
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
