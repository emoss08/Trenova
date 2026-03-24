import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/document-type-table"));

export function DocumentTypesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Document Types",
        description: "Manage and configure document types for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
