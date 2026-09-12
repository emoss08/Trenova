import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const RunsTable = lazy(() => import("../_components/report-runs-table"));

export function ReportRunsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Run History"),
        description: t("Track report generation, download artifacts, and cancel active runs"),
      }}
    >
      <DataTableLazyComponent>
        <RunsTable />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
