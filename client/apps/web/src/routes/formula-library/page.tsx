import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";

const Table = lazy(() => import("./_components/formula-library-table"));

export function FormulaLibrariesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Formula libraries"),
        description: t(
          "Functions, constants and types written once in Tariff and imported by your formula templates",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
