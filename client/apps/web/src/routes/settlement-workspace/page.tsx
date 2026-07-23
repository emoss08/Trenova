import { PageLayout } from "@/components/navigation/sidebar-layout";
import { LazyComponent } from "@/components/error-boundary";
import { lazy } from "react";

const Workspace = lazy(() => import("./_components/workspace"));

export function SettlementWorkspacePage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Settlement Workspace",
        description:
          "Everything needed to run a pay period from one screen — review the queue, transfer pay, manage deductions, and process settlements one at a time or in bulk.",
      }}
    >
      <LazyComponent>
        <Workspace />
      </LazyComponent>
    </PageLayout>
  );
}
