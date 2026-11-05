import { accountingReportService } from "@/services/accounting-report";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const accountingReport = createQueryKeys("accountingReport", {
  trialBalance: (fiscalPeriodId: string) => ({
    queryKey: ["trialBalance", fiscalPeriodId],
    queryFn: async () => accountingReportService.getTrialBalance(fiscalPeriodId),
  }),
  incomeStatement: (fiscalPeriodId: string) => ({
    queryKey: ["incomeStatement", fiscalPeriodId],
    queryFn: async () => accountingReportService.getIncomeStatement(fiscalPeriodId),
  }),
  balanceSheet: (fiscalPeriodId: string) => ({
    queryKey: ["balanceSheet", fiscalPeriodId],
    queryFn: async () => accountingReportService.getBalanceSheet(fiscalPeriodId),
  }),
});
