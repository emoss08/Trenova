import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { startOfRotaWeek } from "@trenova/shared/lib/scheduling";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy } from "react";
import { openSwapsQuery, rotaQuery, shiftTemplatesQuery } from "./_components/queries";
import { SchedulingSkeleton } from "./_components/scheduling-skeleton";

const SchedulingConsole = lazy(() => import("./_components/scheduling-console"));

// The page opens on this week, unfiltered, one week wide; that is the rota it
// paints first. Patterns and open swaps feed the tab badges and the overview,
// so they are warmed alongside for anybody allowed to read them.
export const prefetch: RoutePrefetch = () => {
  const hasPermission = usePermissionStore.getState().hasPermission;
  const list: RoutePrefetchQuery[] = [];
  if (hasPermission(Resource.WorkerSchedule, Operation.Read)) {
    list.push(
      rotaQuery({
        weekStart: startOfRotaWeek(Math.floor(Date.now() / 1000)),
        weeks: 1,
        teamOnly: false,
        fleetCodeId: null,
      }),
    );
  }
  if (hasPermission(Resource.ShiftTemplate, Operation.Read)) {
    list.push(shiftTemplatesQuery());
  }
  if (hasPermission(Resource.ShiftSwap, Operation.Read)) {
    list.push(openSwapsQuery());
  }
  return list;
};

export function SchedulingPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Scheduling"),
        description: t(
          "Who is expected on which days. The board is composed on every read from the pattern, approved time off, open leave and the work dispatch has already assigned — so it cannot go stale.",
        ),
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent fallback={<SchedulingSkeleton />}>
          <SchedulingConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
