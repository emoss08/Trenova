import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/customer-table"));

export function CustomersPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Customers"),
        description: t("Manage and configure customers for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
