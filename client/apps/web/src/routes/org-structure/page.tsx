import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const OrgStructureConsole = lazy(() => import("./_components/org-structure-console"));

export function OrgStructurePage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Org Structure",
        description:
          "The shape of the organisation: the positions the roster is counted by, the headcount in each, and who is approving in whose place.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <OrgStructureConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
