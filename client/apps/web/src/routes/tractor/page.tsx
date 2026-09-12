import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/tractor-table"));

export function TractorsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Tractors"),
        description: t("Manage and configure tractors for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
