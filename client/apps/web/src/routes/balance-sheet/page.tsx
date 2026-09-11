import { FinancialReportEmpty } from "@/components/accounting/accounting-empty";
import { FinancialReportSection } from "@/components/accounting/financial-report-section";
import { FiscalPeriodSelector } from "@/components/accounting/fiscal-period-selector";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Separator } from "@trenova/shared/components/ui/separator";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
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
      className="p-0"
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <FiscalPeriodSelector value={periodId} onChange={setPeriodId} />

        {!periodId ? (
          <FinancialReportEmpty
            title="Pick a period"
            description="Choose a fiscal period above and the assets, liabilities and equity as they stood at its close are laid out here."
          />
        ) : isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-48 w-full" />
            <Skeleton className="h-48 w-full" />
            <Skeleton className="h-48 w-full" />
          </div>
        ) : data ? (
          <div className="space-y-6">
            <FinancialReportSection section={data.assets} />

            <div className="bg-muted/30 flex items-center justify-between rounded-md border px-4 py-3">
              <span className="text-sm font-semibold">Total Assets</span>
              <AmountDisplay value={data.totalAssetsMinor} className="text-lg font-bold" />
            </div>

            <Separator />

            <FinancialReportSection section={data.liabilities} />

            <div className="bg-muted/30 flex items-center justify-between rounded-md border px-4 py-3">
              <span className="text-sm font-semibold">Total Liabilities</span>
              <AmountDisplay value={data.totalLiabilitiesMinor} className="text-lg font-bold" />
            </div>

            <FinancialReportSection section={data.equity} />

            {data.currentYearEarningsMinor !== 0 ? (
              <div className="flex items-center justify-between rounded-md border px-4 py-2 text-sm">
                <span className="text-muted-foreground">Current Year Earnings</span>
                <AmountDisplay
                  value={data.currentYearEarningsMinor}
                  variant="auto"
                  className="font-medium"
                />
              </div>
            ) : null}

            <div className="bg-muted/30 flex items-center justify-between rounded-md border px-4 py-3">
              <span className="text-sm font-semibold">Total Equity</span>
              <AmountDisplay value={data.totalEquityMinor} className="text-lg font-bold" />
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
