import { PageLayout } from "@/components/navigation/sidebar-layout";
import { yearOf } from "@/lib/osha-log";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { OshaLogSkeleton } from "./_components/osha-skeleton";
import { oshaLogQuery, oshaSummariesQuery } from "./_components/queries";

const OshaLogConsole = lazy(() => import("./_components/osha-log-console"));

// The page opens on the current year, so that year's log and every year's
// summary status are warmed; another year is left to the console.
export const prefetch: RoutePrefetch = () => {
  const list: RoutePrefetchQuery[] = [
    oshaLogQuery(yearOf(Math.floor(Date.now() / 1000))),
    oshaSummariesQuery(),
  ];
  return list;
};

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
        <DataTableLazyComponent fallback={<OshaLogSkeleton />}>
          <OshaLogConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
