import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/batches-table"));

export function CarrierSettlementBatchesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Carrier Settlement Batches",
        description: "Generate pay-period AP runs, review totals, and export remittance files.",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
