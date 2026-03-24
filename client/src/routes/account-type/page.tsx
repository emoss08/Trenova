import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/account-type-table"));

export function AccountTypesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Account Types",
        description: "Manage and configure account types for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
