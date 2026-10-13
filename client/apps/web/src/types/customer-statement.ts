import { z } from "zod";

/** One statement emailed to a customer, newest first in its history. */
export const sentStatementSchema = z.object({
  id: z.string(),
  periodStart: z.number().int(),
  periodEnd: z.number().int(),
  endingBalanceMinor: z.number().int(),
  pastDueMinor: z.number().int(),
  currencyCode: z.string(),
  sentTo: z.string(),
  monthly: z.boolean(),
  sentByName: z.string(),
  sentAt: z.number().int(),
});
export type SentStatement = z.infer<typeof sentStatementSchema>;

export const sentStatementListSchema = z.object({
  statements: z.array(sentStatementSchema),
});

export const statementSendResultSchema = z.object({
  sentTo: z.array(z.string()),
  endingBalance: z.string(),
  fileName: z.string(),
});
export type StatementSendResult = z.infer<typeof statementSendResultSchema>;

/** The span a statement covers; a missing start runs from the first transaction. */
export type StatementRange = { startDate?: number; asOfDate?: number };
