import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/custom-field-definition-table"));

export function CustomFieldDefinitionsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Custom Field Definitions"),
        description: t("Define custom fields for trailers, workers, and other resources"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
