import {
  AutoBillCriteria,
  BillingExceptionHandling,
  PaymentTerm,
  TransferCriteria,
  TransferSchedule,
} from "@/types/billing";
import { boolean, InferType, mixed, number, object, string } from "yup";

export const billingControlSchema = object({
  creditMemoNumberPrefix: string()
    .min(3, "Credit memo number prefix must be at least 3 characters")
    .max(10, "Credit memo number prefix must be less than 10 characters")
    .required("Credit memo number prefix is required"),
  invoiceNumberPrefix: string()
    .min(3, "Invoice number prefix must be at least 3 characters")
    .max(10, "Invoice number prefix must be less than 10 characters")
    .required("Invoice number prefix is required"),
  // * Invoice terms
  paymentTerm: mixed<PaymentTerm>().oneOf(Object.values(PaymentTerm)),
  showInvoiceDueDate: boolean(),
  invoiceTerms: string().optional().notRequired(),
  invoiceFooter: string().optional().notRequired(),
  showAmountDue: boolean(),
  // * Controls for the billing process
  autoTransfer: boolean(),
  transferCriteria: mixed<TransferCriteria>().oneOf(
    Object.values(TransferCriteria),
  ),
  transferSchedule: mixed<TransferSchedule>().oneOf(
    Object.values(TransferSchedule),
  ),
  transferBatchSize: number()
    .min(1, "Transfer batch size must be greater than 0")
    .required("Transfer batch size is required"),
  autoMarkReadyToBill: boolean(),
  // * Enforce customer billing requirements before billing
  enforceCustomerBillingReq: boolean(),
  validateCustomerRates: boolean(),
  // * Automated billing controls
  autoBill: boolean(),
  autoBillCriteria: mixed<AutoBillCriteria>()
    .oneOf(Object.values(AutoBillCriteria))
    .when("autoBill", {
      is: true,
      then: (schema) => schema.required("Auto bill criteria is required"),
      otherwise: (schema) => schema.optional().notRequired(),
    }),
  sendAutoBillNotifications: boolean(),
  autoBillBatchSize: number()
    .min(1, "Auto bill batch size must be greater than 0")
    .required("Auto bill batch size is required"),
  // * Exception handling
  billingExceptionHandling: mixed<BillingExceptionHandling>().oneOf(
    Object.values(BillingExceptionHandling),
  ),
  rateDiscrepancyThreshold: number()
    .min(0, "Rate discrepancy threshold must be greater than 0")
    .required("Rate discrepancy threshold is required"),
  autoResolveMinorDiscrepancies: boolean(),
  // * Consolidation options
  allowInvoiceConsolidation: boolean(),
  consolidationPeriodDays: number()
    .min(1, "Consolidation period days must be greater than 0")
    .when("allowInvoiceConsolidation", {
      is: true,
      then: (schema) =>
        schema.required("Consolidation period days is required"),
      otherwise: (schema) => schema.optional().notRequired(),
    }),
  groupConsolidatedInvoices: boolean(),
});

export type BillingControlSchema = InferType<typeof billingControlSchema>;
