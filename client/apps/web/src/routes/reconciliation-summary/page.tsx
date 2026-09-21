import { useT } from "@trenova/shared/i18n/use-t";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { SectionPanel } from "@/components/section-panel";
import { queries } from "@/lib/queries";
import { matchRate, reconciliationHasActivity } from "@/lib/reconciliation-summary";
import type { ReconciliationSummary } from "@/types/bank-receipt";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
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
        title: t("Reconciliation summary"),
        description: t("Overview of bank receipt reconciliation status."),
      }}
    >
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
    </PageLayout>
  );
}

function SummaryBody({ data }: { data: ReconciliationSummary }) {
  const t = useT();

  return (
    <>
      <KpiStrip aria-label={t("Reconciliation totals")}>
        <KpiStripItem
          label={t("Imported")}
          value={data.importedCount}
          sub={formatCurrency(data.importedAmount / 100)}
        />
        <KpiStripItem
          label={t("Matched")}
          value={data.matchedCount}
          sub={formatCurrency(data.matchedAmount / 100)}
        />
        <KpiStripItem
          label={t("Exceptions")}
          value={data.exceptionCount}
          sub={formatCurrency(data.exceptionAmount / 100)}
          tone="danger"
        />
        <KpiStripItem label={t("Match rate")} value={`${matchRate(data)}%`} />
      </KpiStrip>
      <div className="grid gap-4 xl:grid-cols-2">
        <SectionPanel title={t("Exception aging")}>
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
                <td className="px-3 py-2 text-right font-mono text-xs font-semibold text-danger-foreground">
                  {data.exceptionAging.daysOver7Count}
                </td>
              </tr>
            </tbody>
          </table>
        </SectionPanel>
        <SectionPanel title={t("Work items")}>
          <DescriptionList layout="split" className="px-3 py-1.5">
            <DescriptionItem label={t("Active")} numeric>
              {data.activeWorkItemCount}
            </DescriptionItem>
            <DescriptionItem label={t("Assigned")} numeric>
              {data.assignedWorkItemCount}
            </DescriptionItem>
            <DescriptionItem label={t("In review")} numeric>
              {data.inReviewWorkItemCount}
            </DescriptionItem>
          </DescriptionList>
        </SectionPanel>
      </div>
      <div className="flex items-center gap-3">
        <Link to="/accounting/reconciliation/bank-receipts">
          <Button variant="outline" size="sm">
            {t("Bank receipts")}
            <ArrowRightIcon className="ml-1.5 size-3.5" />
          </Button>
        </Link>
        <Link to="/accounting/reconciliation/work-queue">
          <Button variant="outline" size="sm">
            {t("Work queue")}
            <ArrowRightIcon className="ml-1.5 size-3.5" />
          </Button>
        </Link>
      </div>
    </>
  );
}
