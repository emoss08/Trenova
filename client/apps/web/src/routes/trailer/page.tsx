import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/trailer-table"));

export function TrailersPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Trailers",
        description: "Manage and configure trailers for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
