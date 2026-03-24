import { DataTableLazyComponent } from "@/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/hazmat-segregation-rule-table"));

export function HazmatSegregationRulesPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Hazmat Segregation Rules"
        description="Manage and configure hazmat segregation rules for your organization"
      />
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </AdminPageLayout>
  );
}
