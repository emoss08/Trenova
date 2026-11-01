import {
  BalanceSheetDocument,
  IncomeStatementDocument,
  TrialBalanceDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { BalanceSheet } from "@/types/balance-sheet";
import type { PeriodAccountBalance } from "@/types/gl-balance";
import type { IncomeStatement } from "@/types/income-statement";

type RequestOptions = { signal?: AbortSignal };

export class AccountingReportService {
  async getTrialBalance(
    fiscalPeriodId: string,
    { signal }: RequestOptions = {},
  ): Promise<PeriodAccountBalance[]> {
    const response = await requestGraphQL({
      document: TrialBalanceDocument,
      operationName: "TrialBalance",
      variables: { fiscalPeriodId },
      signal,
    });
    return response.trialBalance;
  }

  async getIncomeStatement(
    fiscalPeriodId: string,
    { signal }: RequestOptions = {},
  ): Promise<IncomeStatement> {
    const response = await requestGraphQL({
      document: IncomeStatementDocument,
      operationName: "IncomeStatement",
      variables: { fiscalPeriodId },
      signal,
    });
    return response.incomeStatement;
  }

  async getBalanceSheet(
    fiscalPeriodId: string,
    { signal }: RequestOptions = {},
  ): Promise<BalanceSheet> {
    const response = await requestGraphQL({
      document: BalanceSheetDocument,
      operationName: "BalanceSheet",
      variables: { fiscalPeriodId },
      signal,
    });
    return response.balanceSheet;
  }
}
