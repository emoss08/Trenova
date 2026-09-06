import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const SchedulingConsole = lazy(() => import("./_components/scheduling-console"));

export function SchedulingPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Scheduling",
        description:
          "Who is expected on which days. The board is composed on every read from the pattern, approved time off, open leave and the work dispatch has already assigned — so it cannot go stale.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <SchedulingConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
