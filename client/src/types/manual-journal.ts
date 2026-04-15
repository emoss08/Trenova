import { z } from "zod";
import { nullableStringSchema, optionalStringSchema, tenantInfoSchema } from "./helpers";

export const manualJournalStatusSchema = z.enum([
  "Draft",
  "PendingApproval",
  "Approved",
  "Rejected",
  "Cancelled",
  "Posted",
]);
export type ManualJournalStatus = z.infer<typeof manualJournalStatusSchema>;

export const manualJournalLineSchema = z.object({
  id: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  manualJournalRequestId: optionalStringSchema,
  lineNumber: z.number().int(),
  glAccountId: z.string(),
  description: z.string(),
  debitAmount: z.number().int(),
  creditAmount: z.number().int(),
  customerId: nullableStringSchema,
  locationId: nullableStringSchema,
  createdAt: z.number().int().optional(),
  updatedAt: z.number().int().optional(),
});
export type ManualJournalLine = z.infer<typeof manualJournalLineSchema>;

export const manualJournalSchema = z.object({
  ...tenantInfoSchema.shape,
  requestNumber: z.string(),
  status: manualJournalStatusSchema,
  description: z.string(),
  reason: z.string(),
  accountingDate: z.number().int(),
  requestedFiscalYearId: z.string(),
  requestedFiscalPeriodId: z.string(),
  currencyCode: z.string(),
  totalDebit: z.number().int(),
  totalCredit: z.number().int(),
  approvedAt: z.number().int().nullish(),
  approvedById: nullableStringSchema,
  rejectedAt: z.number().int().nullish(),
  rejectedById: nullableStringSchema,
  rejectionReason: optionalStringSchema,
  cancelledAt: z.number().int().nullish(),
  cancelledById: nullableStringSchema,
  cancelReason: optionalStringSchema,
  postedBatchId: nullableStringSchema,
  createdById: optionalStringSchema,
  updatedById: optionalStringSchema,
  lines: z.array(manualJournalLineSchema).optional(),
});
export type ManualJournal = z.infer<typeof manualJournalSchema>;
