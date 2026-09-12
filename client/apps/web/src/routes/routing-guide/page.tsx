import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/routing-guide-table"));

export function RoutingGuidesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Routing Guides"),
        description: t(
          "Ranked carrier waterfalls that cover a lane automatically when a move tenders",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
