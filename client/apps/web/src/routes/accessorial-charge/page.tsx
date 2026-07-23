import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/accessorial-charge-table"));

export function AccessorialChargesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Accessorial Charges",
        description: "Manage and configure accessorial charges for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
