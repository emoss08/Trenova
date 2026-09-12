import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy } from "react";
import { MyTeamSkeleton } from "./_components/my-team-skeleton";
import { coverQuery, myTeamQuery } from "./_components/team-queries";

const MyTeamConsole = lazy(() => import("./_components/my-team-console"));

// The console opens on active people only; the "include people who have left"
// switch is a second key the page fetches on demand. Cover is warmed only when
// the user may read delegations, matching the gate the console applies.
export const prefetch: RoutePrefetch = () => {
  const list: RoutePrefetchQuery[] = [myTeamQuery(false)];
  const userId = useAuthStore.getState().user?.id;
  const canReadCover = usePermissionStore
    .getState()
    .hasPermission(Resource.ApprovalDelegation, Operation.Read);
  if (userId && canReadCover) {
    list.push(coverQuery(userId));
  }
  return list;
};

export function MyTeamPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("My Team"),
        description: t(
          "Everyone you answer for: your own reports, the terminals you run, and anybody whose approvals have been handed to you.",
        ),
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent fallback={<MyTeamSkeleton />}>
          <MyTeamConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
