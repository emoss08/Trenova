import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/fiscal-year-table"));

export function FiscalYearsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Fiscal Years",
        description: "Manage and configure fiscal years for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
