import { z } from "zod";
import { decimalStringSchema, nullableStringSchema, tenantInfoSchema } from "./helpers";
import { invoiceSplitKeySchema } from "./customer";

export const invoiceRunStatusSchema = z.enum([
  "Building",
  "Ready",
  "Committing",
  "Committed",
  "Failed",
  "Canceled",
]);
export type InvoiceRunStatus = z.infer<typeof invoiceRunStatusSchema>;

export const invoiceRunSourceSchema = z.enum(["Manual", "Scheduled"]);
export type InvoiceRunSource = z.infer<typeof invoiceRunSourceSchema>;

export const invoiceRunGroupStatusSchema = z.enum([
  "Pending",
  "Committed",
  "Skipped",
  "Failed",
]);
export type InvoiceRunGroupStatus = z.infer<typeof invoiceRunGroupStatusSchema>;

export const invoiceRunGroupItemSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  runId: z.string(),
  groupId: z.string(),
  billingQueueItemId: z.string(),
  shipmentId: z.string(),
  orderId: nullableStringSchema,
  proNumber: nullableStringSchema,
  bol: nullableStringSchema,
  poNumber: nullableStringSchema,
  serviceDate: z.number().int().nullish(),
  sortKey: z.number().int().default(0),
  amount: decimalStringSchema,
  excluded: z.boolean().default(false),
  exclusionReason: nullableStringSchema,
});
export type InvoiceRunGroupItem = z.infer<typeof invoiceRunGroupItemSchema>;

export const invoiceRunGroupSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  runId: z.string(),
  customerId: z.string(),
  groupKey: z.string(),
  groupLabel: z.string(),
  splitBy: invoiceSplitKeySchema,
  status: invoiceRunGroupStatusSchema,
  itemCount: z.number().int().default(0),
  subtotalAmount: decimalStringSchema,
  totalAmount: decimalStringSchema,
  currencyCode: z.string().default("USD"),
  invoiceId: nullableStringSchema,
  skipReason: nullableStringSchema,
  customer: z
    .object({ id: z.string(), name: z.string(), code: z.string().nullish() })
    .nullish(),
  items: z.array(invoiceRunGroupItemSchema).default([]),
});
export type InvoiceRunGroup = z.infer<typeof invoiceRunGroupSchema>;

export const invoiceRunSchema = z.object({
  ...tenantInfoSchema.shape,
  id: z.string(),
  number: z.string(),
  status: invoiceRunStatusSchema,
  source: invoiceRunSourceSchema,
  cycle: nullableStringSchema,
  periodStart: z.number().int(),
  periodEnd: z.number().int(),
  invoiceDate: z.number().int(),
  customerIds: z.array(z.string()).nullish(),
  currencyCode: z.string().default("USD"),
  groupCount: z.number().int().default(0),
  itemCount: z.number().int().default(0),
  excludedCount: z.number().int().default(0),
  invoiceCount: z.number().int().default(0),
  totalAmount: decimalStringSchema,
  failureReason: nullableStringSchema,
  builtAt: z.number().int().nullish(),
  committedAt: z.number().int().nullish(),
  canceledAt: z.number().int().nullish(),
  version: z.number().int().optional(),
  createdAt: z.number().int().optional(),
  updatedAt: z.number().int().optional(),
  groups: z.array(invoiceRunGroupSchema).nullish(),
});
export type InvoiceRun = z.infer<typeof invoiceRunSchema>;

export const commitGroupResultSchema = z.object({
  groupId: z.string(),
  groupLabel: z.string(),
  success: z.boolean(),
  skipped: z.boolean(),
  invoiceId: nullableStringSchema,
  invoiceNumber: nullableStringSchema,
  error: nullableStringSchema,
});
export type CommitGroupResult = z.infer<typeof commitGroupResultSchema>;

export const commitInvoiceRunResultSchema = z.object({
  run: invoiceRunSchema,
  results: z.array(commitGroupResultSchema).default([]),
  totalCount: z.number().int().default(0),
  successCount: z.number().int().default(0),
  skippedCount: z.number().int().default(0),
  errorCount: z.number().int().default(0),
});
export type CommitInvoiceRunResult = z.infer<typeof commitInvoiceRunResultSchema>;

export type PreviewInvoiceRunInput = {
  customerIds: string[];
  periodStart: number;
  periodEnd: number;
  invoiceDate?: number;
};

export type AdjustMembershipInput = {
  exclude?: { itemId: string; reason: string }[];
  include?: string[];
  moves?: { itemId: string; targetGroupId: string }[];
};

/** A run an operator can still change. */
export function isRunEditable(status: InvoiceRunStatus): boolean {
  return status === "Ready";
}

/** A run that can never change again. */
export function isRunTerminal(status: InvoiceRunStatus): boolean {
  return status === "Committed" || status === "Failed" || status === "Canceled";
}
