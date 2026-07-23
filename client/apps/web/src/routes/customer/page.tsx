import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/customer-table"));

export function CustomersPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Customers",
        description: "Manage and configure customers for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
