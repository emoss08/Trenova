import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { usePermission } from "@/hooks/use-permission";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy } from "react";
import { HolidayCalendarSkeleton } from "./_components/holiday-calendar-skeleton";

const Calendar = lazy(() =>
  import("./_components/holiday-calendar").then((module) => ({
    default: module.HolidayCalendar,
  })),
);

export function HolidayCalendarPage() {
  const t = useT();
  const { allowed: canCreate } = usePermission(Resource.OrgHoliday, Operation.Create);

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Holiday calendar"),
        description: t(
          "Company holidays PTO policies skip when counting days, and blackout dates no one can request off.",
        ),
      }}
    >
      <LazyComponent fallback={<HolidayCalendarSkeleton canCreate={canCreate} />}>
        <Calendar />
      </LazyComponent>
    </PageLayout>
  );
}
