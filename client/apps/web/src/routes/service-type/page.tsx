import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/service-type-table"));

export function ServiceTypesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Service Types"),
        description: t("Manage and configure service types for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
