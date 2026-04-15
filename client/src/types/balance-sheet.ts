import { z } from "zod";
import { statementSectionSchema } from "./income-statement";

export const balanceSheetSchema = z.object({
  fiscalPeriodId: z.string(),
  assets: statementSectionSchema.nullish(),
  liabilities: statementSectionSchema.nullish(),
  equity: statementSectionSchema.nullish(),
  currentPeriodNetIncomeMinor: z.number().int(),
  totalAssetsMinor: z.number().int(),
  totalLiabilitiesMinor: z.number().int(),
  totalEquityMinor: z.number().int(),
});
export type BalanceSheet = z.infer<typeof balanceSheetSchema>;
