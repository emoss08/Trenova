import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy } from "react";
import { OrgStructureSkeleton } from "./_components/org-structure-skeleton";
import {
  delegationsGivenQuery,
  delegationsReceivedQuery,
  headcountQuery,
  jobPositionsQuery,
} from "./_components/queries";

const OrgStructureConsole = lazy(() => import("./_components/org-structure-console"));

// The headcount and the positions are the whole chart; cover is warmed only
// for somebody allowed to read delegations, matching the gate the console
// applies.
export const prefetch: RoutePrefetch = () => {
  const list: RoutePrefetchQuery[] = [headcountQuery(), jobPositionsQuery()];
  const userId = useAuthStore.getState().user?.id;
  const canReadCover = usePermissionStore
    .getState()
    .hasPermission(Resource.ApprovalDelegation, Operation.Read);
  if (userId && canReadCover) {
    list.push(delegationsGivenQuery(userId), delegationsReceivedQuery(userId));
  }
  return list;
};

export function OrgStructurePage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Org Structure"),
        description: t(
          "The shape of the organisation: the positions the roster is counted by, the headcount in each, and who is approving in whose place.",
        ),
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent fallback={<OrgStructureSkeleton />}>
          <OrgStructureConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
