import { z } from "zod";
import { billTypeSchema, billingQueueItemSchema } from "./billing-queue";
import { customerPaymentTermSchema, customerSchema } from "./customer";
import {
  decimalStringSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  tenantInfoSchema,
} from "./helpers";
import { shipmentSchema } from "./shipment";

export const invoiceStatusSchema = z.enum(["Draft", "Posted"]);
export type InvoiceStatus = z.infer<typeof invoiceStatusSchema>;

export const invoiceLineTypeSchema = z.enum(["Freight", "Accessorial"]);
export type InvoiceLineType = z.infer<typeof invoiceLineTypeSchema>;

export const invoiceLineSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  invoiceId: z.string(),
  lineNumber: z.number().int(),
  type: invoiceLineTypeSchema,
  description: z.string(),
  quantity: decimalStringSchema,
  unitPrice: decimalStringSchema,
  amount: decimalStringSchema,
});
export type InvoiceLine = z.infer<typeof invoiceLineSchema>;

export const invoiceSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  billingQueueItemId: z.string(),
  shipmentId: z.string(),
  customerId: z.string(),
  number: z.string(),
  billType: billTypeSchema,
  status: invoiceStatusSchema,
  paymentTerm: customerPaymentTermSchema,
  currencyCode: z.string(),
  invoiceDate: z.number(),
  dueDate: nullableIntegerSchema,
  postedAt: nullableIntegerSchema,
  shipmentProNumber: z.string().optional().nullable(),
  shipmentBol: z.string().optional().nullable(),
  serviceDate: nullableIntegerSchema,
  billToName: z.string(),
  billToCode: nullableStringSchema,
  billToAddressLine1: nullableStringSchema,
  billToAddressLine2: nullableStringSchema,
  billToCity: nullableStringSchema,
  billToState: nullableStringSchema,
  billToPostalCode: nullableStringSchema,
  billToCountry: nullableStringSchema,
  subtotalAmount: decimalStringSchema,
  otherAmount: decimalStringSchema,
  totalAmount: decimalStringSchema,
  appliedAmount: decimalStringSchema,
  settlementStatus: z.enum(["Unpaid", "PartiallyPaid", "Paid"]),
  disputeStatus: z.enum(["None", "Disputed"]),
  correctionGroupId: nullableStringSchema,
  supersedesInvoiceId: nullableStringSchema,
  supersededByInvoiceId: nullableStringSchema,
  sourceInvoiceAdjustmentId: nullableStringSchema,
  isAdjustmentArtifact: z.boolean().default(false),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  billingQueueItem: billingQueueItemSchema.optional(),
  shipment: shipmentSchema.optional(),
  customer: customerSchema.optional(),
  lines: z.array(invoiceLineSchema).optional().default([]),
});
export type Invoice = z.infer<typeof invoiceSchema>;
