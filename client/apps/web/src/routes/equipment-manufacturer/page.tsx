import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/equipment-manufacturer-table"));

export function EquipmentManufacturersPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Equipment Manufacturers"),
        description: t("Manage and configure equipment manufacturers for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
