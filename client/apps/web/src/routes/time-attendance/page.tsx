import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const TimeAttendanceConsole = lazy(() => import("./_components/time-attendance-console"));

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
