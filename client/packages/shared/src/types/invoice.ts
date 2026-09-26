import { z } from "zod";
import { accessorialChargeMethodSchema, rateUnitSchema } from "./accessorial-charge";
import { billTypeSchema, billingQueueItemSchema } from "./billing-queue";
import { customerPaymentTermSchema, customerSchema } from "./customer";
import { documentSchema } from "./document";
import { emailMessageSchema } from "./email";
import {
  decimalStringSchema,
  nullableEnumSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  tenantInfoSchema,
} from "./helpers";
import { invoiceDetailSchema, invoiceSectionKeySchema } from "./customer";
import { shipmentSchema } from "./shipment";

export const invoiceStatusSchema = z.enum(["Draft", "Posted", "Voided"]);
export type InvoiceStatus = z.infer<typeof invoiceStatusSchema>;

/**
 * What an invoice covers, stamped by the server when it is created. Read this
 * rather than inferring shape from which of shipmentId / orderId is set.
 */
export const invoiceScopeSchema = z.enum([
  "Shipment",
  "Order",
  "Consolidated",
  "Adjustment",
  "Memo",
]);
export type InvoiceScope = z.infer<typeof invoiceScopeSchema>;

export const invoiceSendStatusSchema = z.enum([
  "NotSent",
  "Sending",
  "Sent",
  "PartiallySent",
  "Failed",
]);
export type InvoiceSendStatus = z.infer<typeof invoiceSendStatusSchema>;

/** What happens to the freight behind a voided invoice. */
export const invoiceVoidDispositionSchema = z.enum(["Rebill", "DoNotRebill"]);
export type InvoiceVoidDisposition = z.infer<typeof invoiceVoidDispositionSchema>;

export const invoiceMemoKindSchema = z.enum(["Manual", "LateCharge"]);
export type InvoiceMemoKind = z.infer<typeof invoiceMemoKindSchema>;

/** Where the outbound EDI 210 for an invoice stands, separately from email. */
export const invoiceEdiSendStatusSchema = z.enum([
  "NotSent",
  "NotConfigured",
  "Queued",
  "Generated",
  "Sending",
  "Sent",
  "Failed",
  "DeadLettered",
]);
export type InvoiceEdiSendStatus = z.infer<typeof invoiceEdiSendStatusSchema>;

export const invoiceDisputeCaseStatusSchema = z.enum(["Open", "Resolved", "Withdrawn"]);
export type InvoiceDisputeCaseStatus = z.infer<typeof invoiceDisputeCaseStatusSchema>;

export const invoiceDisputeReasonCodeSchema = z.enum([
  "RateDiscrepancy",
  "AccessorialDisputed",
  "ServiceFailure",
  "DuplicateBilling",
  "WrongBillTo",
  "MissingDocumentation",
  "Other",
]);
export type InvoiceDisputeReasonCode = z.infer<typeof invoiceDisputeReasonCodeSchema>;

export const invoiceDisputeResolutionSchema = z.enum([
  "CreditIssued",
  "InvoiceUpheld",
  "Rebilled",
  "WrittenOff",
  "CustomerWithdrew",
]);
export type InvoiceDisputeResolution = z.infer<typeof invoiceDisputeResolutionSchema>;

/** Resolutions that moved money and must name the executed adjustment. */
export const DISPUTE_RESOLUTIONS_REQUIRING_ADJUSTMENT: readonly InvoiceDisputeResolution[] = [
  "CreditIssued",
  "WrittenOff",
];

export const creditMemoApplicationStatusSchema = z.enum(["Applied", "Unapplied"]);
export type CreditMemoApplicationStatus = z.infer<typeof creditMemoApplicationStatusSchema>;

export const invoiceDisputeSchema = z.object({
  id: z.string(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  invoiceId: z.string(),
  customerId: z.string(),
  status: invoiceDisputeCaseStatusSchema,
  reasonCode: invoiceDisputeReasonCodeSchema,
  disputedAmount: decimalStringSchema,
  disputedAmountMinor: z.number().int(),
  notes: z.string().default(""),
  openedById: z.string(),
  openedAt: z.number(),
  resolvedById: nullableStringSchema,
  resolvedAt: nullableIntegerSchema,
  resolution: nullableEnumSchema(invoiceDisputeResolutionSchema),
  resolutionAdjustmentId: nullableStringSchema,
  resolutionNotes: z.string().default(""),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
});
export type InvoiceDispute = z.infer<typeof invoiceDisputeSchema>;

export const creditMemoApplicationSchema = z.object({
  id: z.string(),
  creditMemoInvoiceId: z.string(),
  invoiceId: z.string(),
  appliedAmountMinor: z.number().int(),
  accountingDate: z.number(),
  lineNumber: z.number().int(),
  status: creditMemoApplicationStatusSchema,
  unappliedAt: nullableIntegerSchema,
  unappliedById: nullableStringSchema,
  unappliedReason: z.string().default(""),
  createdById: z.string().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
});
export type CreditMemoApplication = z.infer<typeof creditMemoApplicationSchema>;

export const invoicePaymentApplicationSchema = z.object({
  id: z.string(),
  customerPaymentId: z.string(),
  invoiceId: z.string(),
  appliedAmountMinor: z.number().int(),
  shortPayAmountMinor: z.number().int(),
  lineNumber: z.number().int(),
  createdAt: z.number().optional(),
  payment: z
    .object({
      id: z.string(),
      paymentDate: z.number(),
      amountMinor: z.number().int(),
      status: z.enum(["Posted", "Reversed"]),
      paymentMethod: z.string(),
      referenceNumber: z.string().default(""),
    })
    .nullish(),
});
export type InvoicePaymentApplication = z.infer<typeof invoicePaymentApplicationSchema>;

/** Whether an invoice's outbound 210 can go, where it would go, and where the last attempt stands. */
export const invoiceEdiSendPlanSchema = z.object({
  invoiceId: z.string(),
  enabled: z.boolean(),
  autoSend: z.boolean(),
  partnerId: nullableStringSchema,
  partnerName: z.string().default(""),
  documentProfileId: nullableStringSchema,
  communicationMethod: z.string().default(""),
  status: invoiceEdiSendStatusSchema,
  lastMessageId: nullableStringSchema,
  lastError: z.string().default(""),
  sentAt: nullableIntegerSchema,
  blockers: z.array(z.string()).default([]),
});
export type InvoiceEdiSendPlan = z.infer<typeof invoiceEdiSendPlanSchema>;

export const invoiceLineTypeSchema = z.enum(["Freight", "Accessorial", "Memo"]);
export type InvoiceLineType = z.infer<typeof invoiceLineTypeSchema>;

export const invoiceLineChargeMethodSchema = z
  .union([accessorialChargeMethodSchema, z.literal("")])
  .nullish()
  .transform((value) => (value ? value : null));
export type InvoiceLineChargeMethod = NonNullable<z.infer<typeof invoiceLineChargeMethodSchema>>;

export const invoiceLineRateUnitSchema = z
  .union([rateUnitSchema, z.literal("")])
  .nullish()
  .transform((value) => (value ? value : null));

export const invoiceLineSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  invoiceId: z.string(),
  shipmentId: nullableStringSchema,
  shipmentProNumber: nullableStringSchema,
  shipmentBol: nullableStringSchema,
  lineNumber: z.number().int(),
  type: invoiceLineTypeSchema,
  description: z.string(),
  quantity: decimalStringSchema,
  unitPrice: decimalStringSchema,
  amount: decimalStringSchema,
  accessorialChargeId: nullableStringSchema,
  chargeCode: nullableStringSchema,
  chargeMethod: invoiceLineChargeMethodSchema,
  rateUnit: invoiceLineRateUnitSchema,
  rate: decimalStringSchema.nullish(),
  rateBasisAmount: decimalStringSchema.nullish(),
  formulaTemplateName: nullableStringSchema,
  /** The share of the charge this line bills, when it is less than all of it. */
  allocationPercent: decimalStringSchema.nullish(),
  chargeAllocationId: nullableStringSchema,
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
  scope: invoiceScopeSchema.default("Shipment"),
  /** The invoice run that billed this invoice. Set on consolidated invoices only. */
  invoiceRunId: nullableStringSchema,
  /** The billing period a consolidated invoice covers. Null on every other scope. */
  periodStart: nullableIntegerSchema,
  periodEnd: nullableIntegerSchema,
  /** Distinct shipments this invoice bills. */
  shipmentCount: z.number().int().nonnegative().default(0),
  detail: invoiceDetailSchema.default("Detailed"),
  sectionBy: invoiceSectionKeySchema.default("Shipment"),
  /**
   * Why this invoice was cut for a customer whose freight was supposed to
   * accumulate onto a statement. Empty on every ordinary invoice.
   */
  offCycleReason: nullableStringSchema,
  orderId: nullableStringSchema,
  customerId: z.string(),
  number: z.string(),
  billType: billTypeSchema,
  status: invoiceStatusSchema,
  paymentTerm: customerPaymentTermSchema,
  currencyCode: z.string(),
  exchangeRate: z.string().nullable().optional(),
  exchangeRateDate: nullableIntegerSchema.optional(),
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
  /** The customer who ordered the freight when somebody else is billed for it. */
  shipperCustomerId: nullableStringSchema,
  shipperCustomer: customerSchema.nullish().transform((value) => value ?? undefined),
  /** Whether this invoice bills only part of what its shipments charged. */
  isSplitBill: z.boolean().default(false),
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
  voidedAt: nullableIntegerSchema,
  voidedById: nullableStringSchema,
  voidReason: nullableStringSchema,
  voidDisposition: nullableEnumSchema(invoiceVoidDispositionSchema),
  voidedByAdjustmentId: nullableStringSchema,
  /** The invoice a memo corrects, when it corrects one. */
  referenceInvoiceId: nullableStringSchema,
  memoReason: nullableStringSchema,
  memoKind: nullableEnumSchema(invoiceMemoKindSchema),
  balanceDueMinor: z.number().int().optional().default(0),
  ediSendStatus: invoiceEdiSendStatusSchema.default("NotSent"),
  lastEdiMessageId: nullableStringSchema,
  ediSentAt: nullableIntegerSchema,
  lastEdiError: nullableStringSchema,
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

export const MAX_VOID_REASON_LENGTH = 1000;
export const MAX_MEMO_REASON_LENGTH = 1000;
export const MAX_DISPUTE_NOTES_LENGTH = 2000;

export const voidInvoiceFormSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(1, "Say why the invoice is being voided")
    .max(MAX_VOID_REASON_LENGTH, `Reason must be at most ${MAX_VOID_REASON_LENGTH} characters`),
  disposition: invoiceVoidDispositionSchema,
});
export type VoidInvoiceFormValues = z.infer<typeof voidInvoiceFormSchema>;

/**
 * An invoice can only be voided while nothing has been settled against it:
 * cash and credit memos applied to it must be unapplied first. The server
 * refuses too; this keeps the button honest before the round trip.
 */
export function invoiceVoidBlocker(
  invoice: Pick<Invoice, "status" | "appliedAmount">,
  applications: { paymentApplications: number; creditApplications: number },
): string | null {
  if (invoice.status === "Voided") return "This invoice has already been voided.";
  if (
    applications.paymentApplications > 0 ||
    applications.creditApplications > 0 ||
    Number(invoice.appliedAmount ?? 0) > 0
  ) {
    return "Unapply the customer payments and credit memos on this invoice before voiding it.";
  }
  return null;
}

export const memoLineFormSchema = z.object({
  description: z.string().trim().min(1, "Description is required"),
  amount: z.coerce.number().positive("Amount must be greater than zero"),
  quantity: z.coerce.number().positive("Quantity must be greater than zero").default(1),
  accessorialChargeId: z.string().nullish(),
});
export type MemoLineFormValues = z.infer<typeof memoLineFormSchema>;

export const memoFormSchema = z.object({
  customerId: z.string().min(1, "Customer is required"),
  billType: z.enum(["CreditMemo", "DebitMemo"]),
  referenceInvoiceId: z.string().nullish(),
  reason: z
    .string()
    .trim()
    .min(1, "Say why the memo is being raised")
    .max(MAX_MEMO_REASON_LENGTH, `Reason must be at most ${MAX_MEMO_REASON_LENGTH} characters`),
  invoiceDate: z.number().int().nullish(),
  memo: z.string().optional().default(""),
  autoPost: z.boolean().default(false),
  lines: z.array(memoLineFormSchema).min(1, "A memo needs at least one line"),
});
export type MemoFormValues = z.infer<typeof memoFormSchema>;

export function memoFormTotal(lines: readonly Pick<MemoLineFormValues, "amount" | "quantity">[]) {
  return lines.reduce((sum, line) => {
    const amount = Number(line.amount) || 0;
    const quantity = Number(line.quantity) || 0;
    return sum + amount * (quantity > 0 ? quantity : 1);
  }, 0);
}

export const openDisputeFormSchema = z.object({
  reasonCode: invoiceDisputeReasonCodeSchema,
  disputedAmount: z.coerce.number().positive("Disputed amount must be greater than zero"),
  notes: z
    .string()
    .max(MAX_DISPUTE_NOTES_LENGTH, `Notes must be at most ${MAX_DISPUTE_NOTES_LENGTH} characters`)
    .optional()
    .default(""),
});
export type OpenDisputeFormValues = z.infer<typeof openDisputeFormSchema>;

export const resolveDisputeFormSchema = z
  .object({
    resolution: invoiceDisputeResolutionSchema,
    resolutionAdjustmentId: z.string().nullish(),
    resolutionNotes: z
      .string()
      .max(MAX_DISPUTE_NOTES_LENGTH, `Notes must be at most ${MAX_DISPUTE_NOTES_LENGTH} characters`)
      .optional()
      .default(""),
  })
  .superRefine((value, ctx) => {
    if (
      DISPUTE_RESOLUTIONS_REQUIRING_ADJUSTMENT.includes(value.resolution) &&
      !value.resolutionAdjustmentId
    ) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["resolutionAdjustmentId"],
        message: "Name the executed adjustment that settled this dispute",
      });
    }
  });
export type ResolveDisputeFormValues = z.infer<typeof resolveDisputeFormSchema>;
