import {
  AutoBillCriteria,
  BillingExceptionHandling,
  TransferCriteria,
} from "@/types/billing";
import { boolean, InferType, mixed, number, object, string } from "yup";

export const billingControlSchema = object({
  // id: string().
  invoiceDueAfterDays: number()
    .min(1, "Invoice due after days must be greater than 0")
    .required("Invoice due after days is required"),
  creditMemoNumberPrefix: string()
    .min(3, "Credit memo number prefix must be at least 3 characters")
    .max(10, "Credit memo number prefix must be less than 10 characters")
    .required("Credit memo number prefix is required"),
  invoiceNumberPrefix: string()
    .min(3, "Invoice number prefix must be at least 3 characters")
    .max(10, "Invoice number prefix must be less than 10 characters")
    .required("Invoice number prefix is required"),
  autoBill: boolean().optional().notRequired(),
  autoBillCriteria: mixed<AutoBillCriteria>()
    .oneOf(Object.values(AutoBillCriteria))
    .when("autoBill", {
      is: true,
      then: (schema) => schema.required("Auto bill criteria is required"),
      otherwise: (schema) => schema.optional().notRequired(),
    }),
  transferCriteria: mixed<TransferCriteria>().oneOf(
    Object.values(TransferCriteria),
  ),
  billingExceptionHandling: mixed<BillingExceptionHandling>().oneOf(
    Object.values(BillingExceptionHandling),
  ),
  rateDiscrepancyThreshold: number()
    .min(0, "Rate discrepancy threshold must be greater than 0")
    .required("Rate discrepancy threshold is required"),
  allowInvoiceConsolidation: boolean().optional().notRequired(),
  consolidationPeriodDays: number()
    .min(1, "Consolidation period days must be greater than 0")
    .when("allowInvoiceConsolidation", {
      is: true,
      then: (schema) =>
        schema.required("Consolidation period days is required"),
      otherwise: (schema) => schema.optional().notRequired(),
    }),
});

export type BillingControlSchema = InferType<typeof billingControlSchema>;
