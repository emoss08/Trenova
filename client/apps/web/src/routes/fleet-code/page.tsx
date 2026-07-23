import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/fleet-code-table"));

export function FleetCodesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Fleet Codes",
        description: "Manage and configure fleet codes for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
