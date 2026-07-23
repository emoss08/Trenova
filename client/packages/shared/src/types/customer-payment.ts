import { z } from "zod";
import { nullableStringSchema, optionalStringSchema, tenantInfoSchema } from "./helpers";

export const customerPaymentStatusSchema = z.enum(["Posted", "Reversed"]);
export type CustomerPaymentStatus = z.infer<typeof customerPaymentStatusSchema>;

export const paymentMethodSchema = z.enum(["ACH", "Check", "Wire", "Card", "Cash", "Other"]);
export type PaymentMethod = z.infer<typeof paymentMethodSchema>;

export const paymentApplicationSchema = z.object({
  id: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  customerPaymentId: z.string(),
  invoiceId: z.string(),
  appliedAmountMinor: z.number().int(),
  shortPayAmountMinor: z.number().int(),
  lineNumber: z.number().int(),
  createdAt: z.number().int().optional(),
  updatedAt: z.number().int().optional(),
});
export type PaymentApplication = z.infer<typeof paymentApplicationSchema>;

export const customerPaymentSchema = z.object({
  ...tenantInfoSchema.shape,
  customerId: z.string(),
  paymentDate: z.number().int(),
  accountingDate: z.number().int(),
  amountMinor: z.number().int(),
  appliedAmountMinor: z.number().int(),
  unappliedAmountMinor: z.number().int(),
  status: customerPaymentStatusSchema,
  paymentMethod: paymentMethodSchema,
  referenceNumber: z.string(),
  memo: optionalStringSchema,
  currencyCode: z.string(),
  postedBatchId: nullableStringSchema,
  reversalBatchId: nullableStringSchema,
  reversedById: nullableStringSchema,
  reversedAt: z.number().int().nullish(),
  reversalReason: optionalStringSchema,
  createdById: optionalStringSchema,
  updatedById: optionalStringSchema,
  applications: z.array(paymentApplicationSchema).optional(),
});
export type CustomerPayment = z.infer<typeof customerPaymentSchema>;

export const cashApplicationRowSchema = z.object({
  invoiceId: z.string(),
  invoiceNumber: z.string(),
  invoiceDate: z.number().int(),
  dueDate: z.number().int(),
  daysPastDue: z.number().int(),
  openAmountMinor: z.number().int(),
  checked: z.boolean(),
  appliedAmount: z.number().min(0, "Applied amount cannot be negative"),
  shortPayAmount: z.number().min(0, "Short-pay amount cannot be negative"),
});
export type CashApplicationRow = z.infer<typeof cashApplicationRowSchema>;

export const recordPaymentSchema = z.object({
  customerId: z.string().min(1, "Customer is required"),
  paymentDate: z.number({ error: "Payment date is required" }).int().positive("Payment date is required"),
  accountingDate: z
    .number({ error: "Accounting date is required" })
    .int()
    .positive("Accounting date is required"),
  amount: z.number({ error: "Amount is required" }).positive("Amount must be greater than zero"),
  paymentMethod: paymentMethodSchema,
  referenceNumber: optionalStringSchema,
  memo: optionalStringSchema,
  applications: z.array(cashApplicationRowSchema),
});
export type RecordPaymentFormValues = z.infer<typeof recordPaymentSchema>;

export const applyUnappliedSchema = z.object({
  accountingDate: z
    .number({ error: "Accounting date is required" })
    .int()
    .positive("Accounting date is required"),
  applications: z.array(cashApplicationRowSchema),
});
export type ApplyUnappliedFormValues = z.infer<typeof applyUnappliedSchema>;
