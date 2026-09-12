import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/account-type-table"));

export function AccountTypesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Account Types"),
        description: t("Manage and configure account types for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
