import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const OshaLogConsole = lazy(() => import("./_components/osha-log-console"));

export function OshaLogPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "OSHA 300 Log",
        description:
          "Recordable injuries and illnesses for the year, and the 300A summary posted over them. The totals are counted from the log every time it is read, so a case corrected years later cannot leave a stale summary behind.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <OshaLogConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
