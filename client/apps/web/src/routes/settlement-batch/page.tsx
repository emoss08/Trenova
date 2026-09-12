import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/batches-table"));

export function SettlementBatchesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Settlement Batches"),
        description: t(
          "Generate pay-period batches, monitor exceptions, and export payroll files.",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
