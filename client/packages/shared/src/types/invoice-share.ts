import { z } from "zod";
import { nullableStringSchema } from "./helpers";
import { createLimitOffsetResponse } from "./server";

export const INVOICE_DETAIL_TABS = [
  "overview",
  "delivery",
  "charges",
  "documents",
  "activity",
] as const;
export type InvoiceDetailTab = (typeof INVOICE_DETAIL_TABS)[number];

export const MAX_INVOICE_SHARE_RECIPIENTS = 25;
export const MAX_INVOICE_SHARE_NOTE_LENGTH = 1000;

export const invoiceDetailTabSchema = z.enum(INVOICE_DETAIL_TABS);

export const invoiceShareUserSchema = z.object({
  id: z.string(),
  name: z.string(),
  username: z.string().optional(),
  emailAddress: z.string().optional().default(""),
  profilePicUrl: nullableStringSchema,
  thumbnailUrl: nullableStringSchema,
});
export type InvoiceShareUser = z.infer<typeof invoiceShareUserSchema>;

export const invoiceShareCandidateListSchema = createLimitOffsetResponse(invoiceShareUserSchema);
export type InvoiceShareCandidateList = z.infer<typeof invoiceShareCandidateListSchema>;

export const invoiceShareSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  invoiceId: z.string(),
  sharedWithId: z.string(),
  sharedById: z.string(),
  note: nullableStringSchema,
  tab: invoiceDetailTabSchema,
  shareCount: z.number().int().positive(),
  firstSharedAt: z.number(),
  lastSharedAt: z.number(),
  sharedWith: invoiceShareUserSchema.nullish(),
  sharedBy: invoiceShareUserSchema.nullish(),
});
export type InvoiceShare = z.infer<typeof invoiceShareSchema>;

export const invoiceShareListSchema = z.object({
  shares: z
    .array(invoiceShareSchema)
    .nullish()
    .transform((value) => value ?? []),
});
export type InvoiceShareList = z.infer<typeof invoiceShareListSchema>;

export const invoiceShareEmailStatusSchema = z.enum([
  "Queued",
  "Partial",
  "Failed",
  "NotConfigured",
]);
export type InvoiceShareEmailStatus = z.infer<typeof invoiceShareEmailStatusSchema>;

export const shareInvoiceResultSchema = z.object({
  shares: z
    .array(invoiceShareSchema)
    .nullish()
    .transform((value) => value ?? []),
  recipientCount: z.number().int().nonnegative(),
  emailsQueued: z.number().int().nonnegative(),
  emailStatus: invoiceShareEmailStatusSchema,
});
export type ShareInvoiceResult = z.infer<typeof shareInvoiceResultSchema>;

export const shareInvoiceFormSchema = z.object({
  userIds: z
    .array(z.string().min(1))
    .min(1, "Choose at least one teammate to share with")
    .max(
      MAX_INVOICE_SHARE_RECIPIENTS,
      "An invoice can be shared with at most 25 teammates at a time",
    ),
  note: z.string().max(MAX_INVOICE_SHARE_NOTE_LENGTH, "Note must be 1000 characters or fewer"),
});
export type ShareInvoiceFormValues = z.infer<typeof shareInvoiceFormSchema>;

export type ShareInvoicePayload = ShareInvoiceFormValues & { tab: InvoiceDetailTab };
