import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const RunsTable = lazy(() => import("../_components/report-runs-table"));

export function ReportRunsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Run History",
        description: "Track report generation, download artifacts, and cancel active runs",
      }}
    >
      <DataTableLazyComponent>
        <RunsTable />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
