import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/commodity-table"));

export function CommoditiesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Commodities"),
        description: t("Manage and configure commodities for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
