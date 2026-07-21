import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@/components/ui/button";
import { usePermission } from "@/hooks/use-permission";
import { Operation, Resource } from "@/types/permission";
import { HandCoinsIcon } from "lucide-react";
import { m } from "motion/react";
import { Link } from "react-router";
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
  const { allowed: canRecordPayment } = usePermission(
    Resource.CustomerPayment,
    Operation.Create,
  );

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Accounting",
        description: "Receivables health, collections, and cash-flow at a glance.",
        actions: canRecordPayment ? (
          <Link to="/accounting/ar/payments?panelType=create">
            <Button size="sm">
              <HandCoinsIcon className="size-4" />
              Record Payment
            </Button>
          </Link>
        ) : undefined,
      }}
    >
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
    </PageLayout>
  );
}
