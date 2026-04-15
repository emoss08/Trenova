import { api } from "@/lib/api";
import type { BalanceSheet } from "@/types/balance-sheet";
import type { PeriodAccountBalance } from "@/types/gl-balance";
import type { IncomeStatement } from "@/types/income-statement";

export class AccountingReportService {
  async getTrialBalance(fiscalPeriodId: string) {
    return api.get<PeriodAccountBalance[]>(`/accounting/trial-balance/${fiscalPeriodId}/`);
  }

  async getIncomeStatement(fiscalPeriodId: string) {
    return api.get<IncomeStatement>(
      `/accounting/statements/income-statement/${fiscalPeriodId}/`,
    );
  }

  async getBalanceSheet(fiscalPeriodId: string) {
    return api.get<BalanceSheet>(`/accounting/statements/balance-sheet/${fiscalPeriodId}/`);
  }
}
