import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/formula-template-table"));

export function FormulaTemplatesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Formula Templates"),
        description: t("Manage and configure formula templates for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
