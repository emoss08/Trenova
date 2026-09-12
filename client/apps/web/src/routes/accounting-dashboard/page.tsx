import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { usePermission } from "@/hooks/use-permission";
import { AR_DASHBOARD_HISTORY_WEEKS, receivablesHaveActivity } from "@/lib/ar-dashboard";
import { queries } from "@/lib/queries";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { HandCoinsIcon } from "lucide-react";
import { m } from "motion/react";
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

function Section({
  index,
  children,
  className,
}: {
  index: number;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <m.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, delay: 0.05 + index * 0.07, ease: "easeOut" }}
      className={className}
    >
      {children}
    </m.div>
  );
}

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
        title: "Accounting",
        description: "Receivables health, collections, and cash-flow at a glance.",
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
        <div className="mx-4 mt-3 mb-4">
          <AccountingDashboardEmpty
            title={t("Nothing on the books yet")}
            description={
              "Receivables begin when the first invoice posts from billing. Once one does, the " +
              "balance, days sales outstanding, aging and collections figures fill in here."
            }
          />
        </div>
      ) : (
        <div className="mx-4 mt-3 mb-4 space-y-4">
          <ARKpiRow />

          <Section index={1} className="grid gap-4 xl:grid-cols-2">
            <DsoTrendCard />
            <CashFlowForecastCard />
          </Section>

          <Section index={2} className="grid gap-4 xl:grid-cols-2">
            <AgingCard />
            <CollectionsPerformanceCard />
          </Section>

          <Section index={3} className="grid gap-4 xl:grid-cols-2">
            <TopOverdueCustomersCard />
            <CollectionsWorklistCard />
          </Section>

          <Section index={4}>
            <AccountingQuickLinks />
          </Section>
        </div>
      )}
    </PageLayout>
  );
}
