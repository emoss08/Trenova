import { z } from "zod";

export const statementLineSchema = z.object({
  accountCode: z.string(),
  accountName: z.string(),
  amountMinor: z.number().int(),
});
export type StatementLine = z.infer<typeof statementLineSchema>;

export const statementSectionSchema = z.object({
  label: z.string(),
  lines: z.array(statementLineSchema),
  totalMinor: z.number().int(),
});
export type StatementSection = z.infer<typeof statementSectionSchema>;

export const incomeStatementSchema = z.object({
  fiscalPeriodId: z.string(),
  revenue: statementSectionSchema.nullish(),
  costOfRevenue: statementSectionSchema.nullish(),
  operatingExpense: statementSectionSchema.nullish(),
  grossProfitMinor: z.number().int(),
  netIncomeMinor: z.number().int(),
});
export type IncomeStatement = z.infer<typeof incomeStatementSchema>;
