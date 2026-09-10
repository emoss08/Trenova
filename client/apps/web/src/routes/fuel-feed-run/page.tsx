import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { SyncFeedButton } from "./_components/sync-feed-button";

const Table = lazy(() => import("./_components/feed-run-table"));

export function FuelFeedRunsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Feed Runs",
        description:
          "Every time a connected fuel card feed has read your transactions, and what became of the rows. Rows a run could not place wait here until the card they were on is assigned or the unit they name is added.",
        actions: <SyncFeedButton />,
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
