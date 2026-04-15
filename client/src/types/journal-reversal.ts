import { z } from "zod";
import { nullableStringSchema, optionalStringSchema, tenantInfoSchema } from "./helpers";

export const journalReversalStatusSchema = z.enum([
  "Requested",
  "PendingApproval",
  "Approved",
  "Rejected",
  "Cancelled",
  "Posted",
]);
export type JournalReversalStatus = z.infer<typeof journalReversalStatusSchema>;

export const journalReversalSchema = z.object({
  ...tenantInfoSchema.shape,
  originalJournalEntryId: z.string(),
  reversalJournalEntryId: nullableStringSchema,
  postedBatchId: nullableStringSchema,
  status: journalReversalStatusSchema,
  requestedAccountingDate: z.number().int(),
  resolvedFiscalYearId: optionalStringSchema,
  resolvedFiscalPeriodId: optionalStringSchema,
  reasonCode: z.string(),
  reasonText: z.string(),
  requestedById: optionalStringSchema,
  approvedById: nullableStringSchema,
  approvedAt: z.number().int().nullish(),
  rejectedById: nullableStringSchema,
  rejectedAt: z.number().int().nullish(),
  rejectionReason: optionalStringSchema,
  cancelledById: nullableStringSchema,
  cancelledAt: z.number().int().nullish(),
  cancelReason: optionalStringSchema,
  postedById: nullableStringSchema,
  postedAt: z.number().int().nullish(),
});
export type JournalReversal = z.infer<typeof journalReversalSchema>;
