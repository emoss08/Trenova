import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/cost-events-table"));

export function CarrierCostEventsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Carrier Cost Events",
        description:
          "Purchased-transportation cost accrued as carrier-covered shipments deliver — the source ledger behind every carrier settlement.",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
