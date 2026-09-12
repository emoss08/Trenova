import { useT } from "@trenova/shared/i18n/use-t";
import { FinancialReportEmpty } from "@/components/accounting/accounting-empty";
import { FinancialReportSection } from "@/components/accounting/financial-report-section";
import { FiscalPeriodSelector } from "@/components/accounting/fiscal-period-selector";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Separator } from "@trenova/shared/components/ui/separator";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useState } from "react";

export function IncomeStatementPage() {
  const t = useT();

  const [periodId, setPeriodId] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    ...queries.accountingReport.incomeStatement(periodId!),
    enabled: Boolean(periodId),
  });

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Income Statement"),
        description: t("Revenue, expenses, and net income for a fiscal period."),
      }}
      className="p-0"
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <FiscalPeriodSelector value={periodId} onChange={setPeriodId} />

        {!periodId ? (
          <FinancialReportEmpty
            title={t("Pick a period")}
            description={t(
              "Choose a fiscal period above and its revenue, cost of revenue, operating expenses and net income are laid out here.",
            )}
          />
        ) : isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-48 w-full" />
            <Skeleton className="h-48 w-full" />
          </div>
        ) : data ? (
          <div className="space-y-6">
            <FinancialReportSection section={data.revenue} />
            <FinancialReportSection section={data.costOfRevenue} />

            <div className="bg-muted/30 flex items-center justify-between rounded-md border px-4 py-3">
              <span className="text-sm font-semibold">{t("Gross Profit")}</span>
              <AmountDisplay
                value={data.grossProfitMinor}
                variant="auto"
                className="text-lg font-bold"
              />
            </div>

            <FinancialReportSection section={data.operatingExpense} />

            <Separator />

            <div className="bg-primary/5 flex items-center justify-between rounded-md border px-4 py-4">
              <span className="text-base font-bold">{t("Net Income")}</span>
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
