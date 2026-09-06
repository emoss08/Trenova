import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const MyTeamConsole = lazy(() => import("./_components/my-team-console"));

export function MyTeamPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "My Team",
        description:
          "Everyone you answer for: your own reports, the terminals you run, and anybody whose approvals have been handed to you.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <MyTeamConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
