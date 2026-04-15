import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const accountingReport = createQueryKeys("accountingReport", {
  trialBalance: (fiscalPeriodId: string) => ({
    queryKey: ["trialBalance", fiscalPeriodId],
    queryFn: async () => apiService.accountingReportService.getTrialBalance(fiscalPeriodId),
  }),
  incomeStatement: (fiscalPeriodId: string) => ({
    queryKey: ["incomeStatement", fiscalPeriodId],
    queryFn: async () => apiService.accountingReportService.getIncomeStatement(fiscalPeriodId),
  }),
  balanceSheet: (fiscalPeriodId: string) => ({
    queryKey: ["balanceSheet", fiscalPeriodId],
    queryFn: async () => apiService.accountingReportService.getBalanceSheet(fiscalPeriodId),
  }),
});
