import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Calendar = lazy(() =>
  import("./_components/holiday-calendar").then((module) => ({
    default: module.HolidayCalendar,
  })),
);

export function HolidayCalendarPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Holiday Calendar"),
        description: t(
          "Company holidays PTO policies skip when counting days, and blackout dates no one can request off.",
        ),
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
