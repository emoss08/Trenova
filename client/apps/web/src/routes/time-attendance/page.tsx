import { PageLayout } from "@/components/navigation/sidebar-layout";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { startOfRotaWeek } from "@trenova/shared/lib/scheduling";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy } from "react";
import {
  awaitingApprovalQuery,
  openEntriesQuery,
  thisWeekQuery,
  unpaidApprovedQuery,
} from "./_components/queries";

const TimeAttendanceConsole = lazy(() => import("./_components/time-attendance-console"));

// The overview strip and the board are what the page paints first; the clock
// and queue tabs read per-worker and per-segment lists the page cannot know
// in advance. Payroll's figure is warmed only for somebody who may run it.
export const prefetch: RoutePrefetch = () => {
  const now = Math.floor(Date.now() / 1000);
  const list: RoutePrefetchQuery[] = [
    awaitingApprovalQuery(),
    thisWeekQuery(startOfRotaWeek(now)),
    openEntriesQuery(false),
  ];
  if (usePermissionStore.getState().hasPermission(Resource.Timesheet, Operation.Export)) {
    list.push(unpaidApprovedQuery());
  }
  return list;
};

export function TimeAttendancePage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Time & Attendance",
        description:
          "Hours worked by staff paid by the clock. A week's totals are frozen when it is handed over, so what a manager approves is what payroll is run from.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <TimeAttendanceConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
