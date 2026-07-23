import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/hazardous-material-table"));

export function HazardousMaterialsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Hazardous Materials",
        description: "Manage and configure hazardous materials for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
