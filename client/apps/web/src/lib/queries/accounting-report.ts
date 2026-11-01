import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const accountingReport = createQueryKeys("accountingReport", {
  trialBalance: (fiscalPeriodId: string) => ({
    queryKey: ["trialBalance", fiscalPeriodId],
    queryFn: async ({ signal }) =>
      apiService.accountingReportService.getTrialBalance(fiscalPeriodId, { signal }),
  }),
  incomeStatement: (fiscalPeriodId: string) => ({
    queryKey: ["incomeStatement", fiscalPeriodId],
    queryFn: async ({ signal }) =>
      apiService.accountingReportService.getIncomeStatement(fiscalPeriodId, { signal }),
  }),
  balanceSheet: (fiscalPeriodId: string) => ({
    queryKey: ["balanceSheet", fiscalPeriodId],
    queryFn: async ({ signal }) =>
      apiService.accountingReportService.getBalanceSheet(fiscalPeriodId, { signal }),
  }),
});
