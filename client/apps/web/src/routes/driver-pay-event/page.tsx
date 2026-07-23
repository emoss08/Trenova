import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/pay-events-table"));

export function DriverPayEventsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Pay Events",
        description:
          "Real-time driver earnings accrued as shipments deliver — the source ledger behind every settlement.",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
