import { AmountDisplay } from "@/components/accounting/amount-display";
import { FinancialReportSection } from "@/components/accounting/financial-report-section";
import { FiscalPeriodSelector } from "@/components/accounting/fiscal-period-selector";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

export function IncomeStatementPage() {
  const [periodId, setPeriodId] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    ...queries.accountingReport.incomeStatement(periodId!),
    enabled: Boolean(periodId),
  });

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Income Statement",
        description: "Revenue, expenses, and net income for a fiscal period.",
      }}
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <FiscalPeriodSelector value={periodId} onChange={setPeriodId} />

        {!periodId ? (
          <div className="flex h-64 items-center justify-center rounded-lg border bg-card">
            <p className="text-sm text-muted-foreground">
              Select a fiscal period to view the income statement.
            </p>
          </div>
        ) : isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-48 w-full" />
            <Skeleton className="h-48 w-full" />
          </div>
        ) : data ? (
          <div className="space-y-6">
            <FinancialReportSection section={data.revenue} />
            <FinancialReportSection section={data.costOfRevenue} />

            <div className="flex items-center justify-between rounded-md border bg-muted/30 px-4 py-3">
              <span className="text-sm font-semibold">Gross Profit</span>
              <AmountDisplay
                value={data.grossProfitMinor}
                variant="auto"
                className="text-lg font-bold"
              />
            </div>

            <FinancialReportSection section={data.operatingExpense} />

            <Separator />

            <div className="flex items-center justify-between rounded-md border bg-primary/5 px-4 py-4">
              <span className="text-base font-bold">Net Income</span>
              <AmountDisplay
                value={data.netIncomeMinor}
                variant="auto"
                className="text-2xl font-bold"
              />
            </div>
          </div>
        ) : null}
      </div>
    </PageLayout>
  );
}
