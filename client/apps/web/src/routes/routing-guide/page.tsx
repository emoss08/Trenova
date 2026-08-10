import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/routing-guide-table"));

export function RoutingGuidesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Routing Guides",
        description:
          "Ranked carrier waterfalls that cover a lane automatically when a move tenders",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
