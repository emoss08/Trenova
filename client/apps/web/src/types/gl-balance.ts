import { z } from "zod";

export const periodAccountBalanceSchema = z.object({
  organizationId: z.string(),
  businessUnitId: z.string(),
  glAccountId: z.string(),
  fiscalYearId: z.string(),
  fiscalPeriodId: z.string(),
  accountCode: z.string(),
  accountName: z.string(),
  accountCategory: z.string(),
  periodDebitMinor: z.number().int(),
  periodCreditMinor: z.number().int(),
  netChangeMinor: z.number().int(),
});
export type PeriodAccountBalance = z.infer<typeof periodAccountBalanceSchema>;
