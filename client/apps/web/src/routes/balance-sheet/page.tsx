import { AmountDisplay } from "@/components/accounting/amount-display";
import { FinancialReportSection } from "@/components/accounting/financial-report-section";
import { FiscalPeriodSelector } from "@/components/accounting/fiscal-period-selector";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

export function BalanceSheetPage() {
  const [periodId, setPeriodId] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    ...queries.accountingReport.balanceSheet(periodId!),
    enabled: Boolean(periodId),
  });

  const isBalanced =
    data && data.totalAssetsMinor === data.totalLiabilitiesMinor + data.totalEquityMinor;

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Balance Sheet",
        description: "Assets, liabilities, and equity as of a fiscal period.",
      }}
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <FiscalPeriodSelector value={periodId} onChange={setPeriodId} />

        {!periodId ? (
          <div className="flex h-64 items-center justify-center rounded-lg border bg-card">
            <p className="text-sm text-muted-foreground">
              Select a fiscal period to view the balance sheet.
            </p>
          </div>
        ) : isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-48 w-full" />
            <Skeleton className="h-48 w-full" />
            <Skeleton className="h-48 w-full" />
          </div>
        ) : data ? (
          <div className="space-y-6">
            <FinancialReportSection section={data.assets} />

            <div className="flex items-center justify-between rounded-md border bg-muted/30 px-4 py-3">
              <span className="text-sm font-semibold">Total Assets</span>
              <AmountDisplay
                value={data.totalAssetsMinor}
                className="text-lg font-bold"
              />
            </div>

            <Separator />

            <FinancialReportSection section={data.liabilities} />

            <div className="flex items-center justify-between rounded-md border bg-muted/30 px-4 py-3">
              <span className="text-sm font-semibold">Total Liabilities</span>
              <AmountDisplay
                value={data.totalLiabilitiesMinor}
                className="text-lg font-bold"
              />
            </div>

            <FinancialReportSection section={data.equity} />

            {data.currentPeriodNetIncomeMinor !== 0 ? (
              <div className="flex items-center justify-between rounded-md border px-4 py-2 text-sm">
                <span className="text-muted-foreground">Current Period Net Income</span>
                <AmountDisplay
                  value={data.currentPeriodNetIncomeMinor}
                  variant="auto"
                  className="font-medium"
                />
              </div>
            ) : null}

            <div className="flex items-center justify-between rounded-md border bg-muted/30 px-4 py-3">
              <span className="text-sm font-semibold">Total Equity</span>
              <AmountDisplay
                value={data.totalEquityMinor}
                className="text-lg font-bold"
              />
            </div>

            <Separator />

            <div
              className={cn(
                "flex items-center justify-between rounded-md border px-4 py-4",
                isBalanced ? "bg-green-50 dark:bg-green-950/20" : "bg-red-50 dark:bg-red-950/20",
              )}
            >
              <span className="text-base font-bold">
                {isBalanced ? "Balance Sheet is Balanced" : "Balance Sheet is NOT Balanced"}
              </span>
              <div className="flex items-center gap-4">
                <div className="text-right">
                  <p className="text-2xs text-muted-foreground">Assets</p>
                  <AmountDisplay value={data.totalAssetsMinor} className="font-semibold" />
                </div>
                <span className="text-muted-foreground">=</span>
                <div className="text-right">
                  <p className="text-2xs text-muted-foreground">L + E</p>
                  <AmountDisplay
                    value={data.totalLiabilitiesMinor + data.totalEquityMinor}
                    className="font-semibold"
                  />
                </div>
              </div>
            </div>
          </div>
        ) : null}
      </div>
    </PageLayout>
  );
}
