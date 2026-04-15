import { z } from "zod";
import { nullableStringSchema, optionalStringSchema } from "./helpers";

export const journalEntryLineSchema = z.object({
  id: z.string(),
  journalEntryId: z.string(),
  glAccountId: z.string(),
  lineNumber: z.number().int(),
  description: z.string(),
  debitAmount: z.number().int(),
  creditAmount: z.number().int(),
  netAmount: z.number().int(),
  customerId: nullableStringSchema,
  locationId: nullableStringSchema,
});
export type JournalEntryLine = z.infer<typeof journalEntryLineSchema>;

export const journalEntrySchema = z.object({
  id: z.string(),
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  batchId: optionalStringSchema,
  fiscalYearId: z.string(),
  fiscalPeriodId: z.string(),
  entryNumber: z.string(),
  entryType: z.string(),
  status: z.string(),
  accountingDate: z.number().int(),
  description: z.string(),
  referenceType: z.string(),
  referenceId: z.string(),
  totalDebit: z.number().int(),
  totalCredit: z.number().int(),
  isPosted: z.boolean(),
  isReversal: z.boolean(),
  reversalOfId: nullableStringSchema,
  reversedById: nullableStringSchema,
  reversalDate: z.number().int().nullish(),
  reversalReason: optionalStringSchema,
  lines: z.array(journalEntryLineSchema).optional(),
});
export type JournalEntry = z.infer<typeof journalEntrySchema>;
