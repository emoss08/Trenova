import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/fiscal-year-table"));

export function FiscalYearsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Fiscal Years"),
        description: t("Manage and configure fiscal years for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
