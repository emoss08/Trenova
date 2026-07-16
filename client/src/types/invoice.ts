import { z } from "zod";
import { billTypeSchema, billingQueueItemSchema } from "./billing-queue";
import { customerPaymentTermSchema, customerSchema } from "./customer";
import { documentSchema } from "./document";
import { emailMessageSchema } from "./email";
import {
  decimalStringSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  tenantInfoSchema,
} from "./helpers";
import { shipmentSchema } from "./shipment";

export const invoiceStatusSchema = z.enum(["Draft", "Posted"]);
export type InvoiceStatus = z.infer<typeof invoiceStatusSchema>;

export const invoiceSendStatusSchema = z.enum([
  "NotSent",
  "Sending",
  "Sent",
  "PartiallySent",
  "Failed",
]);
export type InvoiceSendStatus = z.infer<typeof invoiceSendStatusSchema>;

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

export const settlementStatusSchema = z.enum(["Unpaid", "PartiallyPaid", "Paid"]);
export type SettlementStatus = z.infer<typeof settlementStatusSchema>;

export const invoiceAttachmentSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  invoiceId: z.string(),
  documentId: z.string(),
  selected: z.boolean(),
  sortOrder: z.number().int(),
  createdAt: z.number(),
  updatedAt: z.number(),
  document: documentSchema.nullable().optional(),
});
export type InvoiceAttachment = z.infer<typeof invoiceAttachmentSchema>;

export const invoiceEmailAttemptAttachmentSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  attemptId: z.string(),
  documentId: z.string(),
  fileName: z.string(),
  contentType: z.string(),
  sizeBytes: z.number(),
  encodedBytes: z.number(),
  method: z.enum(["Attached", "Link", "Skipped", "Failed"]),
  shareTokenId: nullableStringSchema,
  reason: nullableStringSchema,
  createdAt: z.number(),
  document: documentSchema.nullable().optional(),
});
export type InvoiceEmailAttemptAttachment = z.infer<typeof invoiceEmailAttemptAttachmentSchema>;

export const invoiceEmailAttemptSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  invoiceId: z.string(),
  emailMessageId: nullableStringSchema,
  attemptNumber: z.number().int(),
  partNumber: z.number().int(),
  totalParts: z.number().int(),
  status: invoiceSendStatusSchema,
  provider: z.string().nullable().optional(),
  providerMessageId: nullableStringSchema,
  toRecipients: z.array(z.string()),
  ccRecipients: z.array(z.string()).nullable().optional(),
  bccRecipients: z.array(z.string()).nullable().optional(),
  subject: z.string(),
  body: nullableStringSchema,
  estimatedSize: z.number(),
  warnings: z.array(z.string()).nullable().optional(),
  error: nullableStringSchema,
  sentAt: nullableIntegerSchema,
  createdById: nullableStringSchema,
  createdAt: z.number(),
  updatedAt: z.number(),
  email: emailMessageSchema.nullish().transform((value) => value ?? undefined),
  attachments: z.array(invoiceEmailAttemptAttachmentSchema).optional().default([]),
});
export type InvoiceEmailAttempt = z.infer<typeof invoiceEmailAttemptSchema>;

export const invoiceSendPlanAttachmentSchema = z.object({
  documentId: z.string(),
  fileName: z.string(),
  contentType: z.string(),
  sizeBytes: z.number(),
  encodedBytes: z.number(),
  invoicePdf: z.boolean(),
});

export const invoiceSendPlanDocumentLinkSchema = z.object({
  documentId: z.string(),
  fileName: z.string(),
  sizeBytes: z.number(),
  reason: z.string(),
  url: z.string().optional(),
});

const stringArraySchema = z
  .array(z.string())
  .nullish()
  .transform((value) => value ?? []);

export const invoiceSendPlanPartSchema = z.object({
  partNumber: z.number().int(),
  estimatedSizeBytes: z.number(),
  attachments: z
    .array(invoiceSendPlanAttachmentSchema)
    .nullish()
    .transform((value) => value ?? []),
  links: z
    .array(invoiceSendPlanDocumentLinkSchema)
    .nullish()
    .transform((value) => value ?? []),
  warnings: stringArraySchema,
});

export const invoiceSendPlanSchema = z.object({
  invoiceId: z.string(),
  providerLimitBytes: z.number(),
  estimatedBodyBytes: z.number(),
  parts: z
    .array(invoiceSendPlanPartSchema)
    .nullish()
    .transform((value) => value ?? []),
  warnings: stringArraySchema,
  errors: stringArraySchema,
  recipients: z.object({
    to: stringArraySchema,
    cc: stringArraySchema,
    bcc: stringArraySchema,
  }),
  fromEmail: z.string().optional().default(""),
  headers: z
    .record(z.string(), z.string())
    .nullish()
    .transform((value) => value ?? {}),
  openTracking: z.boolean().optional().default(false),
  subject: z.string(),
  body: z.string(),
  invoicePdfDocumentId: nullableStringSchema,
});
export type InvoiceSendPlan = z.infer<typeof invoiceSendPlanSchema>;

export const invoiceSendResultSchema = z.object({
  invoice: z.lazy(() => invoiceSchema),
  plan: invoiceSendPlanSchema,
  attempts: z
    .array(invoiceEmailAttemptSchema)
    .nullish()
    .transform((value) => value ?? []),
});
export type InvoiceSendResult = z.infer<typeof invoiceSendResultSchema>;

export const generateInvoicePdfResultSchema = z.object({
  invoiceId: z.string(),
  workflowId: z.string(),
  workflowRunId: z.string(),
  status: z.literal("Queued"),
});
export type GenerateInvoicePdfResult = z.infer<typeof generateInvoicePdfResultSchema>;

export const invoiceSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  billingQueueItemId: z.string(),
  shipmentId: nullableStringSchema,
  orderId: nullableStringSchema,
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
  orderNumber: z.string().optional().nullable(),
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
  settlementStatus: settlementStatusSchema,
  disputeStatus: z.enum(["None", "Disputed"]),
  pdfDocumentId: nullableStringSchema,
  sendStatus: invoiceSendStatusSchema.default("NotSent"),
  sentAt: nullableIntegerSchema,
  sentById: nullableStringSchema,
  lastSendError: nullableStringSchema,
  lastSendWarning: nullableStringSchema,
  memo: nullableStringSchema,
  remittanceInstructions: nullableStringSchema,
  emailSubjectSnapshot: nullableStringSchema,
  emailBodySnapshot: nullableStringSchema,
  emailToSnapshot: z.array(z.string()).nullable().optional(),
  emailCcSnapshot: z.array(z.string()).nullable().optional(),
  emailBccSnapshot: z.array(z.string()).nullable().optional(),
  correctionGroupId: nullableStringSchema,
  supersedesInvoiceId: nullableStringSchema,
  supersededByInvoiceId: nullableStringSchema,
  sourceInvoiceAdjustmentId: nullableStringSchema,
  isAdjustmentArtifact: z.boolean().default(false),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  billingQueueItem: billingQueueItemSchema.nullish().transform((value) => value ?? undefined),
  shipment: shipmentSchema.nullish().transform((value) => value ?? undefined),
  customer: customerSchema.nullish().transform((value) => value ?? undefined),
  pdfDocument: documentSchema.nullish().transform((value) => value ?? undefined),
  lines: z
    .array(invoiceLineSchema)
    .nullish()
    .transform((value) => value ?? []),
  attachments: z
    .array(invoiceAttachmentSchema)
    .nullish()
    .transform((value) => value ?? []),
  emailAttempts: z
    .array(invoiceEmailAttemptSchema)
    .nullish()
    .transform((value) => value ?? []),
});
export type Invoice = z.infer<typeof invoiceSchema>;

export const updateInvoiceDraftSchema = z.object({
  memo: z.string().optional(),
  remittanceInstructions: z.string().optional(),
  emailSubject: z.string().optional(),
  emailBody: z.string().optional(),
  emailTo: z.array(z.string()).optional(),
  emailCc: z.array(z.string()).optional(),
  emailBcc: z.array(z.string()).optional(),
  attachmentIds: z.array(z.string()).optional(),
});
export type UpdateInvoiceDraft = z.infer<typeof updateInvoiceDraftSchema>;
