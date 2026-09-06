import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Calendar = lazy(() =>
  import("./_components/holiday-calendar").then((module) => ({
    default: module.HolidayCalendar,
  })),
);

export function HolidayCalendarPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Holiday Calendar",
        description:
          "Company holidays PTO policies skip when counting days, and blackout dates no one can request off.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <Calendar />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
