/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import {
  BillingExceptionHandling,
  PaymentTerm,
  TransferSchedule,
} from "@/types/billing";
import * as z from "zod/v4";
import {
  decimalStringSchema,
  nullableIntegerSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const billingControlSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,

    // * Core Fields
    creditMemoNumberPrefix: z
      .string({
        error: "Credit memo number prefix is required",
      })
      .min(3, {
        error: "Credit memo number prefix must be at least 3 characters",
      })
      .max(10, {
        error: "Credit memo number prefix must be less than 10 characters",
      }),
    invoiceNumberPrefix: z
      .string({
        error: "Invoice number prefix is required",
      })
      .min(3, { error: "Invoice number prefix must be at least 3 characters" })
      .max(10, {
        error: "Invoice number prefix must be less than 10 characters",
      }),
    // * Invoice terms
    paymentTerm: z.enum(PaymentTerm),
    showInvoiceDueDate: z.boolean(),
    invoiceTerms: z.string().optional(),
    invoiceFooter: z.string().optional(),
    showAmountDue: z.boolean(),
    autoTransfer: z.boolean(),
    transferSchedule: z.enum(TransferSchedule),
    transferBatchSize: nullableIntegerSchema,
    autoMarkReadyToBill: z.boolean(),
    enforceCustomerBillingReq: z.boolean(),
    validateCustomerRates: z.boolean(),
    autoBill: z.boolean(),
    sendAutoBillNotifications: z.boolean(),
    autoBillBatchSize: nullableIntegerSchema,
    billingExceptionHandling: z.enum(BillingExceptionHandling),
    rateDiscrepancyThreshold: decimalStringSchema,
    autoResolveMinorDiscrepancies: z.boolean(),
    allowInvoiceConsolidation: z.boolean(),
    consolidationPeriodDays: nullableIntegerSchema,
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
