import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { matchRate, reconciliationHasActivity } from "@/lib/reconciliation-summary";
import type { ReconciliationSummary } from "@/types/bank-receipt";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { ArrowRightIcon } from "lucide-react";
import { Link } from "react-router";
import { ReconciliationSummaryEmpty } from "./_components/reconciliation-summary-empty";
import { ReconciliationSummarySkeleton } from "./_components/reconciliation-summary-skeleton";

export function ReconciliationSummaryPage() {
  const t = useT();

  const { data, isLoading } = useQuery({
    ...queries.bankReceipt.summary(),
  });

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Reconciliation Summary"),
        description: t("Overview of bank receipt reconciliation status."),
      }}
      className="p-0"
    >
      <div className="mx-4 mt-3 mb-4">
        {isLoading ? (
          <ReconciliationSummarySkeleton />
        ) : data ? (
          reconciliationHasActivity(data) ? (
            <SummaryBody data={data} />
          ) : (
            <ReconciliationSummaryEmpty
              title={t("Nothing to reconcile yet")}
              description={t(
                "Import a bank receipt file and every receipt it carries is matched against open invoices; what matched, what did not, and how long the exceptions have waited are counted here.",
              )}
            />
          )
        ) : null}
      </div>
    </PageLayout>
  );
}

function SummaryBody({ data }: { data: ReconciliationSummary }) {
  const t = useT();

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 gap-2.5 xl:grid-cols-4">
        <SummaryKPICard
          label={t("Imported")}
          count={data.importedCount}
          amount={data.importedAmount}
        />
        <SummaryKPICard
          label={t("Matched")}
          count={data.matchedCount}
          amount={data.matchedAmount}
        />
        <SummaryKPICard
          label={t("Exceptions")}
          count={data.exceptionCount}
          amount={data.exceptionAmount}
          variant="danger"
        />
        <Card className="gap-0 overflow-hidden rounded-md">
          <CardHeader className="pb-1">
            <CardTitle className="text-muted-foreground text-[11px] font-semibold tracking-wide uppercase">
              {t("Match Rate")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-semibold tracking-tight tabular-nums">{matchRate(data)}%</p>
          </CardContent>
        </Card>
      </div>
      <div className="grid gap-4 xl:grid-cols-2">
        <Card className="rounded-md">
          <CardHeader>
            <CardTitle className="text-sm font-semibold">{t("Exception Aging")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-hidden rounded-md border">
              <table className="w-full text-sm">
                <thead className="bg-muted/50 text-muted-foreground text-left">
                  <tr>
                    <th className="px-3 py-2 text-xs font-medium">{t("Period")}</th>
                    <th className="px-3 py-2 text-right text-xs font-medium">{t("Count")}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr className="border-t">
                    <td className="px-3 py-2 text-xs">{t("Current")}</td>
                    <td className="px-3 py-2 text-right font-mono text-xs">
                      {data.exceptionAging.currentCount}
                    </td>
                  </tr>
                  <tr className="border-t">
                    <td className="px-3 py-2 text-xs">{t("1-3 Days")}</td>
                    <td className="px-3 py-2 text-right font-mono text-xs">
                      {data.exceptionAging.days1To3Count}
                    </td>
                  </tr>
                  <tr className="border-t">
                    <td className="px-3 py-2 text-xs">{t("4-7 Days")}</td>
                    <td className="px-3 py-2 text-right font-mono text-xs">
                      {data.exceptionAging.days4To7Count}
                    </td>
                  </tr>
                  <tr className="border-t">
                    <td className="px-3 py-2 text-xs">{t("7+ Days")}</td>
                    <td className="px-3 py-2 text-right font-mono text-xs font-semibold text-red-600 dark:text-red-400">
                      {data.exceptionAging.daysOver7Count}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
        <Card className="rounded-md">
          <CardHeader>
            <CardTitle className="text-sm font-semibold">{t("Work Items")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">{t("Active")}</span>
                <span className="font-mono font-medium">{data.activeWorkItemCount}</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">{t("Assigned")}</span>
                <span className="font-mono font-medium">{data.assignedWorkItemCount}</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">{t("In Review")}</span>
                <span className="font-mono font-medium">{data.inReviewWorkItemCount}</span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
      <div className="flex items-center gap-3">
        <Link to="/accounting/reconciliation/bank-receipts">
          <Button variant="outline" size="sm">
            {t("Bank Receipts")}
            <ArrowRightIcon className="ml-1.5 size-3.5" />
          </Button>
        </Link>
        <Link to="/accounting/reconciliation/work-queue">
          <Button variant="outline" size="sm">
            {t("Work Queue")}
            <ArrowRightIcon className="ml-1.5 size-3.5" />
          </Button>
        </Link>
      </div>
    </div>
  );
}

function SummaryKPICard({
  label,
  count,
  amount,
  variant,
}: {
  label: string;
  count: number;
  amount: number;
  variant?: "danger";
}) {
  return (
    <Card className="gap-0 overflow-hidden rounded-md">
      <CardHeader className="pb-1">
        <CardTitle className="text-muted-foreground text-[11px] font-semibold tracking-wide uppercase">
          {label}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p
          className={`text-2xl font-semibold tracking-tight tabular-nums ${
            variant === "danger" ? "text-red-600 dark:text-red-400" : ""
          }`}
        >
          {count}
        </p>
        <p className="text-muted-foreground text-xs tabular-nums">{formatCurrency(amount / 100)}</p>
      </CardContent>
    </Card>
  );
}
