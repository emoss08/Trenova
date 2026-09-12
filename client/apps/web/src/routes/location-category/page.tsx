import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/location-category-table"));

export function LocationCategoriesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Location Categories"),
        description: t("Manage and configure location categories for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
