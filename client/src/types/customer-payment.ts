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
