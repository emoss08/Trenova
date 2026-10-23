import { useT } from "@trenova/shared/i18n/use-t";
import { PageInsightsCard } from "@/components/insights/page-insights";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { usePermission } from "@/hooks/use-permission";
import { receivablesHaveActivity } from "@/lib/ar-dashboard";
import { ar } from "@/lib/queries/ar";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { CoinsHandIcon } from "@trenova/shared/components/icons";
import { Link } from "react-router";
import { AccountingBriefing } from "./_components/accounting-briefing";
import { AccountingDashboardEmpty } from "./_components/accounting-dashboard-empty";
import { AgingCard } from "./_components/aging-card";
import { ARKpiRow } from "./_components/ar-kpi-row";
import { CashFlowForecastCard } from "./_components/cash-flow-forecast-card";
import { CollectionsPerformanceCard } from "./_components/collections-performance-card";
import { CollectionsWorklistCard } from "./_components/collections-worklist-card";
import { DsoTrendCard } from "./_components/dso-trend-card";
import { TopOverdueCustomersCard } from "./_components/top-overdue-customers-card";

export function AccountingDashboardPage() {
  const t = useT();

  const { allowed: canRecordPayment } = usePermission(Resource.CustomerPayment, Operation.Create);
  // The cards each draw their own zeros for a quiet book; only a tenant that
  // has never invoiced gets the sketch instead, decided from the KPI strip's
  // own query.
  const { data: kpis } = useQuery(ar.dashboardKpis());
  const neverInvoiced = kpis !== undefined && !receivablesHaveActivity(kpis.overview);

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Accounting"),
        description: t("Receivables health, collections, and cash-flow at a glance."),
        actions: canRecordPayment ? (
          <Button size="sm" render={<Link to="/accounting/ar/payments?panelType=create" />}>
            <CoinsHandIcon className="size-4" />
            {t("Record payment")}
          </Button>
        ) : undefined,
      }}
    >
      {neverInvoiced ? (
        <AccountingDashboardEmpty
          title={t("Nothing on the books yet")}
          description={
            "Receivables begin when the first invoice posts from billing. Once one does, the " +
            "balance, days sales outstanding, aging and collections figures fill in here."
          }
        />
      ) : (
        <>
          <AccountingBriefing />

          <ARKpiRow />

          <div className="grid gap-4 xl:grid-cols-2">
            <DsoTrendCard />
            <CashFlowForecastCard />
          </div>

          <div className="grid gap-4 xl:grid-cols-2">
            <AgingCard />
            <CollectionsPerformanceCard />
          </div>

          <div className="grid gap-4 xl:grid-cols-5">
            <CollectionsWorklistCard className="xl:col-span-3" />
            <TopOverdueCustomersCard className="xl:col-span-2" />
          </div>

          {/* Findings about money earned and not collected belong beside the
              worklist that collects it, not on a page of their own. */}
          <PageInsightsCard surface="Accounting" title={t("Cash insights")} />
        </>
      )}
    </PageLayout>
  );
}
