import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/shipment-type-table"));

export function ShipmentTypesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Shipment Types"),
        description: t("Manage and configure shipment types for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
