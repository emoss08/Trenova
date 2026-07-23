import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/batches-table"));

export function SettlementBatchesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Settlement Batches",
        description: "Generate pay-period batches, monitor exceptions, and export payroll files.",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
