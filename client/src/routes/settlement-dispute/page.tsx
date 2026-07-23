import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/disputes-table"));

export function SettlementDisputesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Settlement Disputes",
        description:
          "Driver-submitted questions and challenges against issued settlements — review, resolve with a correcting adjustment, or deny with an explanation.",
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
