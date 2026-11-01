import { z } from "zod";
import { nullableStringSchema } from "./helpers";

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

/** How billing a statement went: the run it committed and each invoice it tried to cut. */
export const commitInvoiceRunResultSchema = z.object({
  runId: z.string(),
  runNumber: z.string(),
  results: z.array(commitGroupResultSchema).default([]),
  totalCount: z.number().int().default(0),
  successCount: z.number().int().default(0),
  skippedCount: z.number().int().default(0),
  errorCount: z.number().int().default(0),
});
export type CommitInvoiceRunResult = z.infer<typeof commitInvoiceRunResultSchema>;
