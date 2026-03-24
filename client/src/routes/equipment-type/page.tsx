import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/equipment-type-table"));

export function EquipmentTypesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Equipment Types",
        description: "Manage and configure equipment types for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
