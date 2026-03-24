import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/formula-template-table"));

export function FormulaTemplatesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Formula Templates",
        description: "Manage and configure formula templates for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
