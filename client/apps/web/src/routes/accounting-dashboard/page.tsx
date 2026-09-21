import { useT } from "@trenova/shared/i18n/use-t";
import { PageInsightsCard } from "@/components/insights/page-insights";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { usePermission } from "@/hooks/use-permission";
import { AR_DASHBOARD_HISTORY_WEEKS, receivablesHaveActivity } from "@/lib/ar-dashboard";
import { queries } from "@/lib/queries";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { HandCoinsIcon } from "lucide-react";
import { Link } from "react-router";
import { AccountingDashboardEmpty } from "./_components/accounting-dashboard-empty";
import { AccountingQuickLinks } from "./_components/accounting-quick-links";
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
  // has never invoiced gets the sketch instead, so the KPI strip's own query
  // is read here alongside a year of the trend before deciding.
  const { data: kpis } = useQuery(queries.ar.dashboardKpis());
  const { data: history } = useQuery(queries.ar.dsoTrend(AR_DASHBOARD_HISTORY_WEEKS));
  const neverInvoiced =
    kpis !== undefined && history !== undefined && !receivablesHaveActivity(kpis.overview, history);

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Accounting"),
        description: t("Receivables health, collections, and cash-flow at a glance."),
        actions: canRecordPayment ? (
          <Link to="/accounting/ar/payments?panelType=create">
            <Button size="sm">
              <HandCoinsIcon className="size-4" />
              {t("Record Payment")}
            </Button>
          </Link>
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
          <ARKpiRow />

          <div className="grid gap-4 xl:grid-cols-2">
            <DsoTrendCard />
            <CashFlowForecastCard />
          </div>

          <div className="grid gap-4 xl:grid-cols-2">
            <AgingCard />
            <CollectionsPerformanceCard />
          </div>

          <div className="grid gap-4 xl:grid-cols-2">
            <TopOverdueCustomersCard />
            <CollectionsWorklistCard />
          </div>

          {/* Findings about money earned and not collected belong beside the
              worklist that collects it, not on a page of their own. */}
          <PageInsightsCard surface="Accounting" title={t("Cash insights")} />

          <AccountingQuickLinks />
        </>
      )}
    </PageLayout>
  );
}
