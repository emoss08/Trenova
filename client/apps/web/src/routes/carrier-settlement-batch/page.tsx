import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/batches-table"));

export function CarrierSettlementBatchesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Carrier Settlement Batches"),
        description: t("Generate pay-period AP runs, review totals, and export remittance files."),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
