import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/equipment-manufacturer-table"));

export function EquipmentManufacturersPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Equipment Manufacturers",
        description: "Manage and configure equipment manufacturers for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
