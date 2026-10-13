import { z } from "zod";

const nullableId = z
  .string()
  .nullish()
  .transform((value) => value ?? null);
const nullableUnix = z
  .number()
  .int()
  .nullish()
  .transform((value) => value ?? null);

export const collectionActivityKindSchema = z.enum([
  "Note",
  "Call",
  "Email",
  "PromiseToPay",
  "CollectorAssigned",
  "FollowUpScheduled",
]);
export type CollectionActivityKind = z.infer<typeof collectionActivityKindSchema>;

/** The kinds a person logs by hand; the rest are written as the account changes. */
export const contactActivityKinds = ["Call", "Email", "Note", "PromiseToPay"] as const;
export type ContactActivityKind = (typeof contactActivityKinds)[number];

export const promiseStatusSchema = z.enum(["Open", "Kept", "Broken"]);
export type PromiseStatus = z.infer<typeof promiseStatusSchema>;

export const collectionPrioritySchema = z.enum(["Critical", "High", "Normal", "Low"]);
export type CollectionPriority = z.infer<typeof collectionPrioritySchema>;

export const collectionPromiseSchema = z.object({
  activityId: z.string(),
  amountMinor: z.number().int(),
  currencyCode: z.string(),
  promisedOn: z.number().int(),
  madeAt: z.number().int(),
  madeByName: z.string(),
  paidSinceMinor: z.number().int(),
  status: promiseStatusSchema,
});
export type CollectionPromise = z.infer<typeof collectionPromiseSchema>;

export const collectionWorklistRowSchema = z.object({
  customerId: z.string(),
  customerName: z.string(),
  customerCode: z.string(),
  creditStatus: z.string(),
  currencyCode: z.string(),
  totalOpenMinor: z.number().int(),
  overdueMinor: z.number().int(),
  oldestDaysPastDue: z.number().int(),
  openInvoiceCount: z.number().int(),
  disputedInvoiceCount: z.number().int(),
  collectorId: nullableId,
  collectorName: z.string(),
  nextFollowUpOn: nullableUnix,
  followUpDue: z.boolean(),
  lastActivityAt: nullableUnix,
  lastActivityKind: z.union([collectionActivityKindSchema, z.literal("")]),
  lastActivitySummary: z.string(),
  promise: collectionPromiseSchema.nullish().transform((value) => value ?? null),
  priority: collectionPrioritySchema,
});
export type CollectionWorklistRow = z.infer<typeof collectionWorklistRowSchema>;

export const collectionWorklistPageSchema = z.object({
  rows: z
    .array(collectionWorklistRowSchema)
    .nullish()
    .transform((value) => value ?? []),
  total: z.number().int(),
  summary: z.object({
    accounts: z.number().int(),
    overdue: z
      .array(z.object({ currencyCode: z.string(), amountMinor: z.number().int() }))
      .nullish()
      .transform((value) => value ?? []),
    followUpsDue: z.number().int(),
    brokenPromises: z.number().int(),
    unassigned: z.number().int(),
  }),
});
export type CollectionWorklistPage = z.infer<typeof collectionWorklistPageSchema>;

export const collectionActivitySchema = z.object({
  id: z.string(),
  customerId: z.string(),
  kind: collectionActivityKindSchema,
  summary: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  contactName: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  invoiceId: nullableId,
  invoiceNumber: z.string().default(""),
  promisedAmountMinor: nullableUnix,
  promisedOn: nullableUnix,
  currencyCode: z.string(),
  followUpOn: nullableUnix,
  collectorId: nullableId,
  collectorName: z.string().default(""),
  createdById: z.string(),
  createdByName: z.string().default(""),
  createdAt: z.number().int(),
});
export type CollectionActivity = z.infer<typeof collectionActivitySchema>;

export const collectionAccountViewSchema = z.object({
  row: collectionWorklistRowSchema,
  activities: z
    .array(collectionActivitySchema)
    .nullish()
    .transform((value) => value ?? []),
  hasMore: z.boolean(),
});
export type CollectionAccountView = z.infer<typeof collectionAccountViewSchema>;

export const paymentReminderSendResultSchema = z.object({
  reminders: z
    .array(
      z.object({
        sentTo: z.array(z.string()),
        invoiceNumbers: z.array(z.string()),
        totalOverdue: z.string(),
      }),
    )
    .nullish()
    .transform((value) => value ?? []),
});
export type PaymentReminderSendResult = z.infer<typeof paymentReminderSendResultSchema>;

export type CollectorScope = "All" | "Mine" | "Unassigned";

export type CollectionWorklistParams = {
  scope: CollectorScope;
  dueOnly: boolean;
  search: string;
  limit: number;
  offset: number;
};

export type LogCollectionActivityInput = {
  kind: ContactActivityKind;
  summary: string;
  contactName: string;
  invoiceId: string | null;
  promisedAmount: string | null;
  promisedOn: number | null;
  followUpOn: number | null;
};
